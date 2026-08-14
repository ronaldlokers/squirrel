import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import { createDrain } from "../../src/capture/drain.js";
import { type Spool, openSpool } from "../../src/capture/spool.js";
import { type Db, fromUrl } from "../../src/db/client.js";
import { items } from "../../src/db/schema.js";
import { seedOwner } from "../../src/people/identities.js";
import { truncateAll, withDb } from "./support.js";

let db: Db;
let close: () => Promise<void>;
let spool: Spool;
let dir: string;

beforeAll(async () => {
  ({ db, close } = await withDb());
});

beforeEach(async () => {
  await truncateAll(db);
  dir = await mkdtemp(join(tmpdir(), "squirrel-drain-"));
  spool = await openSpool(dir);
});

afterAll(async () => {
  await close();
});

function capture(overrides: Partial<Capture> = {}): Capture {
  return {
    transport: "campfire",
    externalId: "42",
    conversationId: "7",
    senderId: "1",
    text: "buy milk",
    receivedAt: new Date("2026-08-14T09:31:04.512Z"),
    payload: { message: { id: 42 } },
    ...overrides,
  };
}

describe("drain", () => {
  it("inserts a spooled capture and deletes the file", async () => {
    await spool.write(capture());
    const drain = createDrain({ spool, db, intervalMs: 1000 });

    expect(await drain.drainOnce()).toEqual({ inserted: 1, quarantined: 0, deferred: 0 });

    const rows = await db.select().from(items);
    expect(rows).toMatchObject([
      { transport: "campfire", externalId: "42", conversationId: "7", rawText: "buy milk" },
    ]);
    expect(await spool.list()).toEqual([]);
  });

  it("resolves a seeded identity to a person", async () => {
    const personId = await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    await spool.write(capture());
    await createDrain({ spool, db, intervalMs: 1000 }).drainOnce();

    expect((await db.select().from(items))[0]?.personId).toBe(personId);
  });

  // A capture is never held hostage to knowing whose it was.
  it("stores a capture from an unknown identity with a null person", async () => {
    const onUnknownIdentity = vi.fn();
    await spool.write(capture({ senderId: "999" }));
    const drain = createDrain({ spool, db, intervalMs: 1000, onUnknownIdentity });

    expect(await drain.drainOnce()).toMatchObject({ inserted: 1 });
    expect((await db.select().from(items))[0]?.personId).toBeNull();
    expect(await spool.list()).toEqual([]);
    expect(onUnknownIdentity).toHaveBeenCalledWith("campfire", "999");
  });

  it("inserts one row for a redelivered message", async () => {
    const drain = createDrain({ spool, db, intervalMs: 1000 });
    await spool.write(capture());
    await drain.drainOnce();
    await spool.write(capture());
    await drain.drainOnce();

    expect(await db.select().from(items)).toHaveLength(1);
  });

  it("keeps the same external id under two transports apart", async () => {
    await spool.write(capture());
    await spool.write(capture({ transport: "matrix" }));
    await createDrain({ spool, db, intervalMs: 1000 }).drainOnce();

    expect(await db.select().from(items)).toHaveLength(2);
  });

  it("defers everything and loses nothing while the database is unreachable", async () => {
    const unreachable = fromUrl("postgres://nobody:nobody@127.0.0.1:1/squirrel");
    await spool.write(capture());
    const drain = createDrain({ spool, db: unreachable.db, intervalMs: 1000 });

    expect(await drain.drainOnce()).toMatchObject({ inserted: 0, deferred: 1 });
    expect(await spool.list()).toHaveLength(1);
    await unreachable.close();

    // And it lands once the database comes back.
    expect(await createDrain({ spool, db, intervalMs: 1000 }).drainOnce()).toMatchObject({
      inserted: 1,
    });
  });

  it("quarantines an unreadable file rather than retrying it forever", async () => {
    await writeFile(join(dir, "000000000000001-campfire-9.json"), "{ not json");
    const drain = createDrain({ spool, db, intervalMs: 1000 });

    expect(await drain.drainOnce()).toMatchObject({ inserted: 0, quarantined: 1 });
    expect(await spool.list()).toEqual([]);
    expect(await drain.drainOnce()).toMatchObject({ quarantined: 0 });
  });

  it("drains on an interval once started, and stops cleanly", async () => {
    await spool.write(capture());
    const drain = createDrain({ spool, db, intervalMs: 10 });
    drain.start();

    await vi.waitFor(async () => expect(await db.select().from(items)).toHaveLength(1));
    await drain.stop();
  });
});
