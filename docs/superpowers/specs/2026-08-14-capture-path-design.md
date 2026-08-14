# Squirrel — phase 1: the capture path

Status: approved 2026-08-14; revised the same day to make the transport a
replaceable adapter, to state how "direct messages only" is enforced, and to
make the transport boundary symmetric — outbound in the interface, and identity
promoted out of transport-local ids into people.

Scope is one thing: a direct message arrives, its raw text is stored durably,
and the conversation gets a 🐿️ back.

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

## Two halves

The process splits in one place, and only one:

**The core** knows about captures, durability, and a database. It has never
heard of Campfire, HTTP, or webhooks.

**A transport** knows about exactly one chat system: how messages arrive from
it, how identity is spelled in it, and how it wants to be answered. It converts
that into a `Capture` and hands it over.

Campfire is the only transport today. The boundary exists anyway, at the
explicit request of the person who will live with this, and against the working
agreement's own rule that two concrete cases should precede an interface. Three
things make it defensible rather than speculative:

1. It has already been needed once. `stringer/src/desks/desk.ts` is this same
   abstraction, written for the same reason, and its header comment already
   works out which parts of a chat transport are load-bearing.
2. Campfire's contract is unusually invasive — a seven-second deadline, replies
   travelling in the HTTP response body, non-200 responses becoming file
   uploads. Without a boundary those quirks end up smeared through the storage
   code, where a second transport cannot dislodge them.
3. The interface is two methods and one data type. If it turns out wrong, the
   second transport reshapes it cheaply.

The known risk is the usual one: an abstraction designed against a single
implementation is usually the wrong abstraction. The mitigation is to keep it
small and to resist adding to it until something real pulls.

### The interface

```ts
/** What every transport must produce. Nothing here is Campfire-shaped. */
export interface Capture {
  readonly transport: string;            // "campfire"
  readonly externalId: string | null;    // the transport's message id
  readonly conversationId: string | null;
  readonly senderId: string | null;
  readonly text: string;                 // verbatim, never trimmed or parsed
  readonly receivedAt: Date;             // our clock
  readonly payload: unknown;             // the original message, untouched
}

export type Outcome = "stored" | "ignored" | "failed";

export interface CaptureSink {
  /** Resolves only once the capture is durable. */
  accept(capture: Capture): Promise<Outcome>;
}

export interface HttpMount {
  post(path: string, handler: WebHandler): void;
}

export interface Transport {
  readonly name: string;
  /** Begin receiving. `http` may be ignored by a transport that polls. */
  start(sink: CaptureSink, http: HttpMount): Promise<() => Promise<void>>;
  /**
   * Send a message the system initiated, rather than one it is answering.
   * Null when this transport cannot initiate — a bot that can only reply has
   * to be able to say so rather than fail at the moment it matters.
   */
  readonly send: ((conversationId: string, text: string) => Promise<void>) | null;
}
```

The process owns one HTTP server, for `/healthz` and for any transport that
receives over HTTP. A polling transport — Matrix, or Telegram in long-poll mode
— takes `http` and does nothing with it.

`send` has no caller in phase 1. It exists because the shape of outbound is not
obvious and differs sharply between transports: Campfire answers inside the HTTP
response *and* offers a separate endpoint for initiating, while most systems
have only the second. Getting that asymmetry into the interface now costs one
nullable method; discovering it while building the scheduler would mean
reshaping the transport boundary at the same time as the thing using it. It is
implemented and tested, and nothing calls it until firings exist.

The nullability is not decoration. `send` is null unless the Campfire adapter
has been given a bot key, and phase 1 ships without one — so the honest answer
to "can this bot nudge me?" is currently *no*, and the type says so.

**The rule that keeps the boundary honest:** a column in the database or a field
on `Capture` exists only if every plausible transport can fill it. Everything
else lives in `payload`. Campfire's `body.html` is the worked example: it is
real, it is preserved verbatim, and it is not a column.

## The Campfire adapter

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

Mapped to a `Capture`: `message.id` → `externalId`, `room.id` →
`conversationId`, `user.id` → `senderId`, `message.body.plain` → `text`, whole
body → `payload`.

Facts that constrain the adapter, all of which stay inside it:

- **No timestamp.** Campfire sends no `created_at`. Our receipt time is the only
  clock we have, and `message.id` the only stable identity.
