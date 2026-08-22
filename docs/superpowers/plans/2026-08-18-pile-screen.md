# Phase 5b — The Pile Screen Implementation Plan

> **Executed.** A record of how the work was done, on the date in its name —
> not a description of what the product is now. Nothing here has to stay true;
> read `DESIGN.md`, `PRODUCT.md` and `docs/roadmap.md` for that.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the web screen for the pile — read, triage and search notes at `squirrel.ronaldlokers.nl`, behind Authentik forward-auth, rendering the approved comp.

**Architecture:** The screen is a transport and lives like one. A new `internal/web` package owns routing, HTML and assets; `internal/squirrel` gains only narrow data functions and never learns what HTML is. Every action is an ordinary form POST answered with a 303 redirect, so the screen works with JavaScript switched off; the card stamp, the hold, the undo and the keyboard are progressive enhancement layered on top of that. Authentication is not Squirrel's code: Traefik calls an Authentik outpost and passes an identity header, and Squirrel compares it to one configured value.

**Tech Stack:** Go 1.26, `html/template`, `embed`, `net/http` (`http.ServeMux` pattern routing), pgx/v5, testify. No frontend framework, no build step, no CDN.

**Spec:** `docs/superpowers/specs/2026-08-18-pile-design.md`

**Design:** `DESIGN.md` (normative for every visual decision), `.impeccable/design.json` (component snippets), and the approved comp at `.impeccable/comps/pile-first-viewport.html` (normative for markup structure, CSS and the enhancement script).

## Global Constraints

Copied from the spec and PRODUCT.md. Every task's requirements implicitly include this section.

- **Never a count.** No badge, no total, no "N to review", no page count, no percentage, in any form, on any surface. A capped list may say *that* there is more, never *how much* more.
- **Never a capture box.** The screen refuses to create an item. This is permanent, not a current limitation.
- **Every state transition is reversible, and repeating one is a no-op rather than an error.**
- **Newest first.** Oldest-first is a backlog you are behind on.
- **Search matches across every state**, including `kept`.
- **The core must not learn what HTML is.** Nothing under `internal/squirrel` may import `html/template`, `internal/web`, or reference a route, a form field or an element. `internal/web` imports `internal/squirrel`; the reverse is an import cycle and will not compile, which is what enforces the boundary.
- **The screen scrolls rather than paginating**, because a page count is a total in disguise.
- **Postgres is on this screen's request path and that is correct.** If the database is down the screen fails visibly with a 503 and nothing is lost.
- **Four states only:** `open`, `done`, `dropped`, `kept`. A note promoted to a chore becomes `done`; there is no `chore` state.
- **The two views must agree.** Anything the screen does to a note, the chat commands must see identically, and vice versa. Where both need the same behaviour, they call the same function.
- Go formatting and vetting are gates: `make check` must pass. Integration tests run under `make test-integration`, which sets `-tags=integration -p 1` and requires `TEST_DATABASE_URL`.

---

## File Structure

**Modified in `internal/squirrel` (data only, no HTML):**

- `notes.go` — `Item.State` filled on the read path; new `ItemByID`; new `PromoteItem`.
- `apply.go` — `Applier.promote` refactored to call `PromoteItem`, so chat and screen share one promotion path.
- `http.go` — new `Server.Get`, mirroring the existing `Server.Post`.
- `config.go` — three new fields for the screen.

**Created — `internal/web`, one responsibility per file:**

- `web.go` — `Mount`, the route table, and the `Store` interface the package needs. Nothing else.
- `auth.go` — the forward-auth identity guard. Only file that reads a request header for identity.
- `pile.go` — the `GET` handlers: the pile and search.
- `act.go` — the `POST` handlers: the four transitions and undo.
- `render.go` — template parsing, the view model, and the one place `html/template` is used.
- `assets.go` — embedded static files and their cache headers.
- `templates/layout.html`, `templates/pile.html`, `templates/results.html`, `templates/empty.html`, `templates/card.html`
- `static/pile.css`, `static/pile.js`, `static/recursive.woff2`, `static/inter-900.woff2`, `static/logo.png`

**Created — tests:**

- `internal/web/auth_test.go`, `internal/web/pile_test.go`, `internal/web/act_test.go`, `internal/web/render_test.go`, `internal/web/assets_test.go`, `internal/web/testsupport_test.go`

---

### Task 1: A note's state reaches the read path

The screen shows a state per note in search results. `itemsWhere` currently selects four columns and `state` is not one of them, so no caller can know what a note became.

**Files:**
- Modify: `internal/squirrel/store.go` (the `Item` struct, around line 44-60)
- Modify: `internal/squirrel/notes.go:92-130` (`itemsWhere`)
- Test: `internal/squirrel/notes_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `Item.State ItemState`, filled by `OpenItems` and `SearchItems`. Always one of `ItemOpen`, `ItemDone`, `ItemDropped`, `ItemKept`.

- [ ] **Step 1: Write the failing test**

Append to `internal/squirrel/notes_test.go`:

```go
func TestSearchItemsCarriesState(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	personID := seedPerson(t, store)

	open := insertNote(t, store, personID, "the boiler makes a noise")
	done := insertNote(t, store, personID, "boiler service is booked")
	require.NoError(t, store.SetItemState(ctx, done, squirrel.ItemDone, time.Now()))

	items, more, err := store.SearchItems(ctx, personID, "boiler", 10)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, items, 2)

	states := map[int64]squirrel.ItemState{}
	for _, it := range items {
		states[it.ID] = it.State
	}
	require.Equal(t, squirrel.ItemOpen, states[open])
	require.Equal(t, squirrel.ItemDone, states[done])
}
```

If `seedPerson` or `insertNote` do not already exist in this package's test support, read `internal/squirrel/notes_test.go` and reuse whatever the existing pile tests use to create a person and an item; do not add a second helper that does the same thing.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=integration -p 1 -run TestSearchItemsCarriesState ./internal/squirrel/`
Expected: FAIL — `states[open]` is `""`, not `"open"`, because nothing sets the field.

- [ ] **Step 3: Write minimal implementation**

In `internal/squirrel/store.go`, add to the `Item` struct, after `RawText`:

```go
	// State is filled on the read path only, by itemsWhere. The capture path
	// does not set it: a fresh row takes the column default, which is `open`,
	// and a capture that had to know about triage would be a capture that can
	// fail for a triage reason.
	State ItemState
```

In `internal/squirrel/notes.go`, change the query and the scan in `itemsWhere`:

```go
	q := `select id, raw_text, received_at, payload, state from items
	       where raw_text <> '' and ` + where +
		` order by received_at desc, id desc`
```

```go
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State); err != nil {
			return nil, false, fmt.Errorf("scanning item: %w", err)
		}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=integration -p 1 ./internal/squirrel/`
Expected: PASS, and every existing test in the package still passes.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/store.go internal/squirrel/notes.go internal/squirrel/notes_test.go
git commit -m "feat: carry a note's state on the read path"
```

---

### Task 2: One promotion path for both views

`Applier.promote` resolves a numbered chat line, then upserts a chore and marks the note done. The screen has an item id, not a line number, and must not re-implement the second half — two implementations of "a note becomes a chore" is exactly the disagreement Principle 4 forbids.

**Files:**
- Modify: `internal/squirrel/notes.go` (append)
- Modify: `internal/squirrel/apply.go:402-451` (`promote`)
- Test: `internal/squirrel/notes_test.go`

**Interfaces:**
- Consumes: `Item.State` from Task 1.
- Produces:
  - `func (s *Store) ItemByID(ctx context.Context, personID, itemID int64) (Item, bool, error)` — scoped to the person; `false` when the row does not exist or belongs to someone else.
  - `func (s *Store) PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (Chore, bool, error)` — `false` when the item does not exist for this person.

- [ ] **Step 1: Write the failing test**

Append to `internal/squirrel/notes_test.go`:

```go
func TestPromoteItemCreatesChoreAndClosesNote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	personID := seedPerson(t, store)
	itemID := insertNote(t, store, personID, "bins out")

	chore, ok, err := store.PromoteItem(ctx, personID, itemID, 14*24*time.Hour)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "bins out", chore.Name)

	items, _, err := store.OpenItems(ctx, personID, 10)
	require.NoError(t, err)
	require.Empty(t, items, "a promoted note leaves the pile")

	found, _, err := store.SearchItems(ctx, personID, "bins", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, squirrel.ItemDone, found[0].State,
		"a promoted note is recorded as done; there is no chore state")
}

