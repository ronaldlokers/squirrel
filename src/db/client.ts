import { type NodePgDatabase, drizzle } from "drizzle-orm/node-postgres";
import pg from "pg";

import type { PostgresConfig } from "../config.js";
import * as schema from "./schema.js";

export type Db = NodePgDatabase<typeof schema>;

export interface DbHandle {
  readonly db: Db;
  close(): Promise<void>;
}

export function urlFor(config: PostgresConfig): string {
  const user = encodeURIComponent(config.user);
  const password = encodeURIComponent(config.password);
  return `postgres://${user}:${password}@${config.host}:${config.port}/${config.database}`;
}

export function openDb(config: PostgresConfig): DbHandle {
  return fromUrl(urlFor(config));
}

export function fromUrl(url: string): DbHandle {
  const pool = new pg.Pool({ connectionString: url, max: 4 });
  return { db: drizzle(pool, { schema }), close: () => pool.end() };
}