- **No signature, no shared secret, no HMAC.** There is nothing to verify. The
  callback URL is the entire authentication story, which is why the Service is
  ClusterIP with no Ingress and a NetworkPolicy names Campfire as the only
  permitted caller — the same reasoning already written down in
  `homelab/apps/base/campfire-status-bot/service.yaml`.
- **In a direct room, every message fires the webhook** — no mention required
  (`app/controllers/messages_controller.rb:85`). In any other room the webhook
  fires only on an explicit mention of the bot. The bot's own messages are
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

Outcomes map to responses entirely within the adapter:

| Outcome | Response |
|---|---|
| `stored` | `200`, `text/plain; charset=utf-8`, body `🐿️` |
| `ignored` | `200`, **no `Content-Type`**, empty body — total silence |
| `failed` | `200`, `text/plain`, `⚠️ couldn't save that — please resend` |

`failed` is a 200 because a non-200 carrying a content type would upload an
error page into the room. That inversion is a Campfire quirk and it does not
leave this table.

### Outbound, and the credential it costs

`send` posts the raw text as the body of
`${CAMPFIRE_BASE_URL}/rooms/${conversationId}/${CAMPFIRE_BOT_KEY}/messages`.

The obvious cheaper route is to reuse `room.path` out of a stored payload, which
already embeds the bot key and would need no secret at all. It is rejected:
outbound would then only reach rooms Squirrel had recently heard from, and a
9am nudge would depend on the capture history rather than on configuration.
Coupling "where can I speak" to "who has spoken to me" is the kind of thing that
works in testing and fails on a quiet Monday.

The cost is real and worth naming. Today this pod holds **no** Campfire
credential — every reply travels back inside the request it is answering. A bot
key changes that: it is a credential that can post as Squirrel into any room in
the account. It is therefore optional, absent in phase 1, and when present it
comes from SOPS like every other secret, with egress restricted to Campfire.

When `CAMPFIRE_BOT_KEY` is unset, `send` is `null`. The transport does not
half-work.

## Direct messages only

Squirrel accepts messages from one configured conversation and one configured
sender. In Campfire that conversation is a direct room.

**This is enforced by allowlist, not by inspection, and the difference is worth
being precise about.** Campfire models direct rooms as a distinct subclass with
a `direct?` predicate (`app/models/room.rb:61`, `Rooms::Direct`), but that
predicate is never serialised into the webhook payload. Squirrel therefore
*cannot* verify that a given message came from a direct room. Claiming
otherwise would be exactly the false trust the principles warn against.

What actually holds the line, in order of strength:

1. `CAMPFIRE_CONVERSATION_ID` names one room. Anything else is silence. If that
   id is a direct room, only direct-room messages are ever stored.
2. `CAMPFIRE_SENDER_ID` names one person. A different member of that room is
   silence too.
3. Outside direct rooms Campfire only fires the webhook on an explicit mention,
   so a group room cannot leak captures merely by having the bot in it.
4. A message from any unconfigured conversation is logged at warn with its
   conversation id. Adding the bot somewhere it does not belong is visible in
   the logs rather than silent.

The operator obligation this leaves is one line, and it belongs in the runbook:
**the id in `CAMPFIRE_CONVERSATION_ID` must be a direct room.** Nothing in the
code can check it.

### The guard, and which way it fails

The direction of failure differs by cause, and this is deliberate:

| Adapter parsed the message | Conversation | Sender | Outcome |
|---|---|---|---|
| yes | matches | matches | `stored` |
| yes | mismatch | — | `ignored` |
| yes | matches | mismatch | `ignored` |
| no, or the id fields are absent | `null` | `null` | `stored`, logged at warn |

That last row is the important one. If Campfire changes its payload shape in an
upgrade and `room.id` becomes `undefined`, a guard that fails closed drops every
capture silently, for days, with the bot still cheerfully answering nothing.
Failing open on *"I do not understand this envelope"* costs at most some junk
rows in a table nothing reads yet. Failing closed there costs the premise of the
system.

Failing closed on *"I understood it and it is the wrong room"* is the opposite
case and is correct: that is a message the system genuinely was not addressed
with.

## Identity

A `senderId` means "user 1, in Campfire". It is a transport-local coordinate and
nothing joins two of them. Add a second transport and the same person arrives as
two unrelated strings, with the items table unable to say they are one.

