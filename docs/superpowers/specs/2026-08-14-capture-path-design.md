# Squirrel — phase 1: the capture path

Status: approved 2026-08-14. Scope is one thing: a Campfire direct message
arrives, its raw text is stored durably, and the room gets a 🐿️ back.

## Why this shape

The system's premise is that a lost thought destroys trust in the whole tool.
Everything below follows from taking that literally: the durability guarantee
must rest on code that runs on every single capture, not on a fallback path
that only executes during an outage.

Two design principles pull against each other here, and the conflict is worth
naming rather than papering over:

- *Acknowledge instantly; parsing and scheduling happen on a queue afterwards.*
- *Never lose input. Store the raw text verbatim before any parsing, always.*

If the 🐿️ goes out before the text is durable, a failed write reads as a
successful capture. That is silent loss — the worst available failure. So the
acknowledgement is **not** instant in the strict sense: it comes after one
`fsync`, roughly a millisecond, against a seven-second budget. Everything that
is genuinely deferrable stays deferred; the write does not.

## What Campfire actually does

Read from `basecamp/once-campfire`, not assumed. Line references are to that
repository.

The webhook body (`app/models/webhook.rb:41`):

```json
{
  "user":    { "id": 1, "name": "Ronald" },
  "room":    { "id": 7, "name": "Squirrel", "path": "/rooms/7/<bot_id>-<bot_token>/messages" },
  "message": { "id": 42, "body": { "html": "…", "plain": "…" }, "path": "/rooms/7/@42" }
}
```

Facts that constrain the design:

- **No timestamp.** Campfire sends no `created_at` for the message. Our receipt
  time is the only clock we have, and `message.id` is the only stable identity.
- **No signature, no shared secret, no HMAC.** There is nothing to verify. The
  callback URL is the entire authentication story, which is why the service is
  ClusterIP with no Ingress and a NetworkPolicy naming Campfire as the only
  permitted caller — the same reasoning already written down in
  `homelab/apps/base/campfire-status-bot/service.yaml`.
- **In a direct room, every message fires the webhook** — no mention required
  (`app/controllers/messages_controller.rb:85`). The bot's own messages are
  excluded, so there is no echo to filter.
- `message.body.plain` arrives with bot mentions already stripped;
  `message.body.html` is the untouched rich text.
- **The reply is the HTTP response body.** A 200 with `text/html` or
  `text/plain` is posted into the room as the bot's message.
- **A non-200 response that still carries a `Content-Type` is uploaded as an
  attachment.** A 500 with a content type puts an error page in the room. The
  only way to say nothing is to send no `Content-Type` at all.
- **Campfire gives up after seven seconds** and posts "Failed to respond within
  7 seconds" itself.
- **Campfire never retries.** `Webhook#deliver` is one shot. A dropped request
  is a lost thought with no second chance, which is what the spool exists for.

The outbound endpoint, `POST /rooms/:id/:bot_key/messages` with the raw message
text as the body, is not used in phase 1. It is noted because `room.path` in
the payload already embeds the bot key, so a future proactive nudge needs no
credential of its own.

## Architecture

Two loops that share only a directory on disk.

```
Campfire ──POST──▶ HTTP edge ──▶ guard ──▶ spool (fsync + rename) ──▶ 🐿️
                                                │
                                          spool directory
                                                │
                       drain loop ──────────────┘──▶ Postgres ──▶ delete file
```

The request path never touches Postgres. A database outage, a CNPG failover, a
slow query, and a migration in progress are all invisible to the person typing
into the room. The cost is that Postgres lags the spool by up to one drain
interval; phase 1 reads nothing back, so the lag is unobservable.

### Components

| Module | Responsibility | Depends on |
|---|---|---|
| `config.ts` | Parse and validate environment at boot; fail loudly | — |
| `server.ts` | Hono app: `POST /webhook`, `GET /healthz` | campfire, spool |
| `campfire.ts` | Payload types, field extraction, the guard, the reply contract | — |
| `spool.ts` | Durable write, list, read, remove, quarantine | fs |
| `drain.ts` | Interval loop moving spooled records into Postgres | spool, db |
| `db/` | Drizzle schema, client, migrations | pg |

`campfire.ts` and `spool.ts` are pure enough to test without a socket or a
database. `drain.ts` needs a real Postgres.

## Request path

1. Read the raw request body as a string. Nothing is parsed yet.
2. `JSON.parse` the envelope. This is reading Campfire's wrapper, not
   interpreting the thought — it is not a gate on the thought's content.
3. Apply the guard (below).
4. Serialise `{ receivedAt, rawBody, extracted }` to
   `<spool>/<timestamp>-<messageId>.json.tmp`, `fsync`, `rename` to `.json`.
   `rename` is atomic, so a partial file is never visible to the drain loop.