func TestPromoteItemRefusesAnotherPersonsNote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	personID := seedPerson(t, store)
	itemID := insertNote(t, store, personID, "bins out")

	_, ok, err := store.PromoteItem(ctx, personID+1, itemID, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=integration -p 1 -run TestPromoteItem ./internal/squirrel/`
Expected: FAIL to compile — `store.PromoteItem` undefined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/squirrel/notes.go`:

```go
// ItemByID reads one note, scoped to its owner.
//
// The person is part of the lookup rather than checked afterwards. A handler
// that receives an id from a form has been handed a number by whoever is on
// the other end, and the only safe shape is a query that cannot return someone
// else's row in the first place.
func (s *Store) ItemByID(ctx context.Context, personID, itemID int64) (Item, bool, error) {
	var it Item
	var payload json.RawMessage
	err := s.pool.QueryRow(ctx, `
		select id, raw_text, received_at, payload, state from items
		 where id = $1 and person_id = $2`, itemID, personID).
		Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, fmt.Errorf("reading item: %w", err)
	}
	return it, true, nil
}

// PromoteItem turns a note into a chore: the note's own text becomes the
// chore's name, and the note becomes `done` — it did its job by turning into
// something that comes back on its own.
//
// Chore first, then the note. A failure between them leaves the chore created
// and the note still in the pile, so a second attempt upserts the same chore by
// name and finishes the job. The other order would leave a note marked done
// with no chore to show for it, which is the one of the two that loses
// something.
//
// Both the chat command and the screen call this. Two implementations of "a
// note becomes a chore" is the disagreement Principle 4 forbids, and the chat
// path already paid for the ordering above.
func (s *Store) PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (Chore, bool, error) {
	it, ok, err := s.ItemByID(ctx, personID, itemID)
	if err != nil || !ok {
		return Chore{}, false, err
	}
	c, err := s.UpsertChore(ctx, personID, it.RawText, every, DefaultTolerance(every))
	if err != nil {
		return Chore{}, false, err
	}
	if err := s.SetItemState(ctx, it.ID, ItemDone, time.Now()); err != nil {
		return Chore{}, false, err
	}
	return c, true, nil
}
```

Add `"errors"` and `"github.com/jackc/pgx/v5"` to that file's imports if they are not already there.

Now replace the tail of `Applier.promote` in `internal/squirrel/apply.go` so it goes through the same function. The block that currently reads:

```go
	c, err := a.store.UpsertChore(ctx, personID, line.Item.RawText, every, DefaultTolerance(every))
	if err != nil {
		return Message{}, err
	}
	// Chore first, then the note. A failure between them leaves the chore
	// created and the note still in the pile, so a second attempt upserts the
	// same chore by name and finishes the job. The other order would leave a
	// note marked done with no chore to show for it, which is the one of the
	// two that loses something.
	if err := a.store.SetItemState(ctx, line.Item.ID, ItemDone, time.Now()); err != nil {
		return Message{}, err
	}
	return Message{Text: RenderDefined(c)}, nil
```

becomes:

```go
	// The ordering argument that used to live here moved to PromoteItem, which
	// the screen calls too. One path, so the two views cannot disagree.
	c, ok, err := a.store.PromoteItem(ctx, personID, line.Item.ID, every)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return noSuchLine(n), nil
	}
	return Message{Text: RenderDefined(c)}, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=integration -p 1 ./internal/squirrel/`
Expected: PASS, including the existing `apply_promote_test.go`, unchanged. If any promotion test now fails, the refactor changed behaviour and the implementation is wrong, not the test.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/notes.go internal/squirrel/apply.go internal/squirrel/notes_test.go
git commit -m "feat: one promotion path for chat and screen"
```

---

### Task 3: The server can route a GET

`Server.Post` exists because every transport so far is a webhook. The screen is read-first and needs `GET`.

**Files:**
- Modify: `internal/squirrel/http.go:47-50`
- Test: `internal/squirrel/http_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func (s *Server) Get(pattern string, h http.HandlerFunc)`.

- [ ] **Step 1: Write the failing test**

Append to `internal/squirrel/http_test.go`:

```go
func TestGetRoutesAndPostToTheSamePathDoesNot(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	s.Get("/pile", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "the pile")
	})
	base := listen(t, s)

	res, err := http.Get(base + "/pile")
	require.NoError(t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "the pile", string(body))

	// A GET-only route must not answer a POST. The pattern carries the method.
	post, err := http.Post(base+"/pile", "text/plain", strings.NewReader(""))
	require.NoError(t, err)
	defer post.Body.Close()
	require.Equal(t, http.StatusNotFound, post.StatusCode)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestGetRoutes ./internal/squirrel/`
Expected: FAIL to compile — `s.Get` undefined.

- [ ] **Step 3: Write minimal implementation**

In `internal/squirrel/http.go`, directly below `Post`:

```go
// Get registers a read route. Separate from Post rather than a method-agnostic
// Handle: the method belongs in the pattern so that an unrouted method falls to
// the same no-body 404 as an unrouted path, and so no route can accidentally
// accept a write it never meant to.
func (s *Server) Get(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc("GET "+pattern, h)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/squirrel/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/http.go internal/squirrel/http_test.go
git commit -m "feat: route GET on the shared server"
```

---

### Task 4: Configuration for the screen

Three values, following `PresenceSecret`'s established shape: an empty required value means the route is not mounted at all, rather than mounted and half-working.

**Files:**
- Modify: `internal/squirrel/config.go` (the `Config` struct and the function that reads the environment)
- Test: `internal/squirrel/config_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `Config.WebPath string`, `Config.WebIdentityHeader string`, `Config.WebIdentity string`.

- [ ] **Step 1: Write the failing test**

Append to `internal/squirrel/config_test.go`. Read the top of that file first and match how the existing tests build their environment map and call the loader — reuse that helper rather than writing a second one.

```go
func TestWebDefaults(t *testing.T) {
	cfg, err := loadConfigFrom(t, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, "/pile", cfg.WebPath)
	require.Equal(t, "X-Authentik-Username", cfg.WebIdentityHeader)
	require.Empty(t, cfg.WebIdentity, "no identity means the screen is not mounted")
}

func TestWebIdentityIsTakenFromTheEnvironment(t *testing.T) {
	cfg, err := loadConfigFrom(t, map[string]string{
		"WEB_IDENTITY":        "ronald",
		"WEB_IDENTITY_HEADER": "X-Forwarded-User",
		"WEB_PATH":            "/notes",
	})
	require.NoError(t, err)
	require.Equal(t, "ronald", cfg.WebIdentity)
	require.Equal(t, "X-Forwarded-User", cfg.WebIdentityHeader)
	require.Equal(t, "/notes", cfg.WebPath)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestWeb ./internal/squirrel/`
Expected: FAIL to compile — `cfg.WebPath` undefined.

- [ ] **Step 3: Write minimal implementation**

Add to the `Config` struct in `internal/squirrel/config.go`, after `PresenceDelay`:

```go
	// WebIdentity is the value Authentik puts in WebIdentityHeader for the one
	// person who may read this. Empty means the screen is not mounted at all —
	// the same refusal PresenceSecret makes, and for a stronger reason: the
	// pile is every thought you have ever had at this bot, and a screen that
	// mounted without knowing who is allowed to read it would be open.
	WebIdentity string
	// WebIdentityHeader is the header Traefik's forward-auth middleware fills
	// from the Authentik outpost. Squirrel writes no authentication code and
	// holds no session; it compares one header to one configured value.
	WebIdentityHeader string
	// WebPath is where the screen is mounted.
	WebPath string
```

In the same file's environment-reading function, alongside the existing `Presence*` lines:

```go
	cfg.WebIdentity = env["WEB_IDENTITY"]
	cfg.WebIdentityHeader = optional(env, "WEB_IDENTITY_HEADER", "X-Authentik-Username")
	cfg.WebPath = optional(env, "WEB_PATH", "/pile")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/squirrel/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/config.go internal/squirrel/config_test.go
git commit -m "feat: configure the pile screen"
```

---

### Task 5: The identity guard

The only place in the codebase that decides whether a request may see the pile.

**Files:**
- Create: `internal/web/web.go`
- Create: `internal/web/auth.go`
- Test: `internal/web/auth_test.go`

**Interfaces:**
- Consumes: `Config.WebIdentity`, `Config.WebIdentityHeader` from Task 4.
- Produces:
  - `type Options struct { Path, IdentityHeader, Identity string; PersonID int64 }`
  - `type Store interface { ... }` — the narrow surface `internal/web` needs, declared here rather than imported, the same way `transport.Sink` is.
  - `func guard(opts Options, h http.HandlerFunc) http.HandlerFunc`

- [ ] **Step 1: Write the failing test**

Create `internal/web/auth_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOptions() Options {
	return Options{
		Path:           "/pile",
		IdentityHeader: "X-Authentik-Username",
		Identity:       "ronald",
		PersonID:       1,
	}
}

func TestGuardAllowsTheConfiguredIdentity(t *testing.T) {
	reached := false
	h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.Header.Set("X-Authentik-Username", "ronald")
	w := httptest.NewRecorder()
	h(w, r)

	require.True(t, reached)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestGuardRefusesEveryoneElse(t *testing.T) {
	for _, name := range []string{"", "someone", "Ronald "} {
		reached := false
		h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

		r := httptest.NewRequest("GET", "/pile", nil)
		if name != "" {
			r.Header.Set("X-Authentik-Username", name)
		}
		w := httptest.NewRecorder()
		h(w, r)

		require.False(t, reached, "handler ran for identity %q", name)
		require.Equal(t, http.StatusForbidden, w.Code, "identity %q", name)
		require.Empty(t, w.Body.String(), "a refusal says nothing about what is behind it")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/`
Expected: FAIL to compile — package does not exist.

- [ ] **Step 3: Write minimal implementation**

Create `internal/web/web.go`:

```go
// Package web is the screen, and it is a transport: it imports
// internal/squirrel and the reverse would be an import cycle, which is what
// keeps HTML out of the core.
//
// It is read-and-triage only. There is no route that creates an item and there
// never will be — two capture surfaces means two places to look for a thought,
// which is the problem this product exists to solve.
package web

import (
	"context"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Options is everything the screen needs to be mounted.
type Options struct {
	// Path is where the screen lives. Sub-routes hang off it.
	Path string
	// IdentityHeader is filled by Traefik's forward-auth middleware.
	IdentityHeader string
	// Identity is the one value that may read this pile. Mount refuses to
	// register a single route when it is empty.
	Identity string
	// PersonID is the owner. There is one person, resolved once at boot,
	// because SeedOwner already reconciles that identity every boot and a
	// second lookup per request would be a second source of truth.
	PersonID int64
}

// Store is the narrow surface the screen consumes. Declared here rather than
// imported: Go satisfies interfaces structurally, so *squirrel.Store fits this
// without either package importing the other's declaration, the same way
// transport.Sink does.
type Store interface {
	OpenItems(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	SearchItems(ctx context.Context, personID int64, query string, limit int) ([]squirrel.Item, bool, error)
	ItemByID(ctx context.Context, personID, itemID int64) (squirrel.Item, bool, error)
	SetItemState(ctx context.Context, itemID int64, state squirrel.ItemState, at time.Time) error
	PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (squirrel.Chore, bool, error)
}
```

Create `internal/web/auth.go`:

```go
package web

import (
	"log/slog"
	"net/http"
)

// guard is the whole of this product's authentication, and that is the point.
// Traefik calls an Authentik outpost, Authentik decides, and Squirrel compares
// one header to one configured value. No sessions, no cookies, no redirect
// flow, no OIDC library in a binary that has none of that anywhere else.
//
// The comparison is exact. Trimming or lower-casing the header would be this
// file quietly deciding that two identities are the same one.
func guard(opts Options, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who := r.Header.Get(opts.IdentityHeader)
		if who == "" || who != opts.Identity {
			// No body: a refusal that describes what it is refusing tells an
			// unauthenticated caller that there is something here.
			slog.Warn("refused the pile", "identity", who, "path", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/web/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/web/web.go internal/web/auth.go internal/web/auth_test.go
git commit -m "feat: guard the pile behind the forward-auth identity"
```

---

### Task 6: Render the pile

The screen with no JavaScript: the mark, the search field, one card with four form buttons, and the stack behind it.

**Files:**
- Create: `internal/web/render.go`
- Create: `internal/web/pile.go`
- Create: `internal/web/templates/layout.html`
- Create: `internal/web/templates/pile.html`
- Create: `internal/web/templates/card.html`
- Create: `internal/web/templates/empty.html`
- Test: `internal/web/testsupport_test.go`
- Test: `internal/web/pile_test.go`

**Interfaces:**
- Consumes: `Options`, `Store`, `guard` from Task 5; `Item.State` from Task 1.
- Produces:
  - `func Mount(m Mux, s Store, opts Options) error` where `type Mux interface { Get(pattern string, h http.HandlerFunc); Post(pattern string, h http.HandlerFunc) }`
  - `type view struct { Note *noteView; More bool; Query string; Results []noteView; Undo *undoView; Path string }`
  - `type noteView struct { ID int64; Text, When, State, StateWord, Tab string }`
  - `func render(w http.ResponseWriter, name string, v view)`

The cap is `pileLimit = 1` for the deck (one card at a time, plus the bool that says there is more) and `searchLimit = 6` for results.

- [ ] **Step 1: Write the failing test**

Create `internal/web/testsupport_test.go`:

```go
package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// fakeStore is an in-memory pile. The screen's own tests must not need
// Postgres: what is under test here is routing, rendering and the refusal to
// count, none of which is a database question. The store's own behaviour is
// covered by the integration tests in internal/squirrel.
type fakeStore struct {
	items []squirrel.Item
	err   error
}

func (f *fakeStore) OpenItems(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if it.State == squirrel.ItemOpen {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

func (f *fakeStore) SearchItems(_ context.Context, _ int64, q string, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if strings.Contains(strings.ToLower(it.RawText), strings.ToLower(q)) {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

func (f *fakeStore) ItemByID(_ context.Context, _ int64, id int64) (squirrel.Item, bool, error) {
	for _, it := range f.items {
		if it.ID == id {
			return it, true, nil
		}
	}
	return squirrel.Item{}, false, nil
}

func (f *fakeStore) SetItemState(_ context.Context, id int64, state squirrel.ItemState, _ time.Time) error {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].State = state
		}
	}
	return nil
}

func (f *fakeStore) PromoteItem(_ context.Context, _ int64, id int64, _ time.Duration) (squirrel.Chore, bool, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].State = squirrel.ItemDone
			return squirrel.Chore{Name: f.items[i].RawText}, true, nil
		}
	}
	return squirrel.Chore{}, false, nil
}

// testMux collects routes so a test can call one directly.
type testMux struct{ routes map[string]http.HandlerFunc }

func newTestMux() *testMux { return &testMux{routes: map[string]http.HandlerFunc{}} }

func (m *testMux) Get(pattern string, h http.HandlerFunc)  { m.routes["GET "+pattern] = h }
func (m *testMux) Post(pattern string, h http.HandlerFunc) { m.routes["POST "+pattern] = h }

func (m *testMux) call(t *testing.T, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	for pattern, h := range m.routes {
		wantMethod, path, _ := strings.Cut(pattern, " ")
		if wantMethod != method || !strings.HasPrefix(target, path) {
			continue
		}
		r := httptest.NewRequest(method, target, body)
		r.Header.Set("X-Authentik-Username", "ronald")
		if method == "POST" {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		w := httptest.NewRecorder()
		h(w, r)
		return w
	}
	t.Fatalf("no route for %s %s", method, target)
	return nil
}

func note(id int64, text string, state squirrel.ItemState) squirrel.Item {
	return squirrel.Item{ID: id, RawText: text, ReceivedAt: time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC), State: state}
}

func mounted(t *testing.T, f *fakeStore) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		Path: "/pile", IdentityHeader: "X-Authentik-Username", Identity: "ronald", PersonID: 1,
	}))
	return m
}
```

Add `"io"`, `"strings"` and `"github.com/stretchr/testify/require"` to that file's imports.

Create `internal/web/pile_test.go`:

```go
package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestPileShowsTheNewestOpenNote(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(1, "the boiler makes a noise on tuesdays", squirrel.ItemOpen),
		note(2, "buy milk", squirrel.ItemOpen),
		note(3, "boiler service is booked", squirrel.ItemDone),
	}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "the boiler makes a noise on tuesdays")
	require.NotContains(t, body, "boiler service is booked", "a triaged note is not in the pile")
}

func TestPileNeverEmitsACount(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 41; i++ {
		items = append(items, note(i, "note number "+strconv.FormatInt(i, 10), squirrel.ItemOpen))
	}
	body := mounted(t, &fakeStore{items: items}).call(t, "GET", "/pile", nil).Body.String()

	require.NotContains(t, body, "41")
	require.NotContains(t, body, "40")
	// The rule is about the fact, not the digit: the page may say there is more.
	require.Contains(t, strings.ToLower(body), "more")
}

func TestEmptyPileDoesNotCelebrate(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "nothing in the pile")
	for _, forbidden := range []string{"well done", "all done", "congrat", "🎉", "streak"} {
		require.NotContains(t, strings.ToLower(body), forbidden)
	}
}

func TestPileHasNoCaptureBox(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "kaas", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.NotContains(t, body, `name="text"`)
	require.NotContains(t, body, "<textarea")
	require.Equal(t, 1, strings.Count(body, "<input"),
		"exactly one input on this screen, and it is the search field")
}

func TestPileFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{err: errors.New("connection refused")}
	w := mounted(t, f).call(t, "GET", "/pile", nil)

	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), "cannot reach")
}
```

Add `"errors"` and `"strconv"` to that file's imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/`
Expected: FAIL to compile — `Mount` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/web/templates/layout.html`. Take the `<style>` block and the lid markup verbatim from `.impeccable/comps/pile-first-viewport.html`, with two changes: the inlined base64 font and logo become `<link>` and `<img src>` pointing at `{{.Path}}/static/...`, and the CSS moves to `static/pile.css` (Task 9). The skeleton:

```html
{{define "layout"}}<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<title>Squirrel</title>
<link rel="stylesheet" href="{{.Path}}/static/pile.css">
</head>
<body>
<div class="lid">
  <span class="brand">
    <img alt="" src="{{.Path}}/static/logo.png" width="84" height="63">
    <span class="wordmark">Squirrel</span>
  </span>
  <span class="spacer"></span>
  <form class="find" method="get" action="{{.Path}}" role="search">
    <svg width="17" height="17" viewBox="0 0 17 17" aria-hidden="true">
      <circle cx="7" cy="7" r="5" fill="none" stroke="#1c110b" stroke-width="2.6"/>
      <path d="M11 11 L15.5 15.5" stroke="#1c110b" stroke-width="2.6" stroke-linecap="round"/>
    </svg>
    <input type="search" name="q" value="{{.Query}}" placeholder="every note, any state" aria-label="Search every note">
  </form>
  <svg class="brim" viewBox="-1 0 102 30" preserveAspectRatio="none" aria-hidden="true">
    <path d="M-1 -2 L-1 5 Q 50 29 101 5 L101 -2 Z" fill="#3b2560"/>
    <path d="M-1 5 Q 50 29 101 5" fill="none" stroke="#1c110b" stroke-width="3" vector-effect="non-scaling-stroke"/>
  </svg>
</div>
<main id="stage"{{if .Query}} class="scrolling"{{end}}>
{{template "content" .}}
</main>
<script src="{{.Path}}/static/pile.js" defer></script>
</body>
</html>{{end}}
```

Create `internal/web/templates/card.html`, the four actions as real form submissions:

```html
{{define "card"}}
<article class="card" id="card" data-id="{{.Note.ID}}">
  <div class="titlebar">
    <svg class="acorn" width="16" height="19" viewBox="0 0 15 18" aria-hidden="true">
      <path d="M1.9 5.3 C1.9 2.5 4.4 1 7.5 1 C10.6 1 13.1 2.5 13.1 5.3 Z" fill="#fdecd4" stroke="#1c110b" stroke-width="1.7" stroke-linejoin="round"/>
      <path d="M2.7 6.3 H12.3 C12.3 11.9 10.2 16.4 7.5 16.4 C4.8 16.4 2.7 11.9 2.7 6.3 Z" fill="#fdecd4" stroke="#1c110b" stroke-width="1.7" stroke-linejoin="round"/>
    </svg>
    <span id="when">{{.Note.When}}</span>
    <span class="spacer"></span>
  </div>
  <div class="note"><p id="noteText">{{.Note.Text}}</p>
    <span class="stamp" id="stamp"><span class="dot"></span><span id="stampText"></span></span>
  </div>
  <div class="tray">
    <form class="row actions" id="actions" method="post" action="{{.Path}}/act">
      <input type="hidden" name="id" value="{{.Note.ID}}">
      <button class="btn" name="act" value="done" data-act="done">DONE <span class="key">D</span></button>
      <button class="btn" name="act" value="keep" data-act="keep">KEEP <span class="key">K</span></button>
      <button class="btn" name="act" value="drop" data-act="drop">DROP <span class="key">X</span></button>
      <button class="btn make" name="act" value="chore" data-act="chore" formaction="{{.Path}}/chore">MAKE A CHORE <span class="key">C</span></button>
    </form>
    <div class="row undoRow" id="undoRow" role="status" hidden>
      <button class="undo" id="undo" type="button">PUT IT BACK</button>
      <span class="said" id="said"></span>
    </div>
  </div>
</article>
{{end}}
```

Create `internal/web/templates/pile.html`:

```html
{{define "content"}}
<section class="shoebox" id="deckView">
  <div class="deck">
    {{if .More}}<div class="behind"></div><div class="behind"></div><div class="behind"></div>{{end}}
    {{template "card" .}}
  </div>
  <p class="hint"><span class="keysonly"><kbd>space</kbd> skips &middot; one key acts &middot; </span>stop whenever you like</p>
</section>
{{end}}
```

Create `internal/web/templates/empty.html`:

```html
{{define "content"}}
<section class="empty">
  <img alt="" src="{{.Path}}/static/logo.png" width="186" height="140">
  <h1>nothing in the pile</h1>
  <p>it fills up again on its own. search still finds everything you have ever said.</p>
</section>
{{end}}
```

Create `internal/web/render.go`:

```go
package web

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

//go:embed templates/*.html
var templateFS embed.FS

// Each page parses layout, card and exactly one content template. Go's
// templates are a flat namespace, so two files both defining "content" cannot
// live in one set — the set is the page.
var pages = map[string]*template.Template{
	"pile":    template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/card.html", "templates/pile.html")),
	"empty":   template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/empty.html")),
	"results": template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/results.html")),
}

type noteView struct {
	ID        int64
	Text      string
	When      string
	State     string
	StateWord string
}

type view struct {
	Path    string
	Query   string
	Note    *noteView
	More    bool
	Results []noteView
	Undo    *undoView
}

type undoView struct {
	ID    int64
	Said  string
	State string
}

// stateWords is the screen's half of the shared vocabulary. `open` is
// deliberately present: a search result still in the pile says so, and it wears
// Notebook Violet rather than one of the three exit colours.
var stateWords = map[squirrel.ItemState]string{
	squirrel.ItemOpen:    "IN THE PILE",
	squirrel.ItemDone:    "DONE",
	squirrel.ItemDropped: "DROPPED",
	squirrel.ItemKept:    "KEPT",
}

func toView(it squirrel.Item) noteView {
	return noteView{
		ID:        it.ID,
		Text:      it.RawText,
		When:      strings.ToUpper(it.ReceivedAt.Local().Format("2 January")),
		State:     string(it.State),
		StateWord: stateWords[it.State],
	}
}

func render(w http.ResponseWriter, name string, v view) {
	t, ok := pages[name]
	if !ok {
		panic("no such page: " + name)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Never cached. The pile is state, and a back button that showed a note you
	// already triaged would be the two views disagreeing with themselves.
	w.Header().Set("Cache-Control", "no-store")
	if err := t.ExecuteTemplate(w, "layout", v); err != nil {
		slog.Error("rendering the pile", "page", name, "error", err)
	}
}

// fail is what "the screen fails visibly and nothing is lost" looks like. The
// note is already durable; this is the exit, not the entrance.
func fail(w http.ResponseWriter, err error) {
	slog.Error("the pile could not be read", "error", err)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<title>Squirrel</title></head><body style="background:#58388a;color:#fffbf3;font:16px system-ui;padding:3rem">` +
		`<p>Squirrel cannot reach its memory right now. Nothing has been lost — everything you said is still there.</p>` +
		`</body></html>`))
}
```

Create `internal/web/pile.go`:

```go
package web

import (
	"fmt"
	"net/http"
	"strings"
)

// The deck shows one card. The second row is never rendered; it is read only so
// that "is there more" can be answered without a count, which is the same
// device OpenItems uses and for the same reason.
const pileLimit = 1

// Mux is the routing surface the screen needs from the shared server.
type Mux interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
}

// Mount registers the screen, or refuses. An empty identity is not a
// misconfiguration to warn about and continue past: the pile is every thought
// you have ever had at this bot.
func Mount(m Mux, s Store, opts Options) error {
	if opts.Identity == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_IDENTITY is empty")
	}
	if opts.PersonID == 0 {
		return fmt.Errorf("refusing to mount the pile: no owner")
	}
	m.Get(opts.Path, guard(opts, pileHandler(s, opts)))
	m.Post(opts.Path+"/act", guard(opts, actHandler(s, opts)))
	m.Post(opts.Path+"/chore", guard(opts, choreHandler(s, opts)))
	m.Get(opts.Path+"/static/", staticHandler(opts))
	return nil
}

func pileHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
			searchInto(w, r, s, opts, q)
			return
		}
		items, more, err := s.OpenItems(r.Context(), opts.PersonID, pileLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Path: opts.Path, More: more}
		if len(items) == 0 {
			render(w, "empty", v)
			return
		}
		n := toView(items[0])
		v.Note = &n
		render(w, "pile", v)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Search, `actHandler`, `choreHandler` and `staticHandler` do not exist yet, so add temporary compiling stubs in the files they will own — `internal/web/act.go` with `func actHandler(Store, Options) http.HandlerFunc { return func(http.ResponseWriter, *http.Request) {} }` and the same shape for `choreHandler`, `searchInto` and `staticHandler` — and a `templates/results.html` containing only `{{define "content"}}{{end}}`. Tasks 7, 8 and 9 replace each.

Run: `go test ./internal/web/`
Expected: PASS for the four pile tests.

- [ ] **Step 5: Commit**

```bash
git add internal/web/
git commit -m "feat: render the pile without javascript"
```

---

### Task 7: The four transitions and undo

Every action is a form POST answered with a 303 back to the pile, so the screen works with scripting off and a reload never repeats a write.

**Files:**
- Create: `internal/web/act.go` (replacing the stubs from Task 6)
- Test: `internal/web/act_test.go`

**Interfaces:**
- Consumes: `Store.SetItemState`, `Store.PromoteItem`, `Store.ItemByID`, `guard`, `view`, `undoView`.
- Produces: `func actHandler(s Store, opts Options) http.HandlerFunc`, `func choreHandler(s Store, opts Options) http.HandlerFunc`.

Form fields: `id` (int64), `act` (one of `done`, `keep`, `drop`, `open`), and on `/chore`, `every` (a string `ParseEvery` understands, e.g. `every 2 weeks`).

- [ ] **Step 1: Write the failing test**

Create `internal/web/act_test.go`:

```go
package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func post(t *testing.T, m *testMux, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return m.call(t, "POST", path, strings.NewReader(form.Encode()))
}

func TestActMovesTheNoteAndRedirects(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	w := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"done"}})

	require.Equal(t, 303, w.Code, "a write answers with See Other so a reload does not repeat it")
	require.Contains(t, w.Header().Get("Location"), "/pile")
	require.Equal(t, squirrel.ItemDone, f.items[0].State)
}

