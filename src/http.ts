import type { AddressInfo } from "node:net";

import { serve } from "@hono/node-server";
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

  return { app, mount };
}

export function startHttp(app: Hono, port: number): Promise<Serving> {
  return new Promise((resolve) => {
    const server = serve({ fetch: app.fetch, port }, (address: AddressInfo) => {
      resolve({
        port: address.port,
        close: () => new Promise((closed) => server.close(() => closed())),
      });
    });
  });
}