5. Respond `200` with `Content-Type: text/plain; charset=utf-8` and body `🐿️`.

Nothing may throw out of the handler. An unhandled throw becomes a 500, and a
500 with a content type uploads an error page into the room.

### The guard, and which way it fails

The service accepts messages from one configured room and one configured user.
The direction of failure differs by cause, and this is deliberate:

- **Understood the envelope, wrong room or wrong user** → fail closed. Log, and
  respond 200 with *no* `Content-Type` and no body: total silence in the room.
- **Could not parse the envelope, or `room.id` / `user.id` is absent** → fail
  **open**. Spool it anyway, reply 🐿️, log at warn.

The second rule exists because of a specific failure I want to make impossible:
Campfire changes its payload shape in an upgrade, `room.id` becomes `undefined`,
the guard rejects every message, and captures vanish silently for days. Failing
open on "I do not understand this envelope" costs at most some junk rows in a
table nobody reads yet. Failing closed there costs the entire premise of the
system.

### When the spool itself fails

Disk full, permissions wrong, volume detached. There is no honest way to hide
this, and inventing one would be exactly the false trust the principles warn
about. Reply, in plain text: `⚠️ couldn't save that — please resend`. Log at
error. This is the one path where the bot admits it failed, and it is correct
that it does.

## Spool format

One file per capture, not an append-only log with offsets.

```
/var/spool/squirrel/
  2026-08-14T09:31:04.512Z-42.json      ready to drain
  2026-08-14T09:31:07.883Z-43.json.tmp  mid-write, invisible to the drain
  quarantine/
    2026-08-14T08:02:11.004Z-38.json    permanently un-insertable, kept forever
```

File contents:

```json
{
  "receivedAt": "2026-08-14T09:31:04.512Z",
  "rawBody": "{\"user\":{…}}",
  "extracted": { "campfireMessageId": 42, "roomId": 7, "…": "…" }
}
```

`rawBody` is the request body byte-for-byte. Every other field is a derived
view over it, and any of them can be recomputed later if the extraction turns
out to be wrong.

Why files rather than an append log: no offset bookkeeping, no partial-line
parsing, no rotation, and crash recovery is a directory listing. A crash at any
instant leaves either a `.tmp` to sweep at boot or a complete file to drain.

Filenames sort chronologically, so the drain inserts in arrival order. When the
message id is unknown (the fail-open case) the name uses a random suffix
instead.

## Drain loop

Every `DRAIN_INTERVAL_MS` (default 1000):

1. List `*.json`, sorted.
2. For each: `INSERT … ON CONFLICT (campfire_message_id) DO NOTHING`.
3. On success, delete the file.

Error handling splits by kind, and the split matters:

- **Transient** (connection refused, timeout, failover in progress): leave the
  file, back off exponentially to a 30-second ceiling, retry. Nothing is lost;
  the directory just grows until Postgres returns.
- **Permanent** (unparseable file, constraint violation that retrying cannot
  fix): move to `quarantine/`, log at error. Never delete. A file that cannot
  be inserted must not spin forever, and it must not disappear either.

`ON CONFLICT DO NOTHING` makes redelivery harmless, which matters because the
crash window between inserting and deleting a file is real, if small.

## Data model

```sql
create table items (
  id                  bigint generated always as identity primary key,
  campfire_message_id bigint,
  room_id             bigint,
  room_name           text,
  user_id             bigint,
  user_name           text,
  raw_text            text        not null,
  body_html           text,
  payload             jsonb       not null,
  received_at         timestamptz not null,
  inserted_at         timestamptz not null default now()
);

create unique index items_campfire_message_id_key
  on items (campfire_message_id)
  where campfire_message_id is not null;
```

Notes on the deviations from the sketch in the kickoff:

- **No `state` column yet.** Every row would be `inbox`, nothing would read it,
  and nothing would transition it. It arrives with the phase that needs it.
- **No `source` column yet.** There is exactly one source. Two concrete cases
  before an interface.
- `campfire_message_id` is nullable so the fail-open path has somewhere to land;
  the partial unique index still enforces one row per real Campfire message.
  `raw_text` stays `not null` and extracts to an empty string when
  `message.body.plain` is absent — an empty capture is still a capture, and
  `payload` holds whatever did arrive.
- `payload` holds the whole envelope. `raw_text`, `body_html` and the rest are
  extractions, kept as columns because querying jsonb for them every time is
  needlessly awkward. If an extraction is wrong, `payload` is the record of
  truth and a backfill is possible.
- `received_at` is our clock, because Campfire does not send one.
- No `triggers`, `firings`, `events`, or `chores` tables. Those belong to later
  phases and nothing in phase 1 writes them.

## Boot order

The HTTP server binds **before** anything touches Postgres.

