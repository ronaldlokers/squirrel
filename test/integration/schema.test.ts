import { sql } from "drizzle-orm";
import { afterAll, beforeAll, beforeEach, describe, expect, it } from "vitest";

import type { Db } from "../../src/db/client.js";
import { items, people } from "../../src/db/schema.js";
import { truncateAll, withDb } from "./support.js";

let db: Db;
let close: () => Promise<void>;

beforeAll(async () => {
  ({ db, close } = await withDb());
});

afterAll(async () => {
  await close();
});

beforeEach(async () => {
  await truncateAll(db);
});

describe("schema", () => {
  it("applies migrations and creates the three tables", async () => {
    const result = await db.execute(
      sql`select table_name from information_schema.tables where table_schema = 'public' order by table_name`,
    );
    const names = result.rows.map((row) => row.table_name);
    expect(names).toContain("people");
    expect(names).toContain("identities");
    expect(names).toContain("items");
  });

  it("stores an item with nulls in every fail-open column", async () => {
    await db.insert(items).values({
      transport: "campfire",
      externalId: null,
      conversationId: null,
      senderId: null,
      personId: null,
      rawText: "",
      payload: { unparseable: "nonsense" },
      receivedAt: new Date("2026-08-14T09:31:04.512Z"),
    });
    expect(await db.select().from(items)).toHaveLength(1);
  });

  it("allows many items with a null external id, since the index is partial", async () => {
    const row = {
      transport: "campfire",
      externalId: null,
      rawText: "",
      payload: {},
      receivedAt: new Date(),
    };
    await db.insert(items).values([row, row]);
    expect(await db.select().from(items)).toHaveLength(2);
  });

  it("rejects a duplicate handle", async () => {
    await db.insert(people).values({ handle: "owner" });
    await expect(db.insert(people).values({ handle: "owner" })).rejects.toThrow();
  });
});
