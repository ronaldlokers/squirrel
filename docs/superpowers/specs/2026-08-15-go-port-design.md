# Squirrel — porting phase 1 to Go

Status: approved 2026-08-15. A port, not a redesign.

This document covers only what changes. Everything else — the design principles,
the Campfire contract, the guard's two-directional failure, the identity model,
the data model, and the deferred list — is unchanged and lives in
[`2026-08-14-capture-path-design.md`](2026-08-14-capture-path-design.md). Read
that first; this is a delta against it.

## Why

The TypeScript implementation is finished, reviewed and merged. Nothing was
wrong with it. What was wrong was the surface it sat on: every genuine surprise
in that build came from a dependency, not from the design.

- `drizzle-kit` bundled an `esbuild` too old for the project's TypeScript target,
  and crashed before it could read the schema.
- `drizzle`'s `onConflictDoNothing` took `where` where the documentation-shaped
  assumption was `targetWhere`, and getting it wrong makes Postgres reject the
  statement outright.
- `@hono/node-server` stamped a default `content-type` onto headerless
  responses, which silently broke the transport's only way to stay silent. Over
  a real socket it would have posted an empty message into the room on every
  ignored webhook.

Each cost a reproduction, a ruling and a fix. None was a design error. The
answer to a category of problem you keep paying for is to stop having the
category.

Three things pushed the decision, in this order: dependency churn, footprint on
three CM5 nodes, and wanting to write it. Go answers all three. The runtime
dependency count goes from three plus their transitive trees to **one**.

## What Go changes about the design

Almost nothing, and that is the point. Three things get *better*, and they are
worth naming because each maps onto a defect the last build had to find.

**The core/transport boundary becomes a compile error.** In TypeScript it was a
test reading the source tree, because nothing else could enforce it. In Go,
`transport` imports `squirrel`, and Go rejects import cycles — so `squirrel`
importing `transport` does not compile. The boundary test has no Go equivalent
because it needs none.

**Silence is expressible.** `w.WriteHeader(http.StatusOK)` followed by no write
emits no `Content-Type`. Go only sniffs a type when there are bytes to sniff.
The `@hono/node-server` defect is not representable.

**Migrations ship inside the binary.** `embed.FS` puts the SQL in the executable,
so there is no `drizzle/` directory to copy, no generator to run, and no
generator to be surprised by.

One thing gets *worse* and is accepted: Go has no discriminated unions, so the
trigger model arriving in later phases — `time | context | none` — will be a
tagged struct with a validating constructor rather than a type the compiler
makes illegal to misuse. That is a real loss, deferred to the phase that needs it.

## Layout

```
cmd/squirrel/main.go          boot, signals, entry point
internal/squirrel/
  config.go                   environment parsing and validation
  capture.go                  Capture, Outcome, Sink
  policy.go                   the guard
  spool.go                    durable write, list, read, remove, quarantine, sweep
  sink.go                     Sink over the spool
  drain.go                    the loop
  people.go                   seed and resolve identities
  store.go                    pgx pool and every query
  migrate.go                  embedded migration applier
  migrations/0001_init.sql
  http.go                     server, /healthz, mount
internal/transport/
  transport.go                Transport, Mount
  campfire.go                 the only implementation
```

Two packages rather than the TypeScript version's five directories. Go tolerates
a larger package than TypeScript tolerates a larger file, and `capture`, `people`
and `store` are used together on every path — splitting them would buy plumbing,
not clarity. `transport` stays separate because that separation is the design.

A second transport splits `campfire.go` into its own package. That is mechanical
and can wait for the second case.

## The interface

```go
type Capture struct {
	Transport      string
	ExternalID     *string
	ConversationID *string
	SenderID       *string
	Text           string
	ReceivedAt     time.Time
	Payload        json.RawMessage
}

type Outcome string

const (
	Stored  Outcome = "stored"
	Ignored Outcome = "ignored"
	Failed  Outcome = "failed"
)

type Sink interface {
	// Accept returns only once the capture is durable.
	Accept(ctx context.Context, c Capture) (Outcome, error)
}
```