So a person is a first-class row, and a transport identity points at it:

- `people` — one row, the owner, seeded from configuration.
- `identities` — `(transport, external_id) → person_id`, unique on the pair.
- `items.person_id` — nullable, resolved at drain time.

**Resolution happens in the drain loop, never on the request path.** This is not
a detail. The request path deliberately does not touch Postgres, because that is
what makes a database outage survivable, and a person lookup is a database read.
The two paths therefore ask different questions:

| Path | Question | Answered from |
|---|---|---|
| Request | Is this addressed to us? | Config allowlist, no I/O |
| Drain | Whose is this? | `identities` join |

An unresolved identity leaves `person_id` null and logs at warn. **Unknown
identities are never auto-created.** Auto-vivifying a person on first sight
would quietly re-admit anyone the allowlist just turned away, which would make
the guard decorative.

Seeding is declarative rather than administrative: at boot, `OWNER_HANDLE` plus
each enabled transport's configured sender id are upserted into `people` and
`identities`. Adding a transport means adding its sender id to configuration in
Git and letting Flux apply it — there is no admin screen to forget about, and
the desired state is reviewable in a diff.

Phase 1 has exactly one person and one identity. The tables are still the right
shape, because the retrofit is the expensive direction: rewriting captured rows
to point at people invented later means reconstructing, months on, which id
meant whom.

## Request path

1. The adapter reads the raw request body as a string. Nothing is parsed yet.
2. It parses the envelope — reading the transport's wrapper, not interpreting
   the thought — and builds a `Capture`, with nulls where a field is missing.
3. `sink.accept(capture)` applies the guard, and for an accepted capture
   serialises `Capture` to `<spool>/<timestamp>-<transport>-<externalId>.json.tmp`,
   `fsync`s, and `rename`s to `.json`. `rename` is atomic, so a partial file is
   never visible to the drain loop.
4. `accept` resolves with an `Outcome`; the adapter turns that into a response.

Nothing may throw out of the handler. An unhandled throw becomes a 500, and a
500 with a content type uploads an error page into the room.

When the spool itself fails — disk full, permissions wrong, volume detached —
there is no honest way to hide it, and inventing one would be the false trust
again. The outcome is `failed` and the room is told to resend. This is the one
path where the bot admits it could not do its job, and it is right that it does.

## Spool format

One file per capture, not an append-only log with offsets.

```
/var/spool/squirrel/
  2026-08-14T09:31:04.512Z-campfire-42.json      ready to drain
  2026-08-14T09:31:07.883Z-campfire-43.json.tmp  mid-write, invisible
  quarantine/
    2026-08-14T08:02:11.004Z-campfire-38.json    un-insertable, kept forever
```

Each file is a serialised `Capture`. `payload` is the original message exactly
as the transport received it; every other field is a derived view over it and
can be recomputed later if the mapping turns out wrong.

Why files rather than an append log: no offset bookkeeping, no partial-line
parsing, no rotation, and crash recovery is a directory listing. A crash at any
instant leaves either a `.tmp` to sweep at boot or a complete file to drain.

Filenames sort chronologically, so the drain inserts in arrival order. Where the
external id is unknown — the fail-open row above — a random suffix is used
instead.

## Drain loop

Every `DRAIN_INTERVAL_MS` (default 1000):

1. List `*.json`, sorted.
2. For each, resolve `(transport, sender_id)` against `identities` to a
   `person_id`, or null with a warn log if it is unknown.
3. `INSERT … ON CONFLICT (transport, external_id) DO NOTHING`.
4. On success, delete the file.

An unknown identity is **not** a permanent error. It leaves `person_id` null,
the row still lands, and the file is still deleted — a capture is never held
hostage to knowing whose it was.

Error handling splits by kind, and the split matters:

- **Transient** (connection refused, timeout, failover in progress): leave the
  file, back off exponentially to a 30-second ceiling, retry. Nothing is lost;
  the directory grows until Postgres returns.
- **Permanent** (unparseable file, a constraint violation that retrying cannot
  fix): move to `quarantine/`, log at error. Never delete. A file that cannot be
  inserted must not spin forever, and it must not disappear either.

`ON CONFLICT DO NOTHING` makes redelivery harmless, which matters because the
crash window between inserting a row and deleting its file is real, if small.

## Data model

