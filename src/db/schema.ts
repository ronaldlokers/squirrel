import { sql } from "drizzle-orm";
import {
  bigint,
  bigserial,
  index,
  jsonb,
  pgTable,
  text,
  timestamp,
  uniqueIndex,
} from "drizzle-orm/pg-core";

/**
 * A person is a row because a transport id is a transport-local coordinate:
 * "user 1, in Campfire" says nothing about the same human on Matrix.
 */
export const people = pgTable("people", {
  id: bigserial("id", { mode: "number" }).primaryKey(),
  handle: text("handle").notNull().unique(),
  createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
});

export const identities = pgTable(
  "identities",
  {
    id: bigserial("id", { mode: "number" }).primaryKey(),
    personId: bigint("person_id", { mode: "number" })
      .notNull()
      .references(() => people.id),
    transport: text("transport").notNull(),
    externalId: text("external_id").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => ({
    unique: uniqueIndex("identities_transport_external_id_key").on(
      table.transport,
      table.externalId,
    ),
  }),
);

export const items = pgTable(
  "items",
  {
    id: bigserial("id", { mode: "number" }).primaryKey(),
    transport: text("transport").notNull(),
    // Identifiers are text, not bigint: Matrix room ids are strings and
    // Telegram chat ids are signed 64-bit. Nothing arithmetic is done with them.
    externalId: text("external_id"),
    conversationId: text("conversation_id"),
    // Kept alongside personId on purpose. personId is an interpretation and can
    // be absent or wrong; senderId is what the transport actually said.
    senderId: text("sender_id"),
    personId: bigint("person_id", { mode: "number" }).references(() => people.id),
    rawText: text("raw_text").notNull(),
    payload: jsonb("payload").notNull(),
    receivedAt: timestamp("received_at", { withTimezone: true }).notNull(),
    insertedAt: timestamp("inserted_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => ({
    // Partial, because the fail-open path has no external id to be unique on.
    external: uniqueIndex("items_transport_external_id_key")
      .on(table.transport, table.externalId)
      .where(sql`${table.externalId} is not null`),
    received: index("items_received_at_idx").on(table.receivedAt),
  }),
);
