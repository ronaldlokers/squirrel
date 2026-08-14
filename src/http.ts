import { type IncomingMessage, type ServerResponse, createServer } from "node:http";
import type { AddressInfo } from "node:net";

import { Hono } from "hono";

import type { Spool } from "./capture/spool.js";
import type { HttpMount, WebHandler } from "./transports/transport.js";

export interface Serving {
  readonly port: number;
  readonly close: () => Promise<void>;
}

/**
 * One server for the whole process: `/healthz`, plus whatever routes the
 * transports mount. A transport that polls simply never calls `mount.post`.
 */
export function createHttp(spool: Pick<Spool, "writable">): { app: Hono; mount: HttpMount } {
  const app = new Hono();

  // Liveness and readiness both, and deliberately not a database check. A
  // readiness probe that failed on a Postgres outage would pull this pod out
  // of its Service; webhook delivery would then fail, and Campfire does not
  // retry. That converts a survivable outage into permanent data loss.
  app.get("/healthz", async (context) =>
    (await spool.writable()) ? context.text("ok") : context.text("spool unwritable", 503),
  );

  const mount: HttpMount = {
    post(path: string, handler: WebHandler) {
      app.post(path, (context) => handler(context.req.raw));
    },
  };

  // Hono's default 404 and 500 both carry a `Content-Type`, which is the same
  // defect that got `@hono/node-server` removed: on the Campfire route, a
  // non-200 (or any 404) with a content type is uploaded into the room as a
  // file attachment, and the message that triggered it is lost with no
  // retry. Bare responses here keep every unrouted or throwing request
  // genuinely silent on the wire.
  app.notFound(() => new Response(null, { status: 404 }));
  app.onError(() => new Response(null, { status: 500 }));

  return { app, mount };
}

/** GET and HEAD carry no body; every other method used here does. */
function hasBody(method: string): boolean {
  return method !== "GET" && method !== "HEAD";
}

function readBody(incoming: IncomingMessage): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    incoming.on("data", (chunk: Buffer) => chunks.push(chunk));
    incoming.on("end", () => resolve(Buffer.concat(chunks)));
    incoming.on("error", reject);
  });
}

/**
 * Node's request, translated into the web `Request` the router expects. The
 * origin is a fixed placeholder — nothing downstream ever looks at it, only
 * the path and query drive routing — and the body is buffered rather than
 * streamed, which is fine at the sizes a webhook payload comes in.
 */
async function requestFrom(incoming: IncomingMessage): Promise<Request> {
  const method = incoming.method ?? "GET";
  const url = new URL(incoming.url ?? "/", "http://localhost");

  const headers = new Headers();
  for (const [name, value] of Object.entries(incoming.headers)) {
    if (value === undefined) continue;
    for (const one of Array.isArray(value) ? value : [value]) headers.append(name, one);
  }

  const init: RequestInit = { method, headers };
  if (hasBody(method)) init.body = await readBody(incoming);
  return new Request(url, init);
}

/**
 * Writes back exactly what the app produced: this status, these headers, and
 * no others. Campfire's whole contract — stored, ignored, failed — is carried
 * entirely in whether a content type is present, so nothing here may add one
 * the handler did not set itself. That is precisely the job a convenience
 * layer cannot be trusted with: `@hono/node-server` stamps a default
 * `content-type` on any response that omits one, which would turn Squirrel's
 * one deliberately silent response (`respond("ignored")`) into a `text/plain`
 * message posted into the room. Hand-rolled here so the bytes on the wire are
 * ours to control, not a library's to rewrite.
 */
async function writeResponse(response: Response, outgoing: ServerResponse): Promise<void> {
  const headers: Record<string, string> = {};
  for (const [name, value] of response.headers) headers[name] = value;

  outgoing.writeHead(response.status, headers);
  outgoing.end(Buffer.from(await response.arrayBuffer()));
}

export function startHttp(app: Hono, port: number): Promise<Serving> {
  return new Promise((resolve, reject) => {
    const server = createServer((incoming, outgoing) => {
      void requestFrom(incoming)
        .then((request) => app.fetch(request))
        .then((response) => writeResponse(response, outgoing))
        .catch((error: unknown) => {
          if (!outgoing.headersSent) outgoing.writeHead(500);
          outgoing.end();
          process.stderr.write(`http: request failed: ${String(error)}\n`);
        });
    });

    server.listen(port, () => {
      // Listening succeeded: a later error (e.g. a client reset) is no
      // longer a startup failure, so stop treating it as one. But an
      // EventEmitter with zero `error` listeners throws on the next `error`
      // event instead of just emitting it, which would kill the whole
      // process — and every in-flight webhook with it. Swap in a listener
      // that logs rather than leaving the server bare.
      server.off("error", reject);
      server.on("error", (error) => {
        process.stderr.write(`http: server error: ${String(error)}\n`);
      });
      const address = server.address() as AddressInfo;
      resolve({
        port: address.port,
        close: () =>
          new Promise((closed) => {
            // `server.close()` stops accepting new connections and waits for
            // in-flight requests, but a keep-alive connection sitting idle
            // between requests is neither — it would hold the server open
            // indefinitely. Idle ones only; a request actually in flight is
            // left alone to finish.
            server.closeIdleConnections?.();
            server.close(() => closed());
          }),
      });
    });

    // EADDRINUSE or EACCES fire here, not through the listening callback.
    // Without this, a failed bind never settles the promise and the caller
    // hangs forever instead of getting an actionable startup error.
    server.once("error", reject);
  });
}