func TestEveryTransitionReverses(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"drop"}})
	require.Equal(t, squirrel.ItemDropped, f.items[0].State)

	post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"open"}})
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "back to the pile is a transition like any other")
}

func TestRepeatingATransitionIsANoOpNotAnError(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	first := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"keep"}})
	second := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"keep"}})

	require.Equal(t, 303, first.Code)
	require.Equal(t, 303, second.Code, "a retry is a state assertion, not a failure")
	require.Equal(t, squirrel.ItemKept, f.items[0].State)
}

func TestActRefusesAnUnknownAction(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	w := post(t, mounted(t, f), "/pile/act", url.Values{"id": {"1"}, "act": {"delete"}})

	require.Equal(t, 400, w.Code)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State)
}

func TestChorePromotesAndTheNoteBecomesDone(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	w := post(t, mounted(t, f), "/pile/chore", url.Values{"id": {"1"}, "every": {"every 2 weeks"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, squirrel.ItemDone, f.items[0].State, "there is no chore state")
}

func TestChoreRefusesAnIntervalItCannotRead(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	w := post(t, mounted(t, f), "/pile/chore", url.Values{"id": {"1"}, "every": {"every"}})

	require.Equal(t, 400, w.Code)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "an unreadable interval must not half-promote")
}
```

Add `"net/http/httptest"` to the imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/`
Expected: FAIL — the stub handlers write nothing, so `w.Code` is 200.

- [ ] **Step 3: Write minimal implementation**

Replace `internal/web/act.go`:

```go
package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The form's vocabulary, and the whole of it. A map rather than a switch so
// that an unknown action is a lookup miss instead of a default branch someone
// later fills in with something destructive.
var actions = map[string]squirrel.ItemState{
	"done": squirrel.ItemDone,
	"keep": squirrel.ItemKept,
	"drop": squirrel.ItemDropped,
	"open": squirrel.ItemOpen,
}

// back sends the browser to the pile with a 303. See Other and not 302: the
// method must become GET, so that a reload after triaging re-reads the pile
// instead of re-submitting the transition.
//
// The undo hint travels in the query string rather than a session, because
// this binary has no sessions and the screen is stateless by construction.
func back(w http.ResponseWriter, r *http.Request, opts Options, undo url.Values) {
	target := opts.Path
	if q := strings.TrimSpace(r.FormValue("q")); q != "" {
		undo.Set("q", q)
	}
	if len(undo) > 0 {
		target += "?" + undo.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func actHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		state, ok := actions[r.FormValue("act")]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		it, found, err := s.ItemByID(r.Context(), opts.PersonID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Writing the state a note already holds is a no-op rather than an
		// error; SetItemState says so itself, and this handler must not add a
		// check that turns a retry into a failure.
		if err := s.SetItemState(r.Context(), it.ID, state, time.Now()); err != nil {
			fail(w, err)
			return
		}
		back(w, r, opts, url.Values{
			"undo":  {strconv.FormatInt(it.ID, 10)},
			"was":   {string(it.State)},
			"state": {string(state)},
		})
	}
}

func choreHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// ParseEvery wants "every <interval> <name>" and returns the name from
		// the same string. Here the name is the note, so a word that is not a
		// unit is appended and only the duration is kept — the same sentinel
		// trick apply.go documents, and for the same reason: without it,
		// "every" alone borrows the next word as its unit and silently creates
		// a chore nobody asked for.
		_, every, ok := squirrel.ParseEvery(strings.TrimSpace(r.FormValue("every")) + " that")
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, found, err := s.PromoteItem(r.Context(), opts.PersonID, id, every)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		back(w, r, opts, url.Values{
			"undo":  {strconv.FormatInt(id, 10)},
			"was":   {string(squirrel.ItemOpen)},
			"state": {"chore"},
		})
	}
}
```

If `intervalSentinel` in `apply.go` is not the literal `that`, use whatever that constant holds and export nothing — copy the literal rather than exporting an internal detail.

Undo is not a new route: it posts to `/pile/act` with `act=open`, which is why `open` is in the `actions` map. The pile handler reads `undo`, `was` and `state` from the query in Task 8's view assembly and renders the undo row.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/web/ && make check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/web/act.go internal/web/act_test.go
git commit -m "feat: triage a note from the screen"
```

---

### Task 8: Search across every state

**Files:**
- Modify: `internal/web/pile.go` (replace the `searchInto` stub)
- Create: `internal/web/templates/results.html` (replacing the empty stub)
- Test: `internal/web/pile_test.go` (append)

**Interfaces:**
- Consumes: `Store.SearchItems`, `toView`, `view.Results`.
- Produces: `func searchInto(w http.ResponseWriter, r *http.Request, s Store, opts Options, q string)`; `const searchLimit = 6`.

- [ ] **Step 1: Write the failing test**

Append to `internal/web/pile_test.go`:

```go
func TestSearchCrossesEveryState(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(1, "the boiler makes a noise", squirrel.ItemOpen),
		note(2, "boiler service is booked", squirrel.ItemDone),
		note(3, "boiler insurance thing", squirrel.ItemDropped),
		note(4, "boiler meter reading 48213", squirrel.ItemKept),
	}}
	body := mounted(t, f).call(t, "GET", "/pile?q=boiler", nil).Body.String()

	require.Contains(t, body, "IN THE PILE")
	require.Contains(t, body, "DONE")
	require.Contains(t, body, "DROPPED")
	require.Contains(t, body, "KEPT")
}