```sql
create table people (
  id         bigint generated always as identity primary key,
  handle     text        not null unique,
  created_at timestamptz not null default now()
);

create table identities (
  id          bigint generated always as identity primary key,
  person_id   bigint      not null references people (id),
  transport   text        not null,
  external_id text        not null,
  created_at  timestamptz not null default now(),
  unique (transport, external_id)
);

create table items (
  id              bigint generated always as identity primary key,
  transport       text        not null,
  external_id     text,
  conversation_id text,
  sender_id       text,
  person_id       bigint      references people (id),
  raw_text        text        not null,
  payload         jsonb       not null,
  received_at     timestamptz not null,
  inserted_at     timestamptz not null default now()
);

create unique index items_transport_external_id_key
  on items (transport, external_id)
  where external_id is not null;
```

Notes, including where this departs from the sketch in the kickoff:

- **`transport` is the `source` column, and it has earned its place.** An
  earlier draft cut `source` on YAGNI grounds — one source, nothing reads it.
  Making the transport replaceable creates the second case that argument was
  waiting for, so the column comes back under its real name.
- **Identifiers are `text`, not `bigint`.** Campfire's are integers; Matrix room
  ids are strings and Telegram chat ids are signed 64-bit. Text is the only type
  that spans them, and nothing arithmetic is ever done with these.
- **No `state` column yet.** Every row would be `inbox`, nothing would read it,
  and nothing would transition it. It arrives with the phase that needs it.
- **No `body_html`, no `room_name`, no `sender_name`.** They are either
  transport-specific or unqueried. All three survive in `payload`.
- `external_id`, `conversation_id` and `sender_id` are nullable so the fail-open
  path has somewhere to land. The partial unique index still enforces one row
  per real message.
- **`sender_id` survives alongside `person_id`**, rather than being replaced by
  it. `person_id` is an interpretation and can be wrong or absent; `sender_id`
  is what the transport actually said. Keeping both means a mis-seeded identity
  is a backfill rather than an archaeology exercise.
- `person_id` is nullable because a capture that fails open has no known sender,
  and because an identity may arrive before anyone has bound it to a person.
- `raw_text` stays `not null` and maps to an empty string when the transport
  found no text. An empty capture is still a capture, and `payload` holds
  whatever did arrive.
- `received_at` is our clock, because Campfire does not send one and there is no
  guarantee the next transport will either.
- No `triggers`, `firings`, `events` or `chores` tables. Those belong to later
  phases and nothing in phase 1 writes them.

## Boot order

The HTTP server binds, and transports start, **before** anything touches
Postgres.

Migrations and the drain loop start afterwards, in the background, with retry.
If the database is unreachable at pod start, Squirrel still receives, still
spools, and still answers 🐿️ — it simply has a growing directory until Postgres
comes back.

The alternative — migrate, then serve — means a database outage during a pod
restart produces a service that refuses connections, and every message sent in
that window is lost with no retry. That is precisely the failure the spool
exists to prevent, so it must not be reintroduced at boot.

## Health

`GET /healthz` returns 200 when the process is serving and the spool directory
is writable. It **deliberately does not check Postgres**.

A readiness probe that failed on a database outage would remove the pod from its
Service, webhook delivery would fail, and — because Campfire never retries —
those messages would be gone. Making the pod unready during a database outage
converts a survivable condition into permanent data loss. The drain loop's
health belongs in logs, not in a readiness gate.

## Configuration

| Variable | Required | Default | Notes |
|---|---|---|---|
| `PORT` | no | `8080` | Shared HTTP server |
| `TRANSPORTS` | no | `campfire` | Comma-separated; each validates its own block |
| `OWNER_HANDLE` | no | `owner` | Seeds `people`; identities bind to it |
| `SPOOL_DIR` | no | `/var/spool/squirrel` | PVC mount point |
| `DRAIN_INTERVAL_MS` | no | `1000` | |
| `CAMPFIRE_PATH` | no | `/transports/campfire` | Webhook route |
| `CAMPFIRE_CONVERSATION_ID` | yes | — | **Must be a direct room** |
| `CAMPFIRE_SENDER_ID` | yes | — | Guard, and the seeded identity |
| `CAMPFIRE_BASE_URL` | no | — | Required only for `send` |
| `CAMPFIRE_BOT_KEY` | no | — | SOPS. Absent in phase 1, so `send` is null |
| `POSTGRES_SERVER` | yes | — | `postgres-cluster-rw.database.svc.cluster.local` |
| `POSTGRES_PORT` | no | `5432` | |
| `POSTGRES_DB` | yes | — | `squirrel` |
| `POSTGRES_USER` | yes | — | From `pg-role-squirrel`, key `username` |
| `POSTGRES_PASSWORD` | yes | — | From `pg-role-squirrel`, key `password` |

