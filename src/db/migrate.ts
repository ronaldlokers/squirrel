import { migrate } from "drizzle-orm/node-postgres/migrator";

import type { Db } from "./client.js";

export function applyMigrations(db: Db): Promise<void> {
  return migrate(db, { migrationsFolder: "drizzle" });
}
