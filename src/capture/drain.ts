import { sql } from "drizzle-orm";

import type { Db } from "../db/client.js";
import { items } from "../db/schema.js";
import { resolvePerson } from "../people/identities.js";
import type { Spool } from "./spool.js";

export interface DrainResult {
  readonly inserted: number;
  readonly quarantined: number;
  readonly deferred: number;
}

export interface Drain {
  drainOnce(): Promise<DrainResult>;
  start(): void;
  stop(): Promise<void>;
}

export interface DrainOptions {
  readonly spool: Spool;
  readonly db: Db;
  readonly intervalMs: number;
  readonly maxBackoffMs?: number;
  readonly onError?: (error: unknown) => void;
  /** Not an error. The row still lands; nobody knows whose it is yet. */
  readonly onUnknownIdentity?: (transport: string, senderId: string) => void;
}

/**
 * Permanent means retrying cannot help: the file is not readable as a capture,
 * or the row violates a constraint that will still be violated next time.
 * Everything else — connection refused, a failover in progress, an error nobody
 * anticipated — is transient, because deferring costs a second and quarantining
 * a good capture costs the thought.
 */
function isPermanent(error: unknown): boolean {
  if (error instanceof SyntaxError) return true;
  const code = (error as { code?: unknown }).code;
  return typeof code === "string" && (code.startsWith("22") || code.startsWith("23"));
}

export function createDrain(options: DrainOptions): Drain {
  const { spool, db, intervalMs } = options;
  const maxBackoffMs = options.maxBackoffMs ?? 30_000;
  const onError = options.onError ?? (() => {});

  let timer: NodeJS.Timeout | undefined;
  let running = false;
  let backoffMs = intervalMs;

  async function drainOnce(): Promise<DrainResult> {
    let inserted = 0;
    let quarantined = 0;
    let deferred = 0;

    for (const name of await spool.list()) {
      try {
        const capture = await spool.read(name);
        const personId = await resolvePerson(db, capture.transport, capture.senderId);
        if (personId === null && capture.senderId !== null) {
          options.onUnknownIdentity?.(capture.transport, capture.senderId);
        }

        await db
          .insert(items)
          .values({
            transport: capture.transport,
            externalId: capture.externalId,
            conversationId: capture.conversationId,
            senderId: capture.senderId,
            personId,
            rawText: capture.text,
            payload: capture.payload,
            receivedAt: capture.receivedAt,
          })
          // Redelivery is harmless. The window between inserting a row and
          // deleting its file is small, but it is real.
          //
          // This predicate is mandatory, not decoration: the index is partial,
          // and Postgres rejects an ON CONFLICT target whose predicate does not
          // match the index with "no unique or exclusion constraint matching
          // the ON CONFLICT specification". Note: onConflictDoNothing's config
          // key for this is `where` (unlike onConflictDoUpdate, which uses
          // `targetWhere`) — verified against the installed drizzle-orm's
          // onConflictDoNothing implementation, which only recognises `target`
          // and `where`.
          .onConflictDoNothing({
            target: [items.transport, items.externalId],
            where: sql`${items.externalId} is not null`,
          });

        await spool.remove(name);
        inserted += 1;
      } catch (error) {
        onError(error);
        if (isPermanent(error)) {
          // Never deleted. A file that cannot be inserted must not spin
          // forever, and it must not disappear either.
          await spool.quarantine(name);
          quarantined += 1;
        } else {
          deferred += 1;
        }
      }
    }

    return { inserted, quarantined, deferred };
  }

  function schedule(delayMs: number): void {
    timer = setTimeout(() => {
      void tick();
    }, delayMs);
    timer.unref();
  }

  async function tick(): Promise<void> {
    if (!running) return;
    let result: DrainResult = { inserted: 0, quarantined: 0, deferred: 0 };
    try {
      result = await drainOnce();
    } catch (error) {
      onError(error);
      result = { inserted: 0, quarantined: 0, deferred: 1 };
    }
    backoffMs = result.deferred > 0 ? Math.min(backoffMs * 2, maxBackoffMs) : intervalMs;
    if (running) schedule(backoffMs);
  }

  return {
    drainOnce,
    start() {
      if (running) return;
      running = true;
      backoffMs = intervalMs;
      schedule(0);
    },
    async stop() {
      running = false;
      if (timer) clearTimeout(timer);
      timer = undefined;
    },
  };
}