func TestSearchSaysThereIsMoreWithoutSayingHowMuch(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 9; i++ {
		items = append(items, note(i, "boiler "+strconv.FormatInt(i, 10), squirrel.ItemOpen))
	}
	body := mounted(t, &fakeStore{items: items}).call(t, "GET", "/pile?q=boiler", nil).Body.String()

	require.Contains(t, strings.ToLower(body), "more")
	require.NotContains(t, body, "9 ")
	require.NotContains(t, body, "of 9")
}

func TestSearchWithNoHitsSaysSo(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile?q=boiler", nil).Body.String()

	require.Contains(t, body, "nothing says")
}

func TestSearchEscapesTheQuery(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{}}
	body := mounted(t, f).call(t, "GET", "/pile?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil).Body.String()

	require.NotContains(t, body, "<script>alert(1)</script>")
	require.Contains(t, body, "&lt;script&gt;")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/`
Expected: FAIL — `searchInto` is a stub and renders nothing.

- [ ] **Step 3: Write minimal implementation**

Create `internal/web/templates/results.html`:

```html
{{define "content"}}
<section class="results">
  {{if .Results}}
  <p class="resultsHead" aria-live="polite">EVERYTHING THAT SAYS <b>{{.Query}}</b></p>
  {{range .Results}}
  <article class="rcard state-{{.State}}">
    <span class="tab"></span>
    <div class="meta"><span>{{.When}}</span><span class="sep">&middot;</span><span class="state">{{.StateWord}}</span></div>
    <p>{{.Text}}</p>
    <form class="racts" method="post" action="{{$.Path}}/act">
      <input type="hidden" name="id" value="{{.ID}}">
      <input type="hidden" name="q" value="{{$.Query}}">
      {{if eq .State "open"}}
      <button class="rbtn" name="act" value="done">DONE</button>
      <button class="rbtn" name="act" value="keep">KEEP</button>
      <button class="rbtn" name="act" value="drop">DROP</button>
      <button class="rbtn make" name="act" value="chore" formaction="{{$.Path}}/chore">MAKE A CHORE</button>
      {{else}}
      <button class="rbtn back" name="act" value="open">BACK IN THE PILE</button>
      {{end}}
    </form>
  </article>
  {{end}}
  {{if .More}}<p class="more">there is more further back &mdash; keep typing to narrow it</p>{{end}}
  {{else}}
  <p class="noHits">nothing says &ldquo;{{.Query}}&rdquo;.</p>
  {{end}}
</section>
{{end}}
```

The state colour comes from the `state-{{.State}}` class, which `pile.css` maps to `--sc` and `--sct`; the comp sets those inline from JavaScript, and a server-rendered page has no JavaScript to do it. Add to `static/pile.css` in Task 9:

```css
.rcard.state-open    { --sc: var(--violet);  --sct: var(--violet-ink); }
.rcard.state-done    { --sc: var(--done);    --sct: var(--done-ink); }
.rcard.state-kept    { --sc: var(--kept);    --sct: var(--kept-ink); }
.rcard.state-dropped { --sc: var(--dropped); --sct: var(--dropped-ink); }
```

Note the `MAKE A CHORE` button in a result posts to `/chore` with no `every` field, which Task 7 answers with a 400. That is correct for the scriptless path and is the one place the screen needs the interval picker; the enhancement in Task 10 opens the picker instead of submitting. Add a `<details>` fallback so the scriptless path can still promote:

```html
      <details class="everyFallback"><summary class="rbtn make">MAKE A CHORE</summary>
        <button class="rbtn" name="act" value="chore" formaction="{{$.Path}}/chore" name="every" value="every day">every day</button>
      </details>
```

Do not use the line above — a `<button>` cannot carry two `name` attributes. Use separate buttons, each with its own `name="every"`:

```html
      <details class="everyFallback"><summary class="rbtn make">MAKE A CHORE</summary>
        <button class="rbtn" name="every" value="every day" formaction="{{$.Path}}/chore">every day</button>
        <button class="rbtn" name="every" value="every week" formaction="{{$.Path}}/chore">every week</button>
        <button class="rbtn" name="every" value="every 2 weeks" formaction="{{$.Path}}/chore">every 2 weeks</button>
        <button class="rbtn" name="every" value="every month" formaction="{{$.Path}}/chore">every month</button>
      </details>
```

Apply the same `<details>` treatment to the deck card's `MAKE A CHORE` in `templates/card.html`, replacing the single `act=chore` button.

Replace the `searchInto` stub in `internal/web/pile.go`:

```go
// searchLimit caps the result list. The cap is what makes "there is more"
// truthful; without it the line would appear over a complete list, which is a
// false claim in the one place the counting rule is most likely to leak.
const searchLimit = 6

func searchInto(w http.ResponseWriter, r *http.Request, s Store, opts Options, q string) {
	items, more, err := s.SearchItems(r.Context(), opts.PersonID, q, searchLimit)
	if err != nil {
		fail(w, err)
		return
	}
	v := view{Path: opts.Path, Query: q, More: more}
	for _, it := range items {
		v.Results = append(v.Results, toView(it))
	}
	render(w, "results", v)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/web/ && make check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/web/pile.go internal/web/templates/results.html internal/web/templates/card.html internal/web/pile_test.go
git commit -m "feat: search the pile across every state"
```

---

### Task 9: Assets, served and embedded

The comp inlines a 190KB font as base64 because a comp must be one file. A served page must not: the font, the mark and the stylesheet are separate cacheable files.

**Files:**
- Create: `internal/web/assets.go` (replacing the stub)
- Create: `internal/web/static/pile.css`
- Create: `internal/web/static/recursive.woff2`
- Create: `internal/web/static/inter-900.woff2`
- Create: `internal/web/static/logo.png`
- Test: `internal/web/assets_test.go`

**Interfaces:**
- Consumes: `Options.Path`.
- Produces: `func staticHandler(opts Options) http.HandlerFunc`.

- [ ] **Step 1: Fetch the assets**

```bash
mkdir -p internal/web/static
curl -sS -o internal/web/static/recursive.woff2 \
  "https://fonts.gstatic.com/s/recursive/v44/8vIk7wMr0mhh-RQChyHEH06JlTRq_gukbYr6Mk1QmnhkyU8G_X-dm23WO1w.woff2"
curl -sS -o internal/web/static/inter-900.woff2 \
  "https://fonts.gstatic.com/s/inter/v20/UcCO3FwrK3iLTeHuS_nVMrMxCp50SjIw2boKoduKmMEVuBWYAZ9hiA.woff2"
magick assets/logo.png -trim +repage -resize x280 -colors 48 -strip internal/web/static/logo.png
```

Both fonts are SIL Open Font License; record that in `internal/web/static/OFL.txt` by fetching `https://raw.githubusercontent.com/arrowtype/recursive/main/OFL.txt` and `https://raw.githubusercontent.com/rsms/inter/master/LICENSE.txt` into one file with a header naming which is which.

- [ ] **Step 2: Write the failing test**

Create `internal/web/assets_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticServesTheStylesheetWithALongCache(t *testing.T) {
	h := staticHandler(Options{Path: "/pile"})
	r := httptest.NewRequest("GET", "/pile/static/pile.css", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/css")
	require.Contains(t, w.Header().Get("Cache-Control"), "max-age=")
	require.Contains(t, w.Body.String(), "--card: #fdecd4")
}

func TestStaticDoesNotEscapeItsDirectory(t *testing.T) {
	h := staticHandler(Options{Path: "/pile"})
	r := httptest.NewRequest("GET", "/pile/static/../../etc/passwd", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestFontsAreEmbedded(t *testing.T) {
	for _, name := range []string{"recursive.woff2", "inter-900.woff2", "logo.png"} {
		b, err := staticFS.ReadFile("static/" + name)
		require.NoError(t, err, name)
		require.NotEmpty(t, b, name)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -run TestStatic ./internal/web/`
Expected: FAIL — `staticFS` undefined and the stub handler returns 200 with no headers.

- [ ] **Step 4: Write the implementation**

Create `internal/web/static/pile.css` by copying the entire contents of the `<style>` element in `.impeccable/comps/pile-first-viewport.html`, with exactly these changes:

1. The two `@font-face` blocks point at files instead of data URIs:

```css
  @font-face {
    font-family: "Recursive";
    src: url("recursive.woff2") format("woff2");
    font-weight: 300 900;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Inter";
    src: url("inter-900.woff2") format("woff2");
    font-weight: 900;
    font-style: normal;
    font-display: swap;
  }
```

`font-display: swap`, not the comp's `block`: a comp is screenshotted and a screen is read, and a blocked first paint on a phone in poor conditions is worse than one reflow.

2. Append the four `.rcard.state-*` rules from Task 8.

Everything else — every token, every component rule, both media queries — is copied unchanged. `DESIGN.md` is normative and the comp is its source; do not re-derive any value.

Create `internal/web/assets.go`:

```go
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFS embed.FS

// staticHandler serves the stylesheet, the script, the two fonts and the mark.
//
// A year of caching with no fingerprint in the filename is deliberate and
// bounded: this is one screen behind an ipAllowList used by one person, and a
// changed asset is one hard reload away. Fingerprinting would mean a build
// step, and this binary has none.
func staticHandler(opts Options) http.HandlerFunc {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	prefix := strings.TrimSuffix(opts.Path, "/") + "/static/"
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.StripPrefix(prefix, files).ServeHTTP(w, r)
	}
}
```

`http.FileServer` cleans `..` out of the path itself, which is what the traversal test pins.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/web/ && make check`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/web/assets.go internal/web/static/ internal/web/assets_test.go
git commit -m "feat: serve the screen's fonts, stylesheet and mark"
```

---

### Task 10: The enhancement layer

Everything works without this file. This file makes it feel like the comp: the stamp, the hold, the undo on the card, and one key per action.

**Files:**
- Create: `internal/web/static/pile.js`
- Test: `internal/web/render_test.go`

**Interfaces:**
- Consumes: the rendered markup from Tasks 6-8; `POST {Path}/act` and `POST {Path}/chore`.
- Produces: no Go API.

- [ ] **Step 1: Write the failing test**

The script is not executed by Go tests; what Go can pin is that the markup it needs is present and that the scriptless path does not depend on it. Create `internal/web/render_test.go`:

```go
package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheCardCarriesWhatTheScriptNeeds(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	for _, hook := range []string{`id="card"`, `data-id="1"`, `id="stamp"`, `id="undoRow"`, `data-act="done"`} {
		require.Contains(t, body, hook, "the script hangs off %s", hook)
	}
}

func TestEveryActionIsAFormSubmissionNotAScriptHook(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `method="post"`)
	require.Contains(t, body, `action="/pile/act"`)
	require.NotContains(t, body, "onclick=",
		"behaviour lives in pile.js; a page that needs inline handlers is a page that fails without them")
	require.Equal(t, strings.Count(body, `name="act"`), strings.Count(body, "<button class=\"btn\""),
		"every action button submits a value the server understands")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run "TestTheCard|TestEveryAction" ./internal/web/`
Expected: FAIL if any hook or form attribute is missing from the templates. Fix the templates, not the test.

- [ ] **Step 3: Write the implementation**

Create `internal/web/static/pile.js`, adapted from the comp's script. The comp holds its pile in a JavaScript array; the screen holds it in Postgres, so the script's job changes: it intercepts the form submission, plays the stamp and the hold, and then lets the form go through. Everything about the two voices, the timings and the key map is unchanged from the comp.

```js
// Progressive enhancement, and nothing here is load-bearing. Every action on
// this page is a form submission that works with this file absent; what this
// adds is the stamp, the moment the card holds still so the undo has somewhere
// to be, and one key per action.
(() => {
  const card = document.getElementById("card");
  if (!card) return;                      // the empty pile, or a search
  const form = document.getElementById("actions");
  const stamp = document.getElementById("stamp");
  const stampText = document.getElementById("stampText");
  const undoRow = document.getElementById("undoRow");
  const said = document.getElementById("said");
  const find = document.querySelector(".find input");

  const STATES = {
    done:  { word: "DONE",    said: "marked done" },
    keep:  { word: "KEPT",    said: "kept as reference" },
    drop:  { word: "DROPPED", said: "dropped" },
    chore: { word: "CHORE",   said: "now a chore" }
  };

  // Habituation is the enemy: the stack never sits the same way twice.
  document.querySelectorAll(".behind").forEach((el, n) => {
    const rot = (n % 2 ? -1 : 1) * (0.7 + Math.random() * 1.3);
    const step = 1 + n * 0.8 + Math.random() * 0.5;
    el.style.transform =
      `translate(calc(var(--o) * ${step.toFixed(2)}), calc(var(--o) * ${(step * 1.1).toFixed(2)})) rotate(${rot.toFixed(2)}deg)`;
  });

  let going = false;

  // Act, then hold, then submit. The delay is not decoration: it is the moment
  // the spec asks for, and the undo has to be reachable while the card that it
  // undoes is still on the screen.
  function go(button) {
    if (going) return;
    going = true;
    const kind = button.dataset.act || "done";
    const s = STATES[kind] || STATES.done;
    stamp.style.setProperty("--sc", `var(--${kind === "keep" ? "kept" : kind === "drop" ? "dropped" : kind})`);
    stamp.style.setProperty("--sct", `var(--${kind === "keep" ? "kept" : kind === "drop" ? "dropped" : kind}-ink)`);
    stampText.textContent = s.word;
    said.textContent = s.said;
    card.classList.add("stamped");
    form.hidden = true;
    undoRow.hidden = false;
    document.getElementById("undo").focus({ preventScroll: true });

    setTimeout(() => {
      card.classList.add("leaving");
      setTimeout(() => {
        // Hand the real submission back to the browser. The server does the
        // write and answers 303, so a reload never repeats it.
        button.form.requestSubmit(button);
      }, 440);
    }, 1150);
  }

  form.addEventListener("click", e => {
    const b = e.target.closest("button[name=act], button[name=every]");
    if (!b || going) return;
    e.preventDefault();
    go(b);
  });

  // PUT IT BACK is the transition back to open, posted like any other.
  document.getElementById("undo").addEventListener("click", () => {
    const back = document.createElement("form");
    back.method = "post";
    back.action = form.action;
    back.innerHTML =
      `<input name="id" value="${card.dataset.id}"><input name="act" value="open">`;
    document.body.append(back);
    back.submit();
  });

  // Letters are actions, always. Movement is space and the arrows, because in a
  // one-card topology j/k has nothing to move between and k is keep.
  const KEYS = { d: "done", k: "keep", x: "drop", c: "chore" };
  addEventListener("keydown", e => {
    if (e.target === find) {
      if (e.key === "Escape") { find.value = ""; find.form.submit(); }
      return;
    }
    if (e.key === "/") { e.preventDefault(); find.focus(); return; }
    // A focused control owns space and enter; that is the platform's contract.
    if ((e.key === " " || e.key === "Enter") && e.target.closest("button")) return;
    if (going) return;
    const act = KEYS[e.key.toLowerCase()];
    if (act) {
      const b = form.querySelector(`[data-act="${act}"]`);
      if (b) { e.preventDefault(); go(b); }
    }
  });
})();
```

The comp's `space skips` has no server-side equivalent yet — the deck renders whichever note is newest and there is no cursor. Leave `space` unbound and change the hint in `templates/pile.html` to `<span class="keysonly">one key acts &middot; </span>stop whenever you like`, so the page does not advertise a key that does nothing. Skipping is deferred; note it in the follow-ups below.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/web/ && make check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/web/static/pile.js internal/web/templates/pile.html internal/web/render_test.go
git commit -m "feat: stamp, hold and one key per action"
```

---

### Task 11: Boot the screen

**Files:**
- Modify: `internal/boot/boot.go` (near the `MountPresence` block, around lines 120-175)
- Test: `internal/boot/boot_pile_test.go`

**Interfaces:**
- Consumes: `web.Mount`, `web.Options`, `Config.Web*`, the `personID` boot already resolves at line 208.
- Produces: the screen, mounted.

- [ ] **Step 1: Write the failing test**

Create `internal/boot/boot_pile_test.go`, matching the harness the existing `boot_test.go` files use to start a booted instance and hit it over a real socket:

```go
//go:build integration

package boot_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheScreenIsBehindTheIdentityHeader(t *testing.T) {
	base := bootWith(t, map[string]string{"WEB_IDENTITY": "ronald"})

	anon, err := http.Get(base + "/pile")
	require.NoError(t, err)
	defer anon.Body.Close()
	require.Equal(t, http.StatusForbidden, anon.StatusCode)

	req, err := http.NewRequest("GET", base+"/pile", nil)
	require.NoError(t, err)
	req.Header.Set("X-Authentik-Username", "ronald")
	ok, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer ok.Body.Close()
	require.Equal(t, http.StatusOK, ok.StatusCode)
}

func TestTheScreenIsNotMountedWithoutAnIdentity(t *testing.T) {
	base := bootWith(t, map[string]string{})

	res, err := http.Get(base + "/pile")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode,
		"no identity, no route — not an open route that warns")
}
```

Read `internal/boot/boot_test.go` first and reuse its existing boot helper; if it does not take an environment override map, extend that helper rather than writing a second one.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=integration -p 1 -run TestTheScreen ./internal/boot/`
Expected: FAIL — 404 on both, because nothing mounts the screen.

- [ ] **Step 3: Write minimal implementation**

In `internal/boot/boot.go`, after the `personID` is resolved and before `server.Listen`, add:

```go
	// The screen mounts after the owner is known, because Options carries the
	// person rather than resolving one per request. An empty WEB_IDENTITY is
	// not a warning to continue past: web.Mount refuses, and the route simply
	// does not exist, which is the same shape MountPresence uses for its own
	// secret.
	if config.WebIdentity != "" {
		if err := web.Mount(server, store, web.Options{
			Path:           config.WebPath,
			IdentityHeader: config.WebIdentityHeader,
			Identity:       config.WebIdentity,
			PersonID:       personID,
		}); err != nil {
			return nil, fmt.Errorf("mounting the pile: %w", err)
		}
		slog.Info("the pile is mounted", "path", config.WebPath)
	}
```

Add `"github.com/ronaldlokers/squirrel/internal/web"` to the imports. If `personID` is resolved after the current `Listen` call, move the mount to just after the resolution and before `Listen` — a route registered after `Listen` races the first request, which is the reason the file's existing comment gives for mounting early.

- [ ] **Step 4: Run test to verify it passes**

Run: `make test-integration && make check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/boot/boot.go internal/boot/boot_pile_test.go
git commit -m "feat: mount the pile screen at boot"
```

---

### Task 12: Document the deployment

The Authentik outpost and the Traefik middleware are infrastructure, not Go, and they are the half of this phase that no test covers.

**Files:**
- Modify: `README.md`
- Create: `docs/pile-screen.md`

- [ ] **Step 1: Write the document**

Create `docs/pile-screen.md` covering, with exact values:

- The three environment variables from Task 4, their defaults, and the fact that an empty `WEB_IDENTITY` leaves the route unmounted.
- The Traefik middleware: a `forwardAuth` calling the Authentik outpost's `/outpost.goauthentik.io/auth/traefik` endpoint, with `authResponseHeaders` including `X-Authentik-Username`, chained *after* the existing phase 4 `ipAllowList` so LAN-only remains the outer layer.
- That Squirrel writes no authentication code, holds no session and sets no cookie, and that this is deliberate: not app-level OIDC, which would add sessions, cookies and a redirect flow to a binary that has none.
- That the screen is read-and-triage permanently, and there is no route that creates an item.
- How to hard-reload after an asset change, given Task 9's year-long cache with no fingerprint.

Add a line to `README.md` pointing at it, in the same style as the existing entries.

- [ ] **Step 2: Commit**

```bash
git add README.md docs/pile-screen.md
git commit -m "docs: deploying the pile screen"
```

---

## Deferred

Recorded so they are decisions rather than omissions:

- **Skipping a note.** The comp binds `space` to advance the deck; the screen has no cursor, so the key is unbound and the hint no longer advertises it. Adding it means a position in the query string or a `skipped` set, and neither is worth inventing before the screen has been used.
- **The chore interval picker as the comp draws it.** Task 8 ships a `<details>` disclosure with four fixed intervals. The comp's in-place chip row, with keys 1-4, is an enhancement Task 10 does not add.
- **Live search.** The comp searches as you type. This screen submits the form. Debounced fetch-and-replace is a later task and needs no server change.

## Self-Review

**Spec coverage.** Lifecycle — Tasks 1, 7. `!` commands and chat search — already shipped in 5a, out of scope. Promotion to a chore — Task 2, 7, shared with chat. The screen — Tasks 3, 5, 6, 8, 9, 10, 11. Authentication — Tasks 5, 11, 12. Newest first — inherited from `OpenItems`, pinned by Task 6's test. Undo on the screen — Task 7 (`act=open`) and Task 10 (on the card). Keyboard-first — Task 10. Search on the same screen — Task 8. Postgres on the request path with visible failure — Task 6's `fail`. No count — pinned by tests in Tasks 6 and 8. No capture box — pinned by a test in Task 6. Reversible and idempotent — pinned by two tests in Task 7. `kept` searchable — Task 8's first test. The screen refuses to create an item — Task 6's test counts the inputs.

**Gap found and closed:** the spec's "the screen refuses to create an item" had no test until the input-count assertion was added to Task 6.

**Not a gap:** the spec's first draft asked for `j`/`k` movement, which collided with `k` for keep. The spec has been amended — letters are actions, movement is `space` and the arrows — so the plan, the comp, `DESIGN.md` and the spec now agree. Skipping entirely is in Deferred above.

**Type consistency.** `Item.State` (Task 1) is read by `toView` (Task 6) and the fake store (Task 6). `PromoteItem(ctx, personID, itemID, every) (Chore, bool, error)` (Task 2) is called by `choreHandler` (Task 7) and by `Applier.promote` (Task 2), with the same three return values in both. `ItemByID(ctx, personID, itemID) (Item, bool, error)` (Task 2) is called by `PromoteItem` (Task 2) and `actHandler` (Task 7). `Options` (Task 5) is constructed in Task 11 with exactly its four fields. `Mux` (Task 6) is satisfied by `*squirrel.Server` once Task 3 adds `Get`, and by `testMux` (Task 6). `staticHandler(opts Options)` (Task 9) is registered in Task 6's `Mount`.