Postgres names follow the existing apps (`mealie`, `linkding`) so the deployment
patch reads the same as its neighbours. Transport settings are namespaced by
transport name, so adding one adds a block rather than editing the core's.
Missing required configuration exits non-zero at boot rather than degrading
quietly.

## Repository layout

```
src/
  index.ts                boot: config, http, transports, then db in background
  config.ts
  http.ts                 shared server, /healthz, HttpMount
  capture/
    capture.ts            Capture, Outcome
    policy.ts             the guard
    sink.ts               CaptureSink over the spool
    spool.ts              write, list, read, remove, quarantine
    drain.ts              the loop
  people/
    identities.ts         resolve, and seed from config
  db/
    schema.ts  client.ts  migrate.ts
  transports/
    transport.ts          Transport, HttpMount
    campfire.ts           the only implementation
```

Nothing under `capture/`, `people/` or `db/` imports from `transports/`. That is
the whole boundary, and it is worth a lint rule rather than a convention.

## Testing

The capture path must not drop anything, so the tests are written against the
failure modes rather than the happy path.

Core, with a fake transport and no HTTP at all:

- A `FakeTransport` feeds the sink directly and the capture reaches Postgres.
  This is the test that proves the core does not know what Campfire is; if it
  ever fails to compile, the boundary has leaked.
- Spool: `.tmp` then `rename`; a `.tmp` left by a crash is swept at boot and
  never drained; concurrent writes do not collide on a filename.
- Policy: matching conversation and sender is `stored`; wrong conversation is
  `ignored`; wrong sender is `ignored`; null ids fail open to `stored`.

Campfire adapter:

- The payload above maps to the expected `Capture`, field by field.
- Unparseable JSON fails open and produces a capture with null ids.
- `stored` sends `text/plain` and `🐿️`.
- The `ignored` path sends **no** `Content-Type` header at all. Asserted
  explicitly, because a framework that helpfully adds one would post junk into
  the room.
- A sink that throws still yields 200 and never a 500.
- `send` is `null` with no bot key configured, and a function with one.
- `send` posts the text to `/rooms/:id/:bot_key/messages`, asserted against a
  stub HTTP server rather than against Campfire.

Identity:

- A configured owner and sender id seed one `people` row and one `identities`
  row; running the seed twice changes nothing.
- A known identity resolves to a `person_id` on the drained row.
- An unknown identity leaves `person_id` null, still stores the row, still
  deletes the spool file, and creates **no** new person — the test that keeps
  the guard from being decorative.
- The same external id under two transports resolves to two identities, and
  binding both to one person makes a single person's captures queryable across
  transports.

Against a real Postgres:

- Drain inserts a spooled file and deletes it.
- Redelivery of the same `(transport, external_id)` inserts one row.
- The same external id under two different transports inserts two rows.
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

Hono's handlers are Web-standard `Request`/`Response`, which is what makes
`HttpMount` expressible without the core learning a framework.

No `pg-boss`. Phase 1 has no scheduler and nothing to enqueue; the drain loop is
an interval over a directory, not a job queue. It arrives with the phase that
needs it.

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

No `CAMPFIRE_BOT_KEY` Secret in phase 1. Adding one later also adds an egress
rule to the Campfire pod, and turns this from a workload holding no Campfire
credential into one that can post as Squirrel anywhere in the account. That is a
deliberate step to take when firings need it, not a box to tick early.

A second transport that talks to something outside the cluster would need its
own egress rule, and — if it polls rather than receives — no ingress at all.
Nothing in the current policy assumes there will only ever be one.

## Deferred

Named so that nobody builds them early: `?` and `done` handling, the scheduler
and firings, chores, the nightly clarification pass, LLM parsing, Home Assistant
sensor evidence, context triggers, any web UI, authentication beyond the network
boundary, and any transport other than Campfire.

The point of phase 1 is a week of real captures. The parser gets written against
those, not against imagined ones.
