import { describe, expect, it } from "vitest";

import { createHttp, startHttp } from "../../src/http.js";

const writableSpool = { writable: () => Promise.resolve(true) };
const brokenSpool = { writable: () => Promise.resolve(false) };

describe("createHttp", () => {
  it("reports health when the spool is writable", async () => {
    const { app } = createHttp(writableSpool);
    const response = await app.request("/healthz");
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("ok");
  });

  // Deliberately not a Postgres check. An unready pod stops receiving
  // webhooks, and Campfire never retries, so that would turn a survivable
  // outage into permanent loss.
  it("reports unhealthy only when the spool is unwritable", async () => {
    const { app } = createHttp(brokenSpool);
    expect((await app.request("/healthz")).status).toBe(503);
  });

  it("404s an unknown path", async () => {
    const { app } = createHttp(writableSpool);
    expect((await app.request("/nope")).status).toBe(404);
  });

  it("lets a transport mount a POST route", async () => {
    const { app, mount } = createHttp(writableSpool);
    mount.post("/transports/fake", async (request) => new Response(await request.text()));

    const response = await app.request("/transports/fake", { method: "POST", body: "hello" });
    expect(await response.text()).toBe("hello");
  });

  it("listens on an ephemeral port and closes cleanly", async () => {
    const { app } = createHttp(writableSpool);
    const server = await startHttp(app, 0);
    expect(server.port).toBeGreaterThan(0);

    const response = await fetch(`http://127.0.0.1:${server.port}/healthz`);
    expect(await response.text()).toBe("ok");
    await server.close();
  });

  // The defect this exists to catch: a convenience HTTP layer stamping a
  // default content-type on a response that deliberately has none, which
  // would turn Campfire's one silent response into a posted message. Unit
  // tests that never cross a socket (like the ones above, via `app.request`)
  // cannot see this — it only shows up on a real response.
  it("carries no content-type over the wire when the handler sets none", async () => {
    const { app, mount } = createHttp(writableSpool);
    mount.post("/silent", () => Promise.resolve(new Response(null, { status: 200 })));

    const server = await startHttp(app, 0);
    try {
      const response = await fetch(`http://127.0.0.1:${server.port}/silent`, { method: "POST" });
      expect(response.status).toBe(200);
      expect(response.headers.get("content-type")).toBeNull();
      expect(await response.text()).toBe("");
    } finally {
      await server.close();
    }
  });

  // The defect this exists to catch: Hono's default 404 (and its default
  // onError 500) carry a `Content-Type`, so on the Campfire route an
  // unrouted or throwing request would be uploaded into the room as a file
  // attachment instead of staying silent — and the message that triggered it
  // is lost with no retry. `app.request(...)` never crosses a socket, so it
  // cannot see this; it has to be asserted on a real response.
  it("carries no content-type on a 404 over the wire", async () => {
    const { app } = createHttp(writableSpool);

    const server = await startHttp(app, 0);
    try {
      const response = await fetch(`http://127.0.0.1:${server.port}/transports/campfire/`);
      expect(response.status).toBe(404);
      expect(response.headers.get("content-type")).toBeNull();
      expect(await response.text()).toBe("");
    } finally {
      await server.close();
    }
  });

  it("rejects rather than hanging when the port is already in use", async () => {
    const { app: firstApp } = createHttp(writableSpool);
    const first = await startHttp(firstApp, 0);

    const { app: secondApp } = createHttp(writableSpool);
    const attempt = startHttp(secondApp, first.port);

    try {
      await expect(attempt).rejects.toThrow();
    } finally {
      await first.close();
      // If the assertion above ever fails because startHttp regresses to
      // resolving anyway, don't leave a second listening socket behind.
      await attempt.then((server) => server.close()).catch(() => undefined);
    }
  }, 2000);
});
