import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { Db } from "../../src/db/client.js";
import { items } from "../../src/db/schema.js";
import { type Squirrel, boot } from "../../src/index.js";
import { truncateAll, withDb } from "./support.js";

const PAYLOAD = {
  user: { id: 1, name: "Ronald" },
  room: { id: 7, name: "Squirrel", path: "/rooms/7/3-abc/messages" },
  message: {
    id: 42,
    body: { html: "<div>buy milk</div>", plain: "buy milk" },
    path: "/rooms/7/@42",
  },
};

let db: Db;
let close: () => Promise<void>;
let squirrel: Squirrel;
let spoolDir: string;

function envFor(url: string, overrides: Record<string, string> = {}) {
  const parsed = new URL(url);
  return {
    PORT: "0",
    SPOOL_DIR: spoolDir,
    DRAIN_INTERVAL_MS: "10",
    OWNER_HANDLE: "ronald",
    CAMPFIRE_CONVERSATION_ID: "7",
    CAMPFIRE_SENDER_ID: "1",
    POSTGRES_SERVER: parsed.hostname,
    POSTGRES_PORT: parsed.port,
    POSTGRES_DB: parsed.pathname.slice(1),
    POSTGRES_USER: decodeURIComponent(parsed.username),
    POSTGRES_PASSWORD: decodeURIComponent(parsed.password),
    ...overrides,
  };
}

async function post(body: string): Promise<Response> {
  return fetch(`http://127.0.0.1:${squirrel.port}/transports/campfire`, { method: "POST", body });
}

beforeAll(async () => {
  ({ db, close } = await withDb());
});

beforeEach(async () => {
  await truncateAll(db);
  spoolDir = await mkdtemp(join(tmpdir(), "squirrel-boot-"));
});

afterEach(async () => {
  await squirrel?.stop();
  await rm(spoolDir, { recursive: true, force: true });
});

afterAll(async () => {
  await close();
});

describe("boot", () => {
  it("captures a message end to end and answers with a squirrel", async () => {
    squirrel = await boot(envFor(process.env.TEST_DATABASE_URL as string));

    const response = await post(JSON.stringify(PAYLOAD));
    expect(await response.text()).toBe("🐿️");

    await vi.waitFor(async () => {
      const rows = await db.select().from(items);
      expect(rows).toMatchObject([{ rawText: "buy milk", senderId: "1" }]);
      expect(rows[0]?.personId).not.toBeNull();
    });
  });

  it("says nothing at all to a message from another room", async () => {
    squirrel = await boot(envFor(process.env.TEST_DATABASE_URL as string));

    const elsewhere = { ...PAYLOAD, room: { ...PAYLOAD.room, id: 8 } };
    const response = await post(JSON.stringify(elsewhere));

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBeNull();
    expect(await response.text()).toBe("");
  });

  it("is healthy", async () => {
    squirrel = await boot(envFor(process.env.TEST_DATABASE_URL as string));
    const response = await fetch(`http://127.0.0.1:${squirrel.port}/healthz`);
    expect(await response.text()).toBe("ok");
  });

  // The whole point of spooling. An unreachable database at boot must not stop
  // a single capture from being accepted, because Campfire will not retry it.
  it("serves, captures and stays healthy with the database unreachable", async () => {
    squirrel = await boot(
      envFor(process.env.TEST_DATABASE_URL as string, {
        POSTGRES_PORT: "1",
        POSTGRES_SERVER: "127.0.0.1",
      }),
    );

    const response = await post(JSON.stringify(PAYLOAD));
    expect(await response.text()).toBe("🐿️");
    expect(await (await fetch(`http://127.0.0.1:${squirrel.port}/healthz`)).text()).toBe("ok");
  });
});
