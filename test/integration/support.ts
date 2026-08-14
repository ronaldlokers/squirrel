import { sql } from "drizzle-orm";

import { type Db, fromUrl } from "../../src/db/client.js";
import { applyMigrations } from "../../src/db/migrate.js";

/** Refuses rather than skips, so an unset variable in CI fails loudly. */
export function testDatabaseUrl(): string {
  const url = process.env.TEST_DATABASE_URL;
  if (!url) throw new Error("TEST_DATABASE_URL is required — see docs/testing.md");
  return url;
}

export async function withDb(): Promise<{ db: Db; close: () => Promise<void> }> {
  const handle = fromUrl(testDatabaseUrl());
  await applyMigrations(handle.db);
  await truncateAll(handle.db);
  return handle;
}

export async function truncateAll(db: Db): Promise<void> {
  await db.execute(sql`truncate table items, identities, people restart identity cascade`);
}
