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
