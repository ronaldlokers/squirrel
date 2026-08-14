import { afterAll, beforeAll, beforeEach, describe, expect, it } from "vitest";

import type { Db } from "../../src/db/client.js";
import { identities, people } from "../../src/db/schema.js";
import { resolvePerson, seedOwner } from "../../src/people/identities.js";
import { truncateAll, withDb } from "./support.js";

let db: Db;
let close: () => Promise<void>;

beforeAll(async () => {
  ({ db, close } = await withDb());
});

beforeEach(async () => {
  await truncateAll(db);
});

afterAll(async () => {
  await close();
});

describe("seedOwner", () => {
  it("creates the person and their identities", async () => {
    const personId = await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);

    expect(await db.select().from(people)).toHaveLength(1);
    expect(await db.select().from(identities)).toMatchObject([
      { personId, transport: "campfire", externalId: "1" },
    ]);
  });

  it("is idempotent, so every boot may run it", async () => {
    const seeds = [{ transport: "campfire", externalId: "1" }];
    const first = await seedOwner(db, "ronald", seeds);
    const second = await seedOwner(db, "ronald", seeds);

    expect(second).toBe(first);
    expect(await db.select().from(people)).toHaveLength(1);
    expect(await db.select().from(identities)).toHaveLength(1);
  });

  it("adds a second transport's identity to the same person", async () => {
    await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    const personId = await seedOwner(db, "ronald", [
      { transport: "campfire", externalId: "1" },
      { transport: "matrix", externalId: "@me:example" },
    ]);

    const rows = await db.select().from(identities);
    expect(rows).toHaveLength(2);
    expect(rows.every((row) => row.personId === personId)).toBe(true);
  });
});

describe("resolvePerson", () => {
  it("resolves a seeded identity", async () => {
    const personId = await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    expect(await resolvePerson(db, "campfire", "1")).toBe(personId);
  });

  it("returns null for an unknown identity", async () => {
    await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    expect(await resolvePerson(db, "campfire", "999")).toBeNull();
  });

  it("returns null for a null external id, which is the fail-open case", async () => {
    expect(await resolvePerson(db, "campfire", null)).toBeNull();
  });

  it("does not match the same external id under another transport", async () => {
    await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    expect(await resolvePerson(db, "matrix", "1")).toBeNull();
  });

  // The test that keeps the guard from being decorative: auto-creating a
  // person on first sight would re-admit whoever the allowlist turned away.
  it("never creates a person for an unknown identity", async () => {
    await resolvePerson(db, "campfire", "999");
    expect(await db.select().from(people)).toEqual([]);
    expect(await db.select().from(identities)).toEqual([]);
  });
});