```go
type Mount interface {
	Post(pattern string, h http.HandlerFunc)
}

// Transport is a struct of funcs rather than an interface, because Send has to
// be nil-able and an interface cannot carry a nil method.
type Transport struct {
	Name string
	// Start begins receiving. Mount may be ignored by a transport that polls.
	Start func(ctx context.Context, sink squirrel.Sink, mount Mount) (stop func(context.Context) error, err error)
	// Send is nil when this transport cannot initiate a conversation.
	Send func(ctx context.Context, conversationID, text string) error
}
```

`Send` stays nil-able for the same reason it was nullable in TypeScript: a
transport that cannot initiate must say so in its type, not fail at the moment
it is needed. An interface method would force every transport to implement one
and return an error at call time — exactly the dishonest shape the original
design rejected. Go cannot express a nil method, so `Transport` is a struct of
funcs instead. That is the idiomatic way to say "this capability may be absent"
in Go, and it keeps the honesty the design is built on.

`Payload` is `json.RawMessage` — the original bytes, never re-marshalled. The
TypeScript version stored a parsed object and relied on round-tripping; keeping
the raw bytes is strictly closer to "store the raw text verbatim".

Nullable identifiers are `*string`. The fail-open path produces captures with no
ids at all, and Go's zero value for a string is `""`, which is a real value here
— an empty conversation id is not the same as an unknown one.

## HTTP

Standard library only. `http.ServeMux` with Go 1.22 method-and-path patterns
covers the two routes; there is no router dependency.

**The 404 trap carries over and must be handled.** `http.NotFound` writes
`404 page not found` with `Content-Type: text/plain; charset=utf-8` — the same
defect the last build found in Hono's default, with the same consequence: a
non-200 carrying a content type is uploaded into the Campfire room as a file
attachment, and the message is lost with no retry.

`ServeMux.Handler(r)` returns the matched pattern, empty when nothing matched, so
the mux is wrapped:

```go
func silentNotFound(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern == "" {
			w.WriteHeader(http.StatusNotFound) // no body, no Content-Type
			return
		}
		mux.ServeHTTP(w, r)
	})
}
```

The response table is unchanged from the phase-1 spec and is written explicitly
in the Campfire transport — `Content-Type` is set by hand for `stored` and
`failed`, and never touched for `ignored`.

`/healthz` still checks only that the spool directory is writable, and still
never touches Postgres, for the reason the phase-1 spec gives.

## Storage

`pgx/v5` directly, with hand-written SQL. The queries are an insert with an
`ON CONFLICT` against a partial index, a lookup, and an upsert; an ORM earns
nothing here and the last build lost a day to one.

`ON CONFLICT` keeps its predicate — the unique index is partial, and Postgres
rejects a conflict target whose predicate does not match:

```sql
insert into items (...) values (...)
on conflict (transport, external_id) where external_id is not null do nothing
```

Written as SQL there is nothing to get wrong about an ORM's argument names.

### Migrations

`embed.FS` over `internal/squirrel/migrations/*.sql`, applied by roughly forty
lines: read the files in lexical order, compare against a `schema_migrations`
table, apply each missing one inside a transaction that also records it.

No `goose`, no `golang-migrate`. Single replica, so advisory locking buys
nothing, and this removes the exact class of tool that produced the `drizzle-kit`
surprise.

### Error classification in the drain

The transient/permanent split is unchanged in meaning and better expressed:

```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
	permanent = strings.HasPrefix(pgErr.Code, "22") || strings.HasPrefix(pgErr.Code, "23")
}
```

plus `errors.Is(err, ErrMalformedSpoolFile)` for a file that will not parse.
Everything else is transient, keeping the phase-1 rule: when in doubt, defer.
Deferring costs a second; quarantining a good capture costs the thought.

This replaces a duck-typed property access on a caught value that the last build
shipped with a known, if unreachable, crash if anything ever threw a non-object.

## Durability

Unchanged in substance. `spool.Write` does create → write → `f.Sync()` →
close → `os.Rename` → **fsync the directory**. That last step was a fix late in
the TypeScript build and it is part of the design now, not an afterthought:

```go
d, err := os.Open(dir)
if err == nil {
	err = d.Sync()
	d.Close()
}
```

A host power loss on a Raspberry Pi without a UPS is the case it exists for.

