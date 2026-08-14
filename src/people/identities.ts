import { and, eq } from "drizzle-orm";

import type { Db } from "../db/client.js";
import { identities, people } from "../db/schema.js";

export interface IdentitySeed {
  readonly transport: string;
  readonly externalId: string;
}

/**
 * Seeding is declarative rather than administrative: the desired state lives in
 * configuration, in Git, and every boot reconciles to it. There is no admin
 * screen to forget about.
 */
export async function seedOwner(
  db: Db,
  handle: string,
  seeds: readonly IdentitySeed[],
): Promise<number> {
  return db.transaction(async (tx) => {
    const inserted = await tx
      .insert(people)
      .values({ handle })
      .onConflictDoNothing({ target: people.handle })
      .returning({ id: people.id });

    const personId =
      inserted[0]?.id ??
      (await tx.select({ id: people.id }).from(people).where(eq(people.handle, handle)))[0]?.id;

    if (personId === undefined) throw new Error(`could not seed person ${handle}`);

    for (const seed of seeds) {
      await tx
        .insert(identities)
        .values({ personId, transport: seed.transport, externalId: seed.externalId })
        .onConflictDoNothing({ target: [identities.transport, identities.externalId] });
    }

    return personId;
  });
}

/**
 * Resolution happens in the drain loop, never on the request path — a person
 * lookup is a database read, and the request path not touching Postgres is what
 * makes an outage survivable.
 *
 * An unknown identity is null. It is never created: auto-vivifying a person on
 * first sight would quietly re-admit anyone the guard had just turned away.
 */
export async function resolvePerson(
  db: Db,
  transport: string,
  externalId: string | null,
): Promise<number | null> {
  if (externalId === null) return null;

  const rows = await db
    .select({ personId: identities.personId })
    .from(identities)
    .where(and(eq(identities.transport, transport), eq(identities.externalId, externalId)))
    .limit(1);

  return rows[0]?.personId ?? null;
}
