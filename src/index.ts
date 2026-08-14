import { type Drain, createDrain } from "./capture/drain.js";
import type { Allow } from "./capture/policy.js";
import { createSink } from "./capture/sink.js";
import { type Spool, openSpool } from "./capture/spool.js";
import { type Config, loadConfig } from "./config.js";
import { type DbHandle, openDb } from "./db/client.js";
import { applyMigrations } from "./db/migrate.js";
import { createHttp, startHttp } from "./http.js";
import { type IdentitySeed, seedOwner } from "./people/identities.js";
import { createCampfireTransport } from "./transports/campfire.js";
import type { Stop, Transport } from "./transports/transport.js";

export interface Squirrel {
  readonly port: number;
  stop(): Promise<void>;
}

function log(event: string, detail: Record<string, unknown> = {}): void {
  process.stdout.write(`${JSON.stringify({ event, ...detail })}\n`);
}

function transportsFrom(config: Config): Transport[] {
  const built: Transport[] = [];
  if (config.campfire !== null) built.push(createCampfireTransport(config.campfire));
  return built;
}

function allowsFrom(config: Config): Allow[] {
  if (config.campfire === null) return [];
  return [
    {
      transport: "campfire",
      conversationId: config.campfire.conversationId,
      senderId: config.campfire.senderId,
    },
  ];
}

function seedsFrom(config: Config): IdentitySeed[] {
  if (config.campfire === null) return [];
  return [{ transport: "campfire", externalId: config.campfire.senderId }];
}

/**
 * Mutable box shared between `boot` and `connectAndDrain`.
 *
 * `stopped` guards against a retry firing after `stop()` has already run: once
 * set, `connectAndDrain` neither reschedules nor touches the (by then closed)
 * pool, and any pending retry timer is cleared so nothing outlives the process
 * that started it.
 */
interface DrainRef {
  drain?: Drain;
  stopped: boolean;
  retryTimer?: NodeJS.Timeout;
}

/**
 * Postgres in the background, on purpose.
 *
 * Migrating before serving means a database outage during a pod restart
 * produces a service that refuses connections — and every message sent in that
 * window is gone, because Campfire does not retry. So the server binds first,
 * transports start, and the database is caught up with afterwards.
 */
async function connectAndDrain(
  config: Config,
  dbHandle: DbHandle,
  drainRef: DrainRef,
  spool: Spool,
): Promise<void> {
  if (drainRef.stopped) return;
  try {
    await applyMigrations(dbHandle.db);
    await seedOwner(dbHandle.db, config.ownerHandle, seedsFrom(config));
    if (drainRef.stopped) return;
    const drain = createDrain({
      spool,
      db: dbHandle.db,
      intervalMs: config.drainIntervalMs,
      onError: (error) => log("drain.error", { error: String(error) }),
      onUnknownIdentity: (transport, senderId) => log("identity.unknown", { transport, senderId }),
    });
    drainRef.drain = drain;
    drain.start();
    log("db.ready");
  } catch (error) {
    log("db.unavailable", { error: String(error) });
    if (drainRef.stopped) return;
    drainRef.retryTimer = setTimeout(() => {
      void connectAndDrain(config, dbHandle, drainRef, spool);
    }, config.drainIntervalMs).unref();
  }
}

export async function boot(env: Record<string, string | undefined>): Promise<Squirrel> {
  const config = loadConfig(env);

  const spool = await openSpool(config.spoolDir);
  const swept = await spool.sweep();
  if (swept > 0) log("spool.swept", { files: swept });

  const sink = createSink(spool, allowsFrom(config), {
    onError: (error) => log("spool.write_failed", { error: String(error) }),
    // Silent in the room, visible here. Being added to a room Squirrel does not
    // belong in should show up somewhere.
    onIgnored: (capture) =>
      log("capture.ignored", {
        transport: capture.transport,
        conversationId: capture.conversationId,
        senderId: capture.senderId,
      }),
  });

  const { app, mount } = createHttp(spool);
  const stops: Stop[] = [];
  for (const transport of transportsFrom(config)) {
    stops.push(await transport.start(sink, mount));
    log("transport.started", { transport: transport.name, sends: transport.send !== null });
  }

  const serving = await startHttp(app, config.port);
  log("http.listening", { port: serving.port });

  const dbHandle = openDb(config.postgres);
  const drainRef: DrainRef = { stopped: false };
  void connectAndDrain(config, dbHandle, drainRef, spool);

  return {
    port: serving.port,
    async stop() {
      drainRef.stopped = true;
      if (drainRef.retryTimer) clearTimeout(drainRef.retryTimer);
      await drainRef.drain?.stop();
      for (const stop of stops) await stop();
      await serving.close();
      await dbHandle.close();
    },
  };
}

// Entry point when run as a program rather than imported by a test.
if (process.argv[1]?.endsWith("index.js")) {
  try {
    const squirrel = await boot(process.env);
    // Node's default action on SIGTERM is immediate termination. Without
    // this, every rollout, node drain or eviction severs whatever webhook is
    // in flight, and Campfire does not retry it. `stop()` already stops
    // accepting new connections and waits for in-flight requests via
    // `serving.close()`.
    for (const signal of ["SIGTERM", "SIGINT"] as const) {
      process.once(signal, () => {
        void squirrel.stop().then(() => process.exit(0));
      });
    }
  } catch (error: unknown) {
    process.stderr.write(`${JSON.stringify({ event: "boot.failed", error: String(error) })}\n`);
    process.exit(1);
  }
}