`quarantine` gets the same directory sync, closing a gap the TypeScript version
parked.

## Configuration and logging

Same environment variables, same names, same validation rules — including
rejecting an unknown value in `TRANSPORTS`, which the last build had to add
after it was found to boot happily with zero transports and answer every webhook
with an attachment-producing 404.

Logging moves to `log/slog` with a JSON handler, replacing a hand-rolled
`JSON.stringify` line writer. Standard library, structured, and it is what the
homelab's log pipeline already expects from everything else.

## Shutdown

`signal.NotifyContext` for `SIGTERM` and `SIGINT`, then `srv.Shutdown(ctx)` with
a timeout, then close the pool. Go's `http.Server` drains in-flight requests as
part of `Shutdown`, which is the behaviour the TypeScript version had to
assemble by hand — and forgot to wire to a signal until review caught it.

Boot order is unchanged and is load-bearing: **listen first, touch Postgres
afterwards, in a goroutine, with retry.** A database outage at pod start must not
stop a capture being accepted, because Campfire does not retry.

## Testing

Standard `testing` plus `testify/require` for assertions. Postgres comes from
the same `docker run` container the TypeScript suite used, and the integration
tests still refuse to run without `TEST_DATABASE_URL` rather than skipping.

`testify` is a test-only dependency, never linked into the binary, and has been
stable on v1 for a decade. Noting that explicitly because adopting Go to shrink
a dependency surface and then adding a library looks like a contradiction, and
is not one — but the next reader deserves the reasoning rather than the
appearance.

**All 94 TypeScript cases port.** They are not transcription exercises; they
encode what review caught, and porting them is how the Go version inherits those
findings instead of rediscovering them. The ones that must survive intact:

| What it proves | Why it exists |
|---|---|
| Headerless response over a real socket | The `@hono/node-server` defect; a test that never crosses a socket cannot see it |
| An unrouted path returns no `Content-Type` | Same class, found separately in the router's default |
| Fail-open on a body parsing to `null` | `JSON.parse("null")` succeeded and the catch never fired |
| Two null-external-id rows coexist | The unique index is partial and must stay so |
| Redelivery inserts one row | The `ON CONFLICT` predicate matches the partial index |
| Unknown identity stores with a null person | A capture is never held hostage to knowing whose it was |
| Backoff grows on deferral and resets after success | The only drain mechanism that had no coverage |
| Serves and captures with Postgres unreachable | The founding claim of the whole design |

The Go equivalents of the two TypeScript-specific tests — the source-tree
boundary check and the fake-transport compile proof — are dropped. The compiler
enforces the first, and the second becomes an ordinary test double.

## Deployment

Multi-stage `Dockerfile`: `golang:1.26` to build with `CGO_ENABLED=0`, then a
distroless static base. No `node_modules` to copy, no `drizzle/` directory to
remember, no runtime to patch. The image goes from roughly 200MB to under 20MB
and the resident set from roughly 50MB to roughly 15MB.

CI keeps its shape: test on every push and pull request against a Postgres
service container, build and push `linux/amd64,linux/arm64` on `v*` tags only.
`go test ./...` replaces the two npm scripts; `go vet` and `gofmt -l` replace
Biome.

Kubernetes manifests still live in `ronaldlokers/homelab` and are still not part
of this repository. The port changes the image reference and lets the memory
request drop; nothing else there moves.

## What is not being ported

The deferred list from the phase-1 spec is unchanged and still binding: `?` and
`done` handling, the scheduler and firings, chores, the nightly clarification
pass, LLM parsing, Home Assistant sensor evidence, context triggers, any web UI,
authentication beyond the network boundary, and any transport other than
Campfire.

`Send` is ported — implemented, tested, and uncalled, still nil without a bot
key — for the reason the phase-1 spec gives: the asymmetry between answering and
initiating is real and transport-specific, and discovering it while building the
scheduler would mean reshaping the boundary and the thing using it at once.

## The TypeScript implementation

Stays on `main` as history. It is the reference for the Campfire contract: seven
findings are encoded in its tests, and where this port and that implementation
disagree about Campfire's behaviour, that implementation is the one that was
verified against the real payload shapes.