Migrations and the drain loop start afterwards, in the background, with retry.
If the database is unreachable at pod start, Squirrel still serves, still
spools, and still answers 🐿️ — it simply has a growing directory until Postgres
comes back.

The alternative — migrate, then serve — means a Postgres outage during a pod
restart produces a service that refuses connections, and every message sent in
that window is lost with no retry. That is precisely the failure the spool
exists to prevent, so it must not be reintroduced at boot.

## Health

`GET /healthz` returns 200 when the process is serving and the spool directory
is writable. It **deliberately does not check Postgres**.

A readiness probe that fails on a database outage would remove the pod from its
Service, Campfire's webhook delivery would fail, and — because Campfire never
retries — those messages would be gone. Making the pod unready during a
database outage would convert a survivable condition into permanent data loss.
The drain loop's health belongs in logs, not in a readiness gate.

## Configuration

| Variable | Required | Default | Notes |
|---|---|---|---|
| `PORT` | no | `8080` | |
| `SPOOL_DIR` | no | `/var/spool/squirrel` | PVC mount point |
| `DRAIN_INTERVAL_MS` | no | `1000` | |
| `CAMPFIRE_ROOM_ID` | yes | — | Guard |
| `CAMPFIRE_USER_ID` | yes | — | Guard |
| `POSTGRES_SERVER` | yes | — | `postgres-cluster-rw.database.svc.cluster.local` |
| `POSTGRES_PORT` | no | `5432` | |
| `POSTGRES_DB` | yes | — | `squirrel` |
| `POSTGRES_USER` | yes | — | From `pg-role-squirrel`, key `username` |
| `POSTGRES_PASSWORD` | yes | — | From `pg-role-squirrel`, key `password` |

Naming follows the existing apps (`mealie`, `linkding`) so the deployment patch
reads the same as its neighbours. Missing required configuration exits non-zero
at boot rather than degrading quietly.

## Testing

The capture path must not drop anything, so the tests are written against the
failure modes rather than the happy path.

Unit, no external services:

- Spool: `.tmp` then `rename`; a `.tmp` left by a crash is swept at boot and
  never drained; concurrent writes do not collide on a filename.
- Guard: right room and user accepted; wrong room silent; wrong user silent;
  absent `room.id` fails open and spools; unparseable JSON fails open and
  spools.
- Response contract: acceptance sends `text/plain` and `🐿️`; the silent path
  sends **no** `Content-Type` header at all (asserted explicitly — a framework
  that helpfully adds one would post junk into the room); a handler whose spool
  throws still returns 200 and never a 500.

Against a real Postgres:

- Drain inserts a spooled file and deletes it.
- Redelivery of the same `campfire_message_id` inserts one row.
- Postgres unreachable: the file stays, the loop backs off, and the row appears
  once the database returns.
- A permanently un-insertable file lands in `quarantine/` and is not retried.

Postgres for tests comes from a GitHub Actions service container in CI and a
documented `docker run` locally. No test-harness dependency is added for it.

## Stack

| Choice | |
|---|---|
| Runtime | Node 24 LTS |
| HTTP | Hono + `@hono/node-server` |
| Database | Drizzle ORM + `pg`, drizzle-kit migrations |
| Tests | Vitest |
| Lint/format | Biome |

No `pg-boss`. Phase 1 has no scheduler and nothing to enqueue; the drain loop is
a `setInterval` over a directory, not a job queue. It arrives with the phase
that needs it.

## Deployment

This repository owns the code, the `Dockerfile`, and CI. Manifests live in
`ronaldlokers/homelab`, matching the convention every other Campfire workload
follows.

CI mirrors `stringer`: typecheck and test on every push and pull request; build
and push `linux/amd64,linux/arm64` to `ghcr.io/ronaldlokers/squirrel` on `v*`
tags only. Both architectures are mandatory — staging is amd64, production is
arm64.

The homelab side, as a separate change:

- `Database` CR named `squirrel`, owner `squirrel`, against `postgres-cluster`.
- SOPS-encrypted `pg-role-squirrel` Secret in `database`, reconciled through
  `cluster.spec.managed.roles`, mirrored into the app namespace by reflector.
- Deployment, `replicas: 1` (the spool PVC is RWO, and every Campfire workload
  is single-replica already), non-root, read-only root filesystem, PVC mounted
  at `/var/spool/squirrel`.
- ClusterIP Service, no Ingress.
- NetworkPolicy: ingress only from the Campfire pod; egress only to
  `postgres-cluster-rw` and DNS.

## Deferred

Named so that nobody builds them early: `?` and `done` handling, the scheduler
and firings, chores, the nightly clarification pass, LLM parsing, Home Assistant
sensor evidence, context triggers, any web UI, and authentication beyond the
network boundary.

The point of phase 1 is a week of real captures. The parser gets written against
those, not against imagined ones.
