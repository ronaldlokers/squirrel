# The Thread, Phase 1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the home screen with a persistent conversation at `/` — a pinned rail of four counted doors, a transcript stored in Postgres, the check-in and the offer as turns, and a JavaScript module that swaps in new turns without a page paint.

**Architecture:** One new table (`turns`) holding what was said, as text. `/` renders the most recent turns plus the rail; every press posts a form, the handler writes turns and returns *the same HTML the page would have rendered* for just the new turns; a small module appends the fragment and scrolls. No JSON, no client templating — handlers return HTML from the existing Go templates in both modes.

**Tech Stack:** Go 1.26, pgx v5, `html/template`, embedded migrations (`embed.FS`), vanilla ES modules. Tests: `go test` with build tags `integration` (needs `TEST_DATABASE_URL`) and `browser` (needs chromium, driven over CDP by `internal/web/cdp_test.go`).

**Spec:** `docs/superpowers/specs/2026-08-24-the-thread-design.md` — read it before Task 1. Phase 1 is the first entry under *Staging*.

## Global Constraints

- **Counts are permitted everywhere.** Principle 2 of `PRODUCT.md` was retired by the owner on 24 August 2026. Do not add a test asserting the absence of a number, and delete any you find that guards the old rule on a surface this plan touches.
- **JavaScript is required.** The progressive-enhancement requirement in `DESIGN.md` was retired the same day. Do not write a no-script fallback path.
- **One rendering path.** Handlers return HTML from `internal/web/templates/*`. No JSON endpoint, no client-side template, no second description of a card.
- **History is never rewritten.** A stored turn holds text, never a foreign key it re-reads at render time.
- **Only the newest Buddy turn carries controls.** Older turns render without buttons.
- **Zero renders as no number**, never as `0`.
- **The One Title Rule survives as `<h2>` per turn-that-opens-a-place.** The thread itself has no `<h1>`, on home's existing exemption.
- **Mutation proof is required for every test.** Before claiming a test passes, revert the behaviour it covers and record the exact assertion-failure text you observed. A compile error (`undefined: Foo`) is **not** proof — the failure must be an assertion failing in code that compiles and runs. For a wiring test, the mutation is reverting the wiring, not deleting the function.
- **Never commit to `main`.** Work on `feat/the-thread`, which already exists and holds the spec.
- Time comes from `now()` in `internal/web` and is passed explicitly into store calls. Do not call `time.Now()` in a handler.

---

## File Structure

**Created**

| file | responsibility |
| --- | --- |
| `internal/squirrel/migrations/0027_turns.sql` | the table |
| `internal/squirrel/turns.go` | `Turn`, `AppendTurn`, `RecentTurns`, `TurnsBefore` |
| `internal/squirrel/turns_test.go` | integration tests for the above |
| `internal/squirrel/waiting.go` | `Waiting`, the four door numbers |
| `internal/squirrel/waiting_test.go` | integration tests for the above |
| `internal/web/thread.go` | `threadHandler`, `threadSayHandler`, turn views, rail view |
| `internal/web/thread_test.go` | handler tests |
| `internal/web/templates/thread.html` | the page: rail, thread, dock |
| `internal/web/templates/turn.html` | one turn — also rendered alone as a fragment |
| `internal/web/static/thread.js` | intercept, post, append, scroll, announce |

**Modified**

| file | change |
| --- | --- |
| `internal/web/web.go:136` | `Store` gains four methods |
| `internal/web/render.go:44` | `pages` gains `thread`; `view` gains `Turns`, `Rail`, `Live` |
| `internal/web/pile.go:39` | `/` routes to the thread; `/say` added; `/buddy` routes removed |
| `internal/web/templates/layout.html` | the acorn goes; `thread.js` is loaded |
| `internal/web/static/pile.css` | the rail, the thread, the turns, the dock |
| `PRODUCT.md`, `DESIGN.md` | the two retirements, dated |

**Deleted**

`internal/web/home.go`, `internal/web/templates/home.html`, `internal/web/templates/askbuddy.html`, `internal/web/coach.go`'s page routes (`/buddy`, `/coach`) and `internal/web/templates/coach.html`. `coachHandler`'s model plumbing stays and is reached from the thread.

---

### Task 1: The turns table and its store

**Files:**
- Create: `internal/squirrel/migrations/0027_turns.sql`
- Create: `internal/squirrel/turns.go`
- Test: `internal/squirrel/turns_test.go`

**Interfaces:**
- Consumes: `*squirrel.Store` (pgx pool), the existing migration runner which reads `migrations/*.sql` in filename order.
- Produces:
  ```go
  type Speaker string
  const (SpeakerBuddy Speaker = "buddy"; SpeakerYou Speaker = "you")

  type Turn struct {
      ID     int64
      Who    Speaker
      Words  string
      Shown  []byte    // JSON, nil for a turn that is only a sentence
      SaidAt time.Time
  }

  func (s *Store) AppendTurn(ctx context.Context, personID int64, t Turn) (Turn, error)
  func (s *Store) RecentTurns(ctx context.Context, personID int64, limit int) ([]Turn, bool, error)
  func (s *Store) TurnsBefore(ctx context.Context, personID, beforeID int64, limit int) ([]Turn, bool, error)
  ```
  `RecentTurns` and `TurnsBefore` return **oldest-first** slices — the order they are rendered in — and a `bool` meaning *there is more above*.

- [ ] **Step 1: Write the failing tests**

Create `internal/squirrel/turns_test.go`:

```go
//go:build integration

package squirrel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestAppendTurnComesBack(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	got, err := store.AppendTurn(ctx, personID, squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "at 14:30 dentist",
	})
	require.NoError(t, err)
	require.NotZero(t, got.ID)

	turns, more, err := store.RecentTurns(ctx, personID, 10)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, turns, 1)
	require.Equal(t, "at 14:30 dentist", turns[0].Words)
	require.Equal(t, squirrel.SpeakerYou, turns[0].Who)
}

// The order the thread reads in. A newest-first slice would render the
// conversation backwards, and a test that only counts rows would not notice.
func TestRecentTurnsAreOldestFirst(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	for _, w := range []string{"first", "second", "third"} {
		_, err := store.AppendTurn(ctx, personID, squirrel.Turn{Who: squirrel.SpeakerYou, Words: w})
		require.NoError(t, err)
	}

	turns, _, err := store.RecentTurns(ctx, personID, 10)
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second", "third"},
		[]string{turns[0].Words, turns[1].Words, turns[2].Words})
}

// The cap keeps the newest, not the oldest — a limit applied before the sort
// would show the beginning of the conversation forever.
func TestRecentTurnsKeepsTheNewestAndSaysThereIsMore(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := store.AppendTurn(ctx, personID,
			squirrel.Turn{Who: squirrel.SpeakerYou, Words: fmt.Sprintf("turn %d", i)})
		require.NoError(t, err)
	}

	turns, more, err := store.RecentTurns(ctx, personID, 2)
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, []string{"turn 3", "turn 4"}, []string{turns[0].Words, turns[1].Words})
}

func TestTurnsBeforePagesBackwards(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	var ids []int64
	for i := 0; i < 5; i++ {
		got, err := store.AppendTurn(ctx, personID,
			squirrel.Turn{Who: squirrel.SpeakerYou, Words: fmt.Sprintf("turn %d", i)})
		require.NoError(t, err)
		ids = append(ids, got.ID)
	}

	turns, more, err := store.TurnsBefore(ctx, personID, ids[3], 2)
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, []string{"turn 1", "turn 2"}, []string{turns[0].Words, turns[1].Words})
}

// Shown is stored as written and is never re-read from another table. This is
// the whole of "history is never rewritten" at the storage layer.
func TestShownIsKeptVerbatim(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	_, err := store.AppendTurn(ctx, personID, squirrel.Turn{
		Who: squirrel.SpeakerBuddy, Words: "Two are due.",
		Shown: []byte(`{"cards":[{"name":"water the plants"}]}`),
	})
	require.NoError(t, err)

	turns, _, err := store.RecentTurns(ctx, personID, 10)
	require.NoError(t, err)
	require.JSONEq(t, `{"cards":[{"name":"water the plants"}]}`, string(turns[0].Shown))
}

// Another person's conversation is not yours.
func TestRecentTurnsAreScopedToThePerson(t *testing.T) {
	store, personID := freshStore(t)
	other := anotherPerson(t, store)
	ctx := context.Background()

	_, err := store.AppendTurn(ctx, other, squirrel.Turn{Who: squirrel.SpeakerYou, Words: "not yours"})
	require.NoError(t, err)

	turns, _, err := store.RecentTurns(ctx, personID, 10)
	require.NoError(t, err)
	require.Empty(t, turns)
}
```

If `freshStore` and `anotherPerson` do not already exist in `package squirrel_test`, find the equivalents the existing integration tests use (`internal/squirrel/notes_test.go` opens the store and makes a person at the top of the file) and reuse those helpers by their real names rather than adding duplicates.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
TEST_DATABASE_URL=... go test -tags integration ./internal/squirrel/ -run 'Turn' -v
```

Expected: compile failure, `undefined: squirrel.Turn`. **This is not yet proof** — it only shows the symbol is missing. The proof comes in Step 5.

- [ ] **Step 3: Write the migration**

Create `internal/squirrel/migrations/0027_turns.sql`:

```sql
-- Everything that has been said on the screen, in order.
--
-- The web surface stopped being a set of pages on 24 August 2026 and became one
-- conversation. This is that conversation, and it is kept indefinitely on the
-- same reasoning that keeps coach_answers: the record is the product now, not a
-- log of it.
--
-- words and shown are text. Neither is a foreign key, and that is the whole
-- design rather than an omission: a turn holding a chore id would re-read the
-- chore at render time and show today's name inside yesterday's sentence, which
-- is rewriting history. The duplication is the price of the record being a
-- record.
create table if not exists turns (
    id          bigserial   primary key,
    person_id   bigint      not null references people (id) on delete cascade,
    -- 'buddy' or 'you'. Text rather than an enum, for the same reason kind is
    -- text on coach_answers: a third speaker should be a phase, not a
    -- migration.
    who         text        not null,
    words       text        not null,
    -- What was drawn beneath the words — the cards, the chips, the picker — as
    -- it was drawn. Null for a turn that was only a sentence.
    shown       jsonb,
    said_at     timestamptz not null default now()
);

-- The only two queries: the newest N for this person, and the N before a given
-- turn. Both walk this index backwards.
create index if not exists turns_person_said
    on turns (person_id, said_at desc, id desc);
```

- [ ] **Step 4: Write the store**

Create `internal/squirrel/turns.go`:

```go
package squirrel

import (
	"context"
	"fmt"
	"time"
)

// Speaker is who said it. Two values, and a third would be a design decision
// rather than a constant.
type Speaker string

const (
	SpeakerBuddy Speaker = "buddy"
	SpeakerYou   Speaker = "you"
)

// Turn is one thing said, as it was said.
//
// Shown is JSON rather than a set of columns because what a turn drew varies by
// what kind of turn it is, and because nothing queries inside it — it is read
// back whole and handed to a template. It is never a pointer at another table;
// see the migration for why.
type Turn struct {
	ID     int64
	Who    Speaker
	Words  string
	Shown  []byte
	SaidAt time.Time
}

// AppendTurn writes one turn and returns it with its id and time filled in.
func (s *Store) AppendTurn(ctx context.Context, personID int64, t Turn) (Turn, error) {
	const q = `
		insert into turns (person_id, who, words, shown)
		values ($1, $2, $3, $4)
		returning id, said_at`

	// nil rather than an empty slice, so a turn with nothing drawn stores SQL
	// null and reads back as nil — an empty JSON document and "there was no
	// document" are different facts about the turn.
	var shown any
	if len(t.Shown) > 0 {
		shown = t.Shown
	}
	if err := s.pool.QueryRow(ctx, q, personID, string(t.Who), t.Words, shown).
		Scan(&t.ID, &t.SaidAt); err != nil {
		return Turn{}, fmt.Errorf("saying it: %w", err)
	}
	return t, nil
}

// RecentTurns is the end of the conversation, oldest first.
//
// The limit is applied to the newest rows and the result is then reversed, so
// the cap keeps the end of the conversation rather than its beginning. more
// means there is something above what was returned.
func (s *Store) RecentTurns(ctx context.Context, personID int64, limit int) ([]Turn, bool, error) {
	const q = `
		select id, who, words, shown, said_at
		  from turns
		 where person_id = $1
		 order by said_at desc, id desc
		 limit $2`
	return s.scanTurns(ctx, limit, q, personID, limit+1)
}

// TurnsBefore is the page above a turn you can already see.
func (s *Store) TurnsBefore(ctx context.Context, personID, beforeID int64, limit int) ([]Turn, bool, error) {
	const q = `
		select id, who, words, shown, said_at
		  from turns
		 where person_id = $1
		   and (said_at, id) < (select said_at, id from turns where id = $2)
		 order by said_at desc, id desc
		 limit $3`
	return s.scanTurns(ctx, limit, q, personID, beforeID, limit+1)
}

// scanTurns reads newest-first rows, reports whether one more than asked for
// came back, and hands the caller the rest oldest-first.
//
// The over-read is how "there is more" is answered without a count query; it
// predates counts being allowed and stays because it costs one row instead of a
// second round trip.
func (s *Store) scanTurns(ctx context.Context, limit int, q string, args ...any) ([]Turn, bool, error) {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, false, fmt.Errorf("reading the conversation: %w", err)
	}
	defer rows.Close()

	var out []Turn
	for rows.Next() {
		var t Turn
		var who string
		if err := rows.Scan(&t.ID, &who, &t.Words, &t.Shown, &t.SaidAt); err != nil {
			return nil, false, fmt.Errorf("reading the conversation: %w", err)
		}
		t.Who = Speaker(who)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("reading the conversation: %w", err)
	}

	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, more, nil
}
```

- [ ] **Step 5: Run the tests, then prove them by mutation**

```bash
TEST_DATABASE_URL=... go test -tags integration ./internal/squirrel/ -run 'Turn' -v
```

Expected: PASS.

Now prove each test bites. Make the mutation, run, record the **exact assertion text**, revert:

| test | mutation | expected failure |
| --- | --- | --- |
| `TestRecentTurnsAreOldestFirst` | delete the reversing loop in `scanTurns` | `Not equal: expected: []string{"first","second","third"} actual: []string{"third","second","first"}` |
| `TestRecentTurnsKeepsTheNewestAndSaysThereIsMore` | change `order by said_at desc, id desc` to `order by said_at, id` in `RecentTurns` | the slice comes back as `turn 0`, `turn 1` |
| `TestShownIsKeptVerbatim` | pass `nil` instead of `shown` in `AppendTurn` | `JSONEq` fails on empty string |
| `TestRecentTurnsAreScopedToThePerson` | drop `where person_id = $1` | `Empty` fails with one turn |
| `TestTurnsBeforePagesBackwards` | change `<` to `>` in the tuple comparison | `turn 4` comes back |

Record all five failure texts in the commit message body. A test whose mutation still passes is not a test; fix it before moving on.

- [ ] **Step 6: Commit**

```bash
git add internal/squirrel/migrations/0027_turns.sql internal/squirrel/turns.go internal/squirrel/turns_test.go
git commit -m "feat: the conversation is kept"
```

---

### Task 2: The four door numbers

**Files:**
- Create: `internal/squirrel/waiting.go`
- Test: `internal/squirrel/waiting_test.go`

**Interfaces:**
- Consumes: `stillMine` (unexported, `internal/squirrel/notes.go:197`), `(*Store).DueChores`, `StartOfDay` (`internal/squirrel/pick.go:343`).
- Produces:
  ```go
  type Waiting struct {
      Pile   int  // notes not yet decided about
      Tasks  int  // tasks not done
      Chores int  // chores due right now
      Agenda int  // fixed points still ahead today
  }
  func (s *Store) Waiting(ctx context.Context, personID int64, now time.Time) (Waiting, error)
  ```

**Why chores are counted differently from the rest:** due-ness is a CTE plus a tolerance gate against the last delivered digest (`internal/squirrel/chores.go:161`). Reimplementing it as a `count(*)` would be a second description of "due" that drifts from the first. So this calls `DueChores` and takes the length. Chores are few; this is the correct trade.

- [ ] **Step 1: Write the failing tests**

Create `internal/squirrel/waiting_test.go`:

```go
//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestWaitingCountsOpenNotes(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	insertItem(t, store, personID, "one thing")
	insertItem(t, store, personID, "another thing")

	w, err := store.Waiting(ctx, personID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 2, w.Pile)
}

// A note that has been decided about is not waiting. Without this the pile
// number would only ever grow.
func TestWaitingIgnoresDecidedNotes(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	id := insertItem(t, store, personID, "one thing")
	require.NoError(t, store.SetItemState(ctx, id, squirrel.StateDone, time.Now()))

	w, err := store.Waiting(ctx, personID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, w.Pile)
}

// A note pointing at an appointment still ahead has somewhere to be. The pile
// screen already excludes it; the number must agree, or the door disagrees with
// the room behind it.
func TestWaitingObeysTheSameRuleAsThePile(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	m, err := store.CreateMoment(ctx, personID, squirrel.Moment{
		Label: "dentist", StartsAt: time.Now().Add(3 * time.Hour),
	})
	require.NoError(t, err)
	id := insertItem(t, store, personID, "the referral letter")
	ok, err := store.AttachNote(ctx, personID, id, m.ID)
	require.NoError(t, err)
	require.True(t, ok)

	w, err := store.Waiting(ctx, personID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, w.Pile)
}

func TestWaitingCountsUndoneTasksOnly(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()

	open := insertItem(t, store, personID, "ring the bank")
	_, err := store.SetItemKind(ctx, personID, open, squirrel.KindTask)
	require.NoError(t, err)

	done := insertItem(t, store, personID, "post the form")
	_, err = store.SetItemKind(ctx, personID, done, squirrel.KindTask)
	require.NoError(t, err)
	require.NoError(t, store.SetItemState(ctx, done, squirrel.StateDone, time.Now()))

	w, err := store.Waiting(ctx, personID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, w.Tasks)
}

// The agenda door is about today. A thing next week is not waiting for you.
func TestWaitingCountsOnlyTodaysFixedPoints(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()
	now := squirrel.StartOfDay(time.Now()).Add(9 * time.Hour)

	_, err := store.CreateMoment(ctx, personID, squirrel.Moment{
		Label: "dentist", StartsAt: now.Add(4 * time.Hour),
	})
	require.NoError(t, err)
	_, err = store.CreateMoment(ctx, personID, squirrel.Moment{
		Label: "next week", StartsAt: now.Add(7 * 24 * time.Hour),
	})
	require.NoError(t, err)

	w, err := store.Waiting(ctx, personID, now)
	require.NoError(t, err)
	require.Equal(t, 1, w.Agenda)
}

// A fixed point that has already started is over, not waiting.
func TestWaitingIgnoresFixedPointsAlreadyPast(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()
	now := squirrel.StartOfDay(time.Now()).Add(14 * time.Hour)

	_, err := store.CreateMoment(ctx, personID, squirrel.Moment{
		Label: "this morning", StartsAt: now.Add(-2 * time.Hour),
	})
	require.NoError(t, err)

	w, err := store.Waiting(ctx, personID, now)
	require.NoError(t, err)
	require.Equal(t, 0, w.Agenda)
}

func TestWaitingCountsDueChores(t *testing.T) {
	store, personID := freshStore(t)
	ctx := context.Background()
	now := time.Now()

	_, err := store.UpsertChore(ctx, personID, "water the plants", 24*time.Hour, time.Hour)
	require.NoError(t, err)

	// Due only once the interval has passed since the chore was created.
	later := now.Add(48 * time.Hour)
	w, err := store.Waiting(ctx, personID, later)
	require.NoError(t, err)
	require.Equal(t, 1, w.Chores)

	// And not before.
	w, err = store.Waiting(ctx, personID, now)
	require.NoError(t, err)
	require.Equal(t, 0, w.Chores)
}
```

Use the real names of the helpers and constants this package's integration tests already use (`insertItem`, `freshStore`, `squirrel.StateDone`, `squirrel.KindTask`). If a name here does not exist, find the equivalent — do not invent a second helper.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
TEST_DATABASE_URL=... go test -tags integration ./internal/squirrel/ -run Waiting -v
```

Expected: compile failure, `undefined: squirrel.Waiting`.

- [ ] **Step 3: Write it**

Create `internal/squirrel/waiting.go`:

```go
package squirrel

import (
	"context"
	"fmt"
	"time"
)

// Waiting is the number on each door.
//
// Principle 2 — nothing accrues that can be destroyed — was retired by the
// owner on 24 August 2026, and this is the first thing built on the retirement.
// What the rule protected against arrives in exactly this shape: a number
// beside a door with an implied target of zero. The decision reverses cleanly
// because nothing here is stored — remove the call and the numbers are gone.
//
// Each field is what is *waiting for you*, not what the door holds, so an empty
// door reads as finished rather than as absent.
type Waiting struct {
	// Pile is notes not yet decided about, under exactly the pile's own rule.
	Pile int
	// Tasks is what you decided and have not done.
	Tasks int
	// Chores is what is due right now — not how many chores you keep.
	Chores int
	// Agenda is fixed points still ahead today. Not this week: a door that
	// counts next Tuesday is a door you cannot empty.
	Agenda int
}

// Waiting reads all four.
//
// Three are count queries; chores are counted by asking DueChores and taking
// the length. Due-ness is a CTE and a tolerance gate against the last delivered
// digest (see DueChores), and a count query would be a second definition of
// "due" that drifts from the first. Chores are few, so this is cheap and it
// cannot disagree.
func (s *Store) Waiting(ctx context.Context, personID int64, now time.Time) (Waiting, error) {
	var w Waiting

	const q = `
		select
		  (select count(*) from items
		    where person_id = $1 and kind = 'note' and state = 'open'
		      and ` + stillMine + `),
		  (select count(*) from items
		    where person_id = $1 and kind = 'task' and state = 'open'),
		  (select count(*) from moments
		    where person_id = $1 and done_at is null
		      and starts_at > $2 and starts_at < $3)`

	endOfDay := StartOfDay(now).Add(24 * time.Hour)
	if err := s.pool.QueryRow(ctx, q, personID, now, endOfDay).
		Scan(&w.Pile, &w.Tasks, &w.Agenda); err != nil {
		return Waiting{}, fmt.Errorf("reading what is waiting: %w", err)
	}

	due, err := s.DueChores(ctx, personID, now)
	if err != nil {
		return Waiting{}, fmt.Errorf("reading what is waiting: %w", err)
	}
	w.Chores = len(due)

	return w, nil
}
```

- [ ] **Step 4: Run the tests**

```bash
TEST_DATABASE_URL=... go test -tags integration ./internal/squirrel/ -run Waiting -v
```

Expected: PASS.

- [ ] **Step 5: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestWaitingObeysTheSameRuleAsThePile` | remove `and ` + `stillMine` from the pile subquery | `Not equal: expected: 0 actual: 1` |
| `TestWaitingIgnoresDecidedNotes` | drop `and state = 'open'` from the pile subquery | expected 0, actual 1 |
| `TestWaitingCountsUndoneTasksOnly` | drop `and state = 'open'` from the task subquery | expected 1, actual 2 |
| `TestWaitingCountsOnlyTodaysFixedPoints` | drop `and starts_at < $3` | expected 1, actual 2 |
| `TestWaitingIgnoresFixedPointsAlreadyPast` | change `starts_at > $2` to `starts_at > $2 - interval '1 day'` | expected 0, actual 1 |
| `TestWaitingCountsDueChores` | replace `len(due)` with a `count(*)` over `chores where active` | the "and not before" half fails, expected 0, actual 1 |

The last one is the important one: it is what stops a future author from "optimising" the extra query into SQL and silently redefining due.

Record the failure texts in the commit body.

- [ ] **Step 6: Commit**

```bash
git add internal/squirrel/waiting.go internal/squirrel/waiting_test.go
git commit -m "feat: what each door is holding for you"
```

---

### Task 3: The thread renders at `/`

**Files:**
- Create: `internal/web/thread.go`, `internal/web/templates/thread.html`, `internal/web/templates/turn.html`
- Modify: `internal/web/web.go:136` (Store interface), `internal/web/render.go:44` and `:95` (pages, view), `internal/web/pile.go:39` (route), `internal/web/static/pile.css`
- Test: `internal/web/thread_test.go`

**Interfaces:**
- Consumes: `squirrel.Turn`, `squirrel.Speaker`, `squirrel.SpeakerBuddy`, `squirrel.SpeakerYou`, `(*Store).RecentTurns`, `(*Store).TurnsBefore`, `(*Store).Waiting`, `squirrel.Waiting` — all from Tasks 1 and 2.
- Produces:
  ```go
  // in internal/web
  type turnView struct {
      ID    int64
      Buddy bool      // who said it, as the template asks the question
      Words string
      Place string    // the <h2> when this turn opens a place, else empty
      Cards []cardView
      Chips []chipView
      Live  bool      // the newest Buddy turn, and only it, carries controls
  }
  type cardView struct {
      Title string
      Meta  string
      Acts  []actView
  }
  type actView struct{ Label, Action, Name, Value, Style string }
  type doorView struct {
      Href, Label, Art string
      Count            int   // 0 renders no number at all
      Here             bool
  }

  func threadHandler(s Store, opts Options) http.HandlerFunc
  func railFor(s Store, ctx context.Context, personID int64, here string) []doorView
  ```
  `view` gains `Turns []turnView`, `Rail []doorView`, `MoreAbove bool`, `Oldest int64`.

- [ ] **Step 1: Write the failing tests**

Create `internal/web/thread_test.go`. Follow the shape of `internal/web/at_test.go` — it builds a `realMux`, registers routes against a fake Store, and drives them with `httptest`.

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The thread renders what was said, in order, with the newest at the bottom.
func TestThreadRendersTurnsInOrder(t *testing.T) {
	s := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Morning."},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "the chores"},
	}}
	body := get(t, s, "/")

	first := strings.Index(body, "Morning.")
	second := strings.Index(body, "the chores")
	require.NotEqual(t, -1, first)
	require.NotEqual(t, -1, second)
	require.Less(t, first, second, "the conversation reads top to bottom")
}

// Only the newest Buddy turn carries controls. This is the live-edge rule, and
// a test that asserts the newest turn HAS buttons would pass with the rule
// deleted — so this asserts the older one does NOT.
func TestOnlyTheNewestBuddyTurnHasControls(t *testing.T) {
	s := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Two are due.",
			Shown: []byte(`{"cards":[{"title":"water the plants","acts":[{"label":"DID IT","action":"/chores/act"}]}]}`)},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "never mind"},
		{ID: 3, Who: squirrel.SpeakerBuddy, Words: "Right.",
			Shown: []byte(`{"cards":[{"title":"descale the kettle","acts":[{"label":"DID IT","action":"/chores/act"}]}]}`)},
	}}
	body := get(t, s, "/")

	require.Contains(t, body, "descale the kettle")
	require.Contains(t, body, "water the plants")
	// Exactly one DID IT on the page: the live edge's.
	require.Equal(t, 1, strings.Count(body, "DID IT"),
		"a card in scrollback keeps its words and loses its buttons")
}

func TestRailShowsFourDoorsWithTheirNumbers(t *testing.T) {
	s := &fakeStore{waiting: squirrel.Waiting{Pile: 3, Tasks: 1, Chores: 2, Agenda: 1}}
	body := get(t, s, "/")

	for _, name := range []string{"the pile", "the tasks", "the chores", "the agenda"} {
		require.Contains(t, body, name)
	}
	require.Contains(t, body, `class="doorcount">3<`)
	require.Contains(t, body, `class="doorcount">2<`)
}

// Zero is no number, not a nought. The empty case is the one nobody renders by
// hand, and a door reading "0" is the scoreboard the retired rule feared.
func TestADoorWithNothingWaitingShowsNoNumber(t *testing.T) {
	s := &fakeStore{waiting: squirrel.Waiting{Pile: 0, Tasks: 0, Chores: 0, Agenda: 0}}
	body := get(t, s, "/")

	require.Contains(t, body, "the pile")
	require.NotContains(t, body, "doorcount")
}

// The thread has no <h1> — home's own exemption — but a turn that opens a place
// carries that place's name as an <h2>, so heading navigation still works.
func TestATurnThatOpensAPlaceCarriesAHeading(t *testing.T) {
	s := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Two are due.",
			Shown: []byte(`{"place":"the chores"}`)},
	}}
	body := get(t, s, "/")

	require.NotContains(t, body, "<h1")
	require.Contains(t, body, "<h2")
	require.Contains(t, body, "the chores")
}

func TestThreadOffersThePageAboveWhenThereIsOne(t *testing.T) {
	s := &fakeStore{
		turns:     []squirrel.Turn{{ID: 7, Who: squirrel.SpeakerYou, Words: "hello"}},
		moreTurns: true,
	}
	body := get(t, s, "/")
	require.Contains(t, body, `/?before=7`)
}

func TestThreadDoesNotOfferAPageAboveWhenThereIsNone(t *testing.T) {
	s := &fakeStore{turns: []squirrel.Turn{{ID: 7, Who: squirrel.SpeakerYou, Words: "hello"}}}
	body := get(t, s, "/")
	require.NotContains(t, body, "before=")
}

func get(t *testing.T, s Store, path string) string {
	t.Helper()
	m := &realMux{mux: http.NewServeMux()}
	Routes(m, s, testOptions())
	rec := httptest.NewRecorder()
	m.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}
```

`fakeStore` and `testOptions` already exist in this package's tests — extend `fakeStore` with `turns []squirrel.Turn`, `moreTurns bool` and `waiting squirrel.Waiting` fields and the three new methods, rather than writing a second fake. `Routes` is whatever `internal/web/pile.go:39`'s registration function is actually called; use its real name.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/web/ -run 'Thread|Rail|Door|Turn' -v
```

Expected: compile failure on the new `fakeStore` methods, then — once the fake compiles — assertion failures such as `Error: "..." does not contain "the pile"`.

- [ ] **Step 3: Widen the Store interface**

In `internal/web/web.go`, inside `type Store interface`, after the `SaveSteps`/`ClearSteps` block:

```go
	// The conversation. The screen is one now — see
	// docs/superpowers/specs/2026-08-24-the-thread-design.md — and these are
	// the only three things it does with it: add to it, read the end of it,
	// and walk back up it.
	AppendTurn(ctx context.Context, personID int64, t squirrel.Turn) (squirrel.Turn, error)
	RecentTurns(ctx context.Context, personID int64, limit int) ([]squirrel.Turn, bool, error)
	TurnsBefore(ctx context.Context, personID, beforeID int64, limit int) ([]squirrel.Turn, bool, error)
	// The four numbers on the doors. Computed at read time and stored
	// nowhere, which is what makes the decision that allowed them reversible.
	Waiting(ctx context.Context, personID int64, now time.Time) (squirrel.Waiting, error)
```

- [ ] **Step 4: Write the handler and views**

Create `internal/web/thread.go`:

```go
package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The thread: the whole of the screen.
//
// This replaced home on 24 August 2026. Home's argument was that a front door
// showing what is waiting greets you with what is waiting; the owner retired
// that along with Principle 2, and the doors carry numbers now. What survives
// unchanged is that the doors are equals — one grid, four cells, the same stock
// — and that the slot is first in reach even though it is last in the markup.
//
// Only the newest Buddy turn carries controls. A card from this morning keeps
// its words and loses its buttons, because pressing DID IT on a card from a
// conversation three days old acts on a state nobody is looking at. See The
// live edge in the spec.

// threadLimit is how much of the conversation one render holds. A bound rather
// than a page: everything above it is still there and one press away.
const threadLimit = 40

// shown is what a turn drew, as it was drawn. Decoded from the turn's own JSON
// and never re-read from another table — a turn holding a chore id would show
// today's chore inside yesterday's sentence.
type shown struct {
	Place string     `json:"place,omitempty"`
	Cards []cardView `json:"cards,omitempty"`
	Chips []chipView `json:"chips,omitempty"`
}

type turnView struct {
	ID    int64
	Buddy bool
	Words string
	// Place is the <h2> when this turn opens one, and empty otherwise. The
	// thread has no <h1> — home's exemption, and nobody arrives at the place
	// they started wondering where they are — so these are what heading
	// navigation walks.
	Place string
	Cards []cardView
	Chips []chipView
	// Live is the newest Buddy turn and nothing else.
	Live bool
}

type cardView struct {
	Title string    `json:"title"`
	Meta  string    `json:"meta,omitempty"`
	Acts  []actView `json:"acts,omitempty"`
}

type actView struct {
	Label  string `json:"label"`
	Action string `json:"action"`
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Style  string `json:"style,omitempty"`
}

type doorView struct {
	Href  string
	Label string
	Art   string
	// Count is what is waiting behind the door. Zero renders no number at
	// all — a door reading "0" is a scoreboard, and that is the thing the
	// retired rule was actually protecting against.
	Count int
	Here  bool
}

func threadHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		ctx := r.Context()

		var (
			turns []squirrel.Turn
			more  bool
			err   error
		)
		// `?before=` walks up the conversation. It is in the address bar rather
		// than in a cursor because a page of the past is a place you can send
		// yourself back to.
		if before, perr := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64); perr == nil && before > 0 {
			turns, more, err = s.TurnsBefore(ctx, personID, before, threadLimit)
		} else {
			turns, more, err = s.RecentTurns(ctx, personID, threadLimit)
		}
		if err != nil {
			fail(w, err)
			return
		}

		v := view{
			Home:      true,
			Here:      "thread",
			Scrolling: true,
			Turns:     turnViews(turns),
			Rail:      railFor(ctx, s, personID, ""),
			MoreAbove: more,
		}
		if len(turns) > 0 {
			v.Oldest = turns[0].ID
		}
		renderWith(w, r, s, opts, "thread", v)
	}
}

// turnViews decodes each turn's own record of what it drew, and marks the live
// edge.
//
// The scan for the live edge runs backwards and stops at the first Buddy turn,
// so a run of your own turns at the bottom does not leave the conversation
// without anything to press.
func turnViews(turns []squirrel.Turn) []turnView {
	out := make([]turnView, 0, len(turns))
	for _, t := range turns {
		v := turnView{ID: t.ID, Buddy: t.Who == squirrel.SpeakerBuddy, Words: t.Words}
		if len(t.Shown) > 0 {
			var sh shown
			if err := json.Unmarshal(t.Shown, &sh); err != nil {
				// A turn whose record cannot be read still said something, and
				// the words are the part that matters. Losing the cards is
				// better than losing the turn.
				slog.Error("reading what a turn drew", "turn", t.ID, "error", err)
			} else {
				v.Place, v.Cards, v.Chips = sh.Place, sh.Cards, sh.Chips
			}
		}
		out = append(out, v)
	}
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Buddy {
			out[i].Live = true
			break
		}
	}
	return out
}

// railFor is the four doors, with what is waiting behind each.
//
// A failed read is four doors and no numbers rather than an error page: the
// doors are how you get anywhere, and a database that cannot count is not a
// reason to take the navigation away.
func railFor(ctx context.Context, s Store, personID int64, here string) []doorView {
	rail := []doorView{
		{Href: "/pile", Label: "the pile", Art: "door-pile.png"},
		{Href: "/tasks", Label: "the tasks", Art: "door-tasks.png"},
		{Href: "/chores", Label: "the chores", Art: "door-chores.png"},
		{Href: "/at", Label: "the agenda", Art: "door-at.png"},
	}
	for i := range rail {
		rail[i].Here = rail[i].Label == here
	}
	w, err := s.Waiting(ctx, personID, now())
	if err != nil {
		slog.Error("counting what is waiting", "error", err)
		return rail
	}
	rail[0].Count, rail[1].Count, rail[2].Count, rail[3].Count = w.Pile, w.Tasks, w.Chores, w.Agenda
	return rail
}
```

- [ ] **Step 5: Add the view fields and register the page**

In `internal/web/render.go`, add to `type view struct` (after `Clash bool`):

```go
	// Turns is the conversation, oldest first. The screen is one page now;
	// see internal/web/thread.go.
	Turns []turnView
	// Rail is the four doors, pinned under the lid at every width, with what
	// is waiting behind each.
	Rail []doorView
	// MoreAbove and Oldest are the page above this one. Oldest is the id the
	// "earlier" control walks back from.
	MoreAbove bool
	Oldest    int64
```

Add to `pages`:

```go
	"thread": page("templates/layout.html", "templates/turn.html", "templates/thread.html"),
```

- [ ] **Step 6: Write the templates**

Create `internal/web/templates/turn.html`:

```html
{{/* One turn, and the same markup whether it arrives with the page or as a
     fragment after a press. That sameness is the single rendering path, and
     it is the only thing keeping the server and the browser from growing two
     descriptions of a card. */}}
{{define "turn"}}
<div class="turn {{if .Buddy}}frombuddy{{else}}fromyou{{end}}" id="turn-{{.ID}}">
  {{/* The place's name, when this turn opens one. The thread has no <h1>, so
       these are what heading navigation walks — see The One Title Rule. */}}
  {{if .Place}}<h2 class="turnplace">{{.Place}}</h2>{{end}}
  <p class="bub">{{.Words}}</p>

  {{range .Cards}}
  <div class="tcard">
    <p class="name">{{.Title}}</p>
    {{if .Meta}}<p class="meta">{{.Meta}}</p>{{end}}
    {{/* Controls only on the live edge. A card in scrollback keeps its words
         and loses its buttons: pressing DID IT on a card from three days ago
         acts on a state nobody is looking at. */}}
    {{if and $.Live .Acts}}
    <div class="acts">
      {{range .Acts}}
      <form method="post" action="{{.Action}}">
        {{if .Name}}<input type="hidden" name="{{.Name}}" value="{{.Value}}">{{end}}
        <button class="abtn {{.Style}}" type="submit">{{.Label}}</button>
      </form>
      {{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if and .Live .Chips}}
  <div class="chips">
    {{range .Chips}}<a class="chip" href="{{.Href}}">{{.Label}}</a>{{end}}
  </div>
  {{end}}
</div>
{{end}}
```

`chipView` already exists in `internal/web/render.go` — use its real fields rather than inventing new ones; if it has no `Href`, add one there rather than declaring a second chip type.

Create `internal/web/templates/thread.html`:

```html
{{define "content"}}
{{/* The doors, pinned under the lid and never scrolled away. Four equals: one
     grid, four cells, the same stock, and they render identically in every
     state. What changed on 24 August 2026 is that they carry numbers — see
     Waiting in internal/squirrel for what that decision cost and how it
     reverses. */}}
<nav class="railwrap" aria-label="The four things the box holds">
  <div class="rail">
    {{range .Rail}}
    <a class="rdoor{{if .Here}} here{{end}}" href="{{.Href}}">
      <img alt="" src="/static/{{.Art}}?v={{$.V}}">
      <span class="rname">{{.Label}}</span>
      {{/* Zero is no number, not a nought. */}}
      {{if .Count}}<span class="doorcount">{{.Count}}</span>{{end}}
    </a>
    {{end}}
  </div>
</nav>

<div class="thread" id="thread">
  {{/* The way up the conversation. It says that there is more and never how
       much more — not because a count is banned any longer, but because the
       number above you is not a thing you can act on. */}}
  {{if .MoreAbove}}<p class="earlier"><a href="/?before={{.Oldest}}">earlier</a></p>{{end}}
  {{range .Turns}}{{template "turn" .}}{{end}}
</div>

{{/* Anything this page changes without a navigation is said out loud here too,
     or the screen is silent for exactly the person who cannot see it change. */}}
<p class="visually-hidden" id="threadsay" role="status" aria-live="polite"></p>

<form class="dock" method="post" action="/say">
  <div class="slot">
    <input name="words" placeholder="{{.SaySlot}}" autocomplete="off" autofocus>
    <button class="post" type="submit">Tell it</button>
  </div>
</form>
{{end}}
```

- [ ] **Step 7: Route it**

In `internal/web/pile.go`, replace line 39:

```go
	m.Get("/{$}", guard(opts, threadHandler(s, opts)))
```

- [ ] **Step 8: Style it**

Add to `internal/web/static/pile.css`, using the existing custom properties (`--outline`, `--paper`, `--tail`, `--ink`) rather than the literal hexes from the mockup — the mockup at `$SCRATCH/spa/spa2.html` is the reference for geometry, not for colour:

```css
/* THE RAIL — the doors, pinned, four across at every width. */
.railwrap { position: sticky; top: 0; z-index: 5; background: var(--lid);
  border-bottom: 3px solid var(--outline); box-shadow: 0 12px 24px -16px rgba(0,0,0,.85); }
.rail { width: min(720px, 100%); margin: 0 auto; display: grid;
  grid-template-columns: repeat(4, 1fr); gap: 9px; padding: 10px 22px; }
.rdoor { display: flex; align-items: center; justify-content: center; gap: 9px;
  min-height: 52px; padding: 8px 12px; background: var(--paper); color: var(--ink);
  border: 3px solid var(--outline); border-radius: 999px;
  box-shadow: 0 4px 0 0 var(--outline); text-decoration: none; }
.rdoor img { height: 28px; width: auto; display: block; flex: none; }
.rname { font-size: 14.5px; font-variation-settings: "MONO" 0, "CASL" 0, "wght" 800; line-height: 1.05; }
.rdoor.here { background: var(--orange); }
.doorcount { min-width: 22px; height: 22px; padding: 0 5px; display: grid; place-items: center;
  background: var(--outline); color: var(--paper); border-radius: 999px;
  font-size: 12px; font-variation-settings: "MONO" 0, "CASL" 0, "wght" 800; }

@media (max-width: 620px) {
  .rail { gap: 7px; padding: 9px 14px; }
  .rdoor { flex-direction: column; gap: 3px; min-height: 60px; padding: 7px 4px;
    border-radius: 16px; position: relative; }
  .rdoor img { height: 24px; }
  .rname { font-size: 11px; }
  .doorcount { position: absolute; top: -6px; right: -2px; }
}

/* THE THREAD */
.thread { width: min(720px, 100%); margin: 0 auto;
  padding: 22px 22px 120px; display: flex; flex-direction: column; gap: 16px; }
.turn { display: flex; flex-direction: column; gap: 10px; max-width: 92%; }
.frombuddy { align-self: flex-start; }
.fromyou { align-self: flex-end; align-items: flex-end; max-width: 80%; }
.bub { margin: 0; padding: 11px 15px; border: 3px solid var(--outline); border-radius: 14px;
  font-size: 16px; line-height: 1.4; font-variation-settings: "MONO" 0, "CASL" 1, "wght" 500; }
.frombuddy .bub { background: var(--paper); color: var(--ink); box-shadow: 0 4px 0 0 var(--outline); }
.fromyou .bub { background: var(--tail); color: var(--ink); }
.turnplace { margin: 0; font-family: Inter, sans-serif; font-weight: 900; font-size: 19px;
  letter-spacing: -.02em; color: var(--paper); }
.earlier { text-align: center; margin: 0; }

/* THE DOCK — the slot, on every view of the app, because there is one view. */
.dock { position: fixed; left: 0; right: 0; bottom: 0;
  padding: 14px 22px calc(16px + env(safe-area-inset-bottom));
  background: linear-gradient(to top, var(--lid) 60%, transparent); }
```

Reuse the existing `.tcard`/`.acts`/`.abtn`/`.chip`/`.slot`/`.post` rules if this stylesheet already has equivalents under other names — check before adding. `.visually-hidden` already exists.

- [ ] **Step 9: Run the tests**

```bash
go test ./internal/web/ -run 'Thread|Rail|Door|Turn' -v
go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 10: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestOnlyTheNewestBuddyTurnHasControls` | remove `$.Live` from the `{{if and $.Live .Acts}}` guard in `turn.html` | `Not equal: expected: 1 actual: 2` |
| `TestADoorWithNothingWaitingShowsNoNumber` | change `{{if .Count}}` to always render the span | `"..." should not contain "doorcount"` |
| `TestRailShowsFourDoorsWithTheirNumbers` | drop the `Waiting` call from `railFor` and return the bare rail | the `doorcount>3<` assertion fails |
| `TestThreadRendersTurnsInOrder` | reverse `v.Turns` before rendering | `"first" is not less than "second"` |
| `TestATurnThatOpensAPlaceCarriesAHeading` | drop the `{{if .Place}}<h2>` line | does not contain `<h2` |
| `TestThreadOffersThePageAboveWhenThereIsOne` | hardcode `MoreAbove: false` | does not contain `/?before=7` |

Record the failure texts in the commit body.

- [ ] **Step 11: Commit**

```bash
git add internal/web/thread.go internal/web/thread_test.go \
        internal/web/templates/thread.html internal/web/templates/turn.html \
        internal/web/web.go internal/web/render.go internal/web/pile.go \
        internal/web/static/pile.css
git commit -m "feat: the front door is a conversation"
```

---

### Task 4: The dock writes a pair of turns

**Files:**
- Create: nothing
- Modify: `internal/web/thread.go` (add `threadSayHandler`), `internal/web/pile.go` (add `POST /say`)
- Test: `internal/web/thread_test.go`

**Interfaces:**
- Consumes: `(*Store).AppendTurn`, `captureHandler`'s existing write path (`InsertItem`), `squirrel.ParseIntent` (`internal/squirrel/intent.go`), `captureLimit` (`internal/web/capture.go`).
- Produces: `func threadSayHandler(s Store, opts Options) http.HandlerFunc`, routed at `POST /say`.

**What this does and does not do:** it keeps the words as a note exactly as `/capture` does today, and it writes two turns — yours, then Buddy's acknowledgement. It does **not** yet route a sentence to a chore, a fixed point, or the model; that is phase 2. A sentence `ParseIntent` recognises as anything other than a capture still becomes a capture here, which is the safe direction to be wrong in and is what `/capture` already does.

- [ ] **Step 1: Write the failing tests**

Add to `internal/web/thread_test.go`:

```go
// Two turns for one press: what you said, and what Buddy said back. A test that
// only checks the note was kept would pass with the conversation missing.
func TestSayingSomethingWritesBothTurns(t *testing.T) {
	s := &fakeStore{}
	post(t, s, "/say", "words=milk")

	require.Len(t, s.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, s.appended[0].Who)
	require.Equal(t, "milk", s.appended[0].Words)
	require.Equal(t, squirrel.SpeakerBuddy, s.appended[1].Who)
	require.NotEmpty(t, s.appended[1].Words)
}

// The words still land in the pile. The thread is a record of the conversation,
// not a replacement for capture — losing a thought is the one failure this
// product exists to prevent.
func TestSayingSomethingStillKeepsTheNote(t *testing.T) {
	s := &fakeStore{}
	post(t, s, "/say", "words=milk")

	require.Len(t, s.inserted, 1)
	require.Equal(t, "milk", s.inserted[0].RawText)
}

// An empty slot is not a turn. Pressing the button by accident must not put a
// blank bubble in the record that can never be removed.
func TestSayingNothingWritesNothing(t *testing.T) {
	s := &fakeStore{}
	post(t, s, "/say", "words=%20%20")

	require.Empty(t, s.appended)
	require.Empty(t, s.inserted)
}

// A note that could not be kept must not be acknowledged as kept. Buddy saying
// "kept" over a failed write is the two views disagreeing about the pile.
func TestSayingSomethingThatCannotBeKeptSaysSo(t *testing.T) {
	s := &fakeStore{insertErr: errors.New("no")}
	post(t, s, "/say", "words=milk")

	require.Len(t, s.appended, 2)
	require.Contains(t, s.appended[1].Words, "cannot reach")
}

func post(t *testing.T, s Store, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	m := &realMux{mux: http.NewServeMux()}
	Routes(m, s, testOptions())
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.test")
	req.Host = "example.test"
	rec := httptest.NewRecorder()
	m.mux.ServeHTTP(rec, req)
	return rec
}
```

Extend `fakeStore` with `appended []squirrel.Turn`, `inserted []squirrel.Item` and `insertErr error`. Check how the existing capture tests satisfy `sameOrigin` (`internal/web/csrf.go:28`) and copy that, rather than guessing at the headers.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run Saying -v
```

Expected: `404` from the recorder, so `require.Len(t, s.appended, 2)` fails with `expected 2, actual 0`.

- [ ] **Step 3: Write the handler**

Add to `internal/web/thread.go`:

```go
// threadSayHandler is the dock: one line in, two turns out.
//
// The note is kept first and the turns are written after, in that order and not
// the other way round: if the process dies between them the thought is in the
// pile and the conversation is missing a line, which is recoverable. The
// reverse loses the thought, and losing thoughts is the one failure this
// product exists to prevent.
func threadSayHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		ctx := r.Context()
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		text := strings.TrimSpace(r.FormValue("words"))
		if text == "" {
			// An empty slot is not a turn. A blank bubble in a record that is
			// never rewritten is a blank bubble forever.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}

		_, insertErr := s.InsertItem(ctx, squirrel.Item{
			Transport: "screen", PersonID: &personID, RawText: text,
			Payload: []byte(squirrel.ScreenCapture), ReceivedAt: now(),
		})

		if _, err := s.AppendTurn(ctx, personID, squirrel.Turn{
			Who: squirrel.SpeakerYou, Words: text,
		}); err != nil {
			slog.Error("keeping what you said", "error", err)
		}

		// What Buddy says back, and it must never claim more than happened.
		reply := "Kept."
		if insertErr != nil {
			slog.Error("keeping a note from the dock", "error", insertErr)
			reply = "Not kept — Squirrel cannot reach its memory. Your words are still above; try again in a moment."
		}
		if _, err := s.AppendTurn(ctx, personID, squirrel.Turn{
			Who: squirrel.SpeakerBuddy, Words: reply,
		}); err != nil {
			slog.Error("saying it back", "error", err)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
```

Add `"strings"` to the imports.

- [ ] **Step 4: Route it**

In `internal/web/pile.go`, beside the other posts:

```go
	m.Post("/say", guard(opts, sameOrigin(threadSayHandler(s, opts))))
```

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/web/ -run Saying -v
```

Expected: PASS.

- [ ] **Step 6: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestSayingSomethingWritesBothTurns` | drop the second `AppendTurn` | `expected 2, actual 1` |
| `TestSayingSomethingStillKeepsTheNote` | drop the `InsertItem` call | `expected 1, actual 0` |
| `TestSayingNothingWritesNothing` | remove the `text == ""` guard | `Empty` fails with two turns |
| `TestSayingSomethingThatCannotBeKeptSaysSo` | ignore `insertErr` and always reply `"Kept."` | does not contain `cannot reach` |

- [ ] **Step 7: Commit**

```bash
git add internal/web/thread.go internal/web/thread_test.go internal/web/pile.go
git commit -m "feat: the dock says it back"
```

---

### Task 5: The check-in is a turn

**Files:**
- Modify: `internal/web/thread.go`, `internal/web/pile.go`
- Test: `internal/web/thread_test.go`

**Interfaces:**
- Consumes: `(Store).LatestCheckin`, `(Store).RecordCheckin`, `squirrel.Moods`, `squirrel.Words`, `(squirrel.Checkin).Fresh`, `(*Store).AppendTurn`.
- Produces: `func checkinTurn(ctx context.Context, s Store, personID int64) (squirrel.Turn, bool)` and `func threadMoodHandler(s Store, opts Options) http.HandlerFunc` at `POST /mood` (replacing the existing `moodHandler` registration).

**The change in behaviour, stated because it is deliberate:** today the answer *replaces* the question and the morning is gone. Here the question is a turn and the answer is a turn, so both stay in the record. The owner was told this and chose it.

- [ ] **Step 1: Write the failing tests**

```go
// Buddy asks on a day it has not asked yet, and the question is a turn, so the
// morning is still in the record this evening.
func TestBuddyAsksHowYouAreWhenTheReadingIsStale(t *testing.T) {
	s := &fakeStore{}  // no check-in at all
	get(t, s, "/")

	require.Len(t, s.appended, 1)
	require.Equal(t, squirrel.SpeakerBuddy, s.appended[0].Who)
	require.Contains(t, string(s.appended[0].Shown), "chips")
}

// And does not ask twice. A question re-asked on every render would fill the
// record with the same turn.
func TestBuddyDoesNotAskTwiceInOneDay(t *testing.T) {
	s := &fakeStore{
		checkin:      squirrel.Checkin{Mood: squirrel.Moods[0], At: time.Now()},
		checkinFound: true,
	}
	get(t, s, "/")
	require.Empty(t, s.appended)
}

// Answering writes your turn and Buddy's, and records the reading.
func TestAnsweringTheCheckinWritesTurnsAndRecords(t *testing.T) {
	s := &fakeStore{}
	post(t, s, "/mood", "mood="+string(squirrel.Moods[0]))

	require.True(t, s.recordedCheckin)
	require.Len(t, s.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, s.appended[0].Who)
	require.Equal(t, squirrel.SpeakerBuddy, s.appended[1].Who)
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run 'Checkin|HowYouAre|AsksTwice' -v
```

Expected: `expected 1, actual 0` on the first.

- [ ] **Step 3: Write it**

Add to `internal/web/thread.go`:

```go
// checkinTurn is Buddy's question, on a day it has not been asked.
//
// It is a turn rather than a region, which is the change: the answer used to
// replace the question and the morning was gone. Both stay now, which is what
// a record that is never rewritten buys.
func checkinTurn(ctx context.Context, s Store, personID int64) (squirrel.Turn, bool) {
	c, found, err := s.LatestCheckin(ctx, personID)
	if err != nil {
		slog.Error("reading how you are", "error", err)
		return squirrel.Turn{}, false
	}
	if found && c.Fresh(now()) {
		return squirrel.Turn{}, false
	}
	chips := make([]chipView, 0, len(squirrel.Moods))
	for _, m := range squirrel.Moods {
		chips = append(chips, chipView{Label: squirrel.Words[m], Href: "/mood?mood=" + string(m)})
	}
	body, err := json.Marshal(shown{Chips: chips})
	if err != nil {
		slog.Error("drawing the faces", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "How are you doing?", Shown: body}, true
}
```

In `threadHandler`, after the turns are read and before the view is built:

```go
		// The one thing Buddy opens with, and only on a day it has not asked.
		// Written rather than rendered, because a question that is not in the
		// record is a question the record cannot show you answering.
		if r.URL.Query().Get("before") == "" {
			if t, ask := checkinTurn(ctx, s, personID); ask {
				if saved, err := s.AppendTurn(ctx, personID, t); err == nil {
					turns = append(turns, saved)
				} else {
					slog.Error("asking how you are", "error", err)
				}
			}
		}
```

Add `threadMoodHandler`, which does what `moodHandler` does and writes the two turns:

```go
// threadMoodHandler is the answer to Buddy's question.
//
// It records the reading exactly as the old home screen did, and adds the two
// turns. The reading is written first: an answer that is in the conversation
// but not in the readings would make the moods page disagree with the thread.
func threadMoodHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		ctx := r.Context()
		mood, ok := squirrel.ParseMood(r.FormValue("mood"))
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := s.RecordCheckin(ctx, personID, mood, "screen", now()); err != nil {
			slog.Error("recording how you are", "error", err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if _, err := s.AppendTurn(ctx, personID, squirrel.Turn{
			Who: squirrel.SpeakerYou, Words: squirrel.Words[mood],
		}); err != nil {
			slog.Error("keeping your answer", "error", err)
		}
		if _, err := s.AppendTurn(ctx, personID, squirrel.Turn{
			Who: squirrel.SpeakerBuddy, Words: squirrel.Ack(mood, now()),
		}); err != nil {
			slog.Error("saying it back", "error", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
```

`squirrel.ParseMood` and the acknowledgement wording already exist — the existing `moodHandler` in `internal/web/pile.go` or `internal/web/mood.go` shows both. Use those real names; if there is no acknowledgement function, use the wording the old home template used after an answer rather than writing new copy.

Route it: replace the `/mood` registration with `threadMoodHandler`. It must accept GET as well as POST, because the chips render as links — either register both verbs or make the chips a form. Prefer the form, and give `chipView` an `Action`/`Name`/`Value` if it lacks them.

- [ ] **Step 4: Run the tests**

```bash
go test ./internal/web/ -run 'Checkin|HowYouAre|AsksTwice' -v
```

Expected: PASS.

- [ ] **Step 5: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestBuddyAsksHowYouAreWhenTheReadingIsStale` | delete the `checkinTurn` block from `threadHandler` | `expected 1, actual 0` |
| `TestBuddyDoesNotAskTwiceInOneDay` | remove the `c.Fresh(now())` guard | `Empty` fails with one turn |
| `TestAnsweringTheCheckinWritesTurnsAndRecords` | drop `RecordCheckin` | `True` fails on `recordedCheckin` |

- [ ] **Step 6: Commit**

```bash
git add internal/web/thread.go internal/web/thread_test.go internal/web/pile.go
git commit -m "feat: buddy asks, and the morning stays"
```

---

### Task 6: The offer is a turn

**Files:**
- Modify: `internal/web/thread.go`, `internal/web/pile.go`
- Test: `internal/web/thread_test.go`

**Interfaces:**
- Consumes: `offerFor` (`internal/web/now.go` — the existing helper `homeHandler` called), `offerView`, `nowActHandler` (`POST /now/act`), `(*Store).AppendTurn`.
- Produces: `func offerTurn(ctx context.Context, s Store, opts Options, r *http.Request) (squirrel.Turn, bool)`.

- [ ] **Step 1: Write the failing tests**

```go
// The one thing Squirrel picked, offered as a turn with something to press.
func TestTheOfferArrivesAsATurn(t *testing.T) {
	s := &fakeStore{
		checkin:      squirrel.Checkin{Mood: squirrel.Moods[0], At: time.Now()},
		checkinFound: true,
		offer:        squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Label: "water the plants"},
		offerFound:   true,
	}
	body := get(t, s, "/")
	require.Contains(t, body, "water the plants")
	require.Contains(t, body, "/now/act")
}

// Nothing to hand you is a normal state and renders nothing at all — not an
// empty region and not a reassuring sentence.
func TestNoOfferIsNoTurn(t *testing.T) {
	s := &fakeStore{
		checkin:      squirrel.Checkin{Mood: squirrel.Moods[0], At: time.Now()},
		checkinFound: true,
	}
	get(t, s, "/")
	require.Empty(t, s.appended)
}

// Buddy does not hand you a job in the same breath as asking how you are.
// This was home's rule and it survives the move.
func TestNoOfferUntilTheQuestionIsAnswered(t *testing.T) {
	s := &fakeStore{
		offer:      squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Label: "water the plants"},
		offerFound: true,
	}
	body := get(t, s, "/")
	require.NotContains(t, body, "water the plants")
}

// Turning it down stays in the record. Stopping partway is a normal ending, and
// a record that keeps it is what that looks like when it is structural.
func TestTurningTheOfferDownStaysInTheThread(t *testing.T) {
	s := &fakeStore{}
	post(t, s, "/now/act", "kind=chore&id=4&act=not")

	require.Len(t, s.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, s.appended[0].Who)
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run Offer -v
```

- [ ] **Step 3: Write it**

Add to `internal/web/thread.go`:

```go
// offerTurn is the one thing Squirrel picked, or nothing.
//
// Nothing renders nothing: being handed nothing is a normal state, and a
// reassuring sentence in its place would be the product deciding you ought to
// be busy. That was home's rule and it is unchanged by the move.
func offerTurn(ctx context.Context, s Store, opts Options, r *http.Request) (squirrel.Turn, bool) {
	o := offerFor(s, opts, r, false, true)
	if o == nil {
		return squirrel.Turn{}, false
	}
	body, err := json.Marshal(shown{Cards: []cardView{{
		Title: o.Label,
		Meta:  o.Because,
		Acts: []actView{
			{Label: "DID IT", Action: "/now/act", Name: "act", Value: "did", Style: "did"},
			{Label: "NOT NOW", Action: "/now/act", Name: "act", Value: "not", Style: "not"},
		},
	}}})
	if err != nil {
		slog.Error("drawing the offer", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: nowWords(), Shown: body}, true
}
```

`offerView`'s real fields are in `internal/web/render.go` — use those. The offer's action form needs `kind` and `id` as well as `act`; `actView` has one `Name`/`Value` pair, so either widen it to `Fields map[string]string` or add the two hidden inputs from the template when the action is `/now/act`. **Widen `actView`** — a card that can only carry one hidden field will be wrong again in phase 2:

```go
type actView struct {
	Label  string            `json:"label"`
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
	Style  string            `json:"style,omitempty"`
}
```

and in `turn.html`:

```html
        {{range $k, $v := .Fields}}<input type="hidden" name="{{$k}}" value="{{$v}}">{{end}}
```

Update Task 3's `Name`/`Value` usages accordingly — this is the one type that changes shape after being introduced, and it changes here rather than later on purpose.

In `threadHandler`, after the check-in block:

```go
		// Only once the question has been answered. Asking how you are and
		// then handing you a job in the same breath is the interruption this
		// product exists to reduce.
		if r.URL.Query().Get("before") == "" && !asked {
			if t, has := offerTurn(ctx, s, opts, r); has {
				if saved, err := s.AppendTurn(ctx, personID, t); err == nil {
					turns = append(turns, saved)
				} else {
					slog.Error("offering it", "error", err)
				}
			}
		}
```

where `asked` is the bool `checkinTurn` returned.

In `nowActHandler`, after the existing write succeeds, append the two turns — your press and Buddy's answer — and redirect to `/` rather than to `/`-with-query.

- [ ] **Step 4: Run the tests**

```bash
go test ./internal/web/ -run Offer -v
```

- [ ] **Step 5: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestNoOfferUntilTheQuestionIsAnswered` | drop `&& !asked` | contains `water the plants` |
| `TestNoOfferIsNoTurn` | return a turn when `o == nil` | `Empty` fails with one turn |
| `TestTurningTheOfferDownStaysInTheThread` | remove the `AppendTurn` calls from `nowActHandler` | `expected 2, actual 0` |
| `TestTheOfferArrivesAsATurn` | drop the `Acts` from the card | does not contain `/now/act` |

- [ ] **Step 6: Commit**

```bash
git add internal/web/thread.go internal/web/thread_test.go internal/web/pile.go
git commit -m "feat: the one thing, as something buddy said"
```

---

### Task 7: The swap — turns arrive without a paint

**Files:**
- Create: `internal/web/static/thread.js`
- Modify: `internal/web/thread.go` (fragment response), `internal/web/templates/layout.html` (load the module)
- Test: `internal/web/thread_test.go`, `internal/web/browser_test.go`

**Interfaces:**
- Consumes: `render`, `pages["thread"]`, the `turn` template from Task 3.
- Produces: `func renderTurns(w http.ResponseWriter, turns []turnView)` — writes only `{{template "turn"}}` for each, no layout. Every POST handler in Tasks 4–6 gains: if the request carries `X-Thread: fragment`, respond with the new turns' HTML and status 200 instead of redirecting.

**Why a header rather than a path:** one URL per action, and the same handler doing the same write. A second route would be a second place the write can drift.

- [ ] **Step 1: Write the failing tests**

```go
// The fragment and the page render the same card. This is the only thing
// holding the single rendering path in place, and it fails the moment somebody
// adds a client-side template.
func TestAFragmentAndAPageRenderTheSameCard(t *testing.T) {
	turn := squirrel.Turn{ID: 9, Who: squirrel.SpeakerBuddy, Words: "Kept."}

	page := get(t, &fakeStore{turns: []squirrel.Turn{turn}}, "/")

	s := &fakeStore{}
	rec := postFragment(t, s, "/say", "words=milk")
	fragment := rec.Body.String()

	require.Contains(t, page, `<p class="bub">Kept.</p>`)
	require.Contains(t, fragment, `<p class="bub">Kept.</p>`)
	require.NotContains(t, fragment, "<html", "a fragment is turns and nothing else")
}

func TestAFragmentPostAnswersWithTheNewTurnsAndNoRedirect(t *testing.T) {
	s := &fakeStore{}
	rec := postFragment(t, s, "/say", "words=milk")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "milk")
}

// Without the header it still redirects, because that is what a form does and
// the handler must not have two behaviours by accident.
func TestAnOrdinaryPostStillRedirects(t *testing.T) {
	s := &fakeStore{}
	rec := post(t, s, "/say", "words=milk")
	require.Equal(t, http.StatusSeeOther, rec.Code)
}

func postFragment(t *testing.T, s Store, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	m := &realMux{mux: http.NewServeMux()}
	Routes(m, s, testOptions())
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("X-Thread", "fragment")
	req.Host = "example.test"
	rec := httptest.NewRecorder()
	m.mux.ServeHTTP(rec, req)
	return rec
}
```

And a browser test in `internal/web/browser_test.go` (build tag `browser`), following the shape of the tests already there:

```go
// Pressing the dock adds a turn without navigating. The assertion is on the
// page not having reloaded — a test that only checked the turn appeared would
// pass with the JavaScript deleted, because the redirect would put it there.
func TestTheThreadGrowsWithoutANavigation(t *testing.T) {
	b := openBrowser(t)
	b.goTo("/")
	b.eval(`window.__stillHere = true`)

	b.type_(".dock input", "milk")
	b.click(".dock .post")
	b.waitFor(`document.querySelectorAll('.turn').length >= 2`)

	require.True(t, b.evalBool(`window.__stillHere === true`),
		"the page navigated; the swap did not happen")
}
```

Use this file's real helper names — `openBrowser`, `goTo`, `click` and the rest may be spelled differently; read the file first.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run Fragment -v
go test -tags browser ./internal/web/ -run ThreadGrows -v
```

- [ ] **Step 3: Write the fragment response**

Add to `internal/web/thread.go`:

```go
// wantsFragment is a press made by the script rather than by the browser's own
// form machinery.
//
// A header rather than a second route: one URL per action, one handler, one
// write. A `/say/fragment` twin would be a second place the write can drift
// from the first.
func wantsFragment(r *http.Request) bool { return r.Header.Get("X-Thread") == "fragment" }

// renderTurns writes turns and nothing else — no layout, no lid, no rail.
func renderTurns(w http.ResponseWriter, turns []turnView) {
	t := pages["thread"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	for _, v := range turns {
		if err := t.ExecuteTemplate(w, "turn", v); err != nil {
			slog.Error("drawing a turn", "turn", v.ID, "error", err)
			return
		}
	}
}
```

In `threadSayHandler`, `threadMoodHandler` and `nowActHandler`, replace the final redirect with:

```go
		if wantsFragment(r) {
			// The new turns, and the live edge moves to the last of them.
			vs := turnViews(written)
			if len(vs) > 0 {
				vs[len(vs)-1].Live = true
			}
			renderTurns(w, vs)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
```

where `written` is the slice of turns that handler just appended.

- [ ] **Step 4: Write the module**

Create `internal/web/static/thread.js`:

```js
// The swap. JavaScript is required here — the progressive-enhancement rule was
// retired with Principle 2 on 24 August 2026 — and this file is what makes a
// press feel like a press instead of a page load.
//
// It posts the same form to the same URL the browser would have. The only
// difference is one header, and what comes back is the same HTML the page
// would have rendered for those turns. There is no JSON and no template here;
// a second description of a card is how the two ends drift apart.
(() => {
  const thread = document.getElementById("thread");
  const say = document.getElementById("threadsay");
  if (!thread) return;

  function announce(what) {
    if (say) say.textContent = what;
  }

  function toTheEnd() {
    const last = thread.lastElementChild;
    if (last) last.scrollIntoView({ block: "end", behavior: "smooth" });
  }

  // Controls belong to the live edge alone. When new turns arrive, the turns
  // that were the edge stop being it — the same rule the server renders by,
  // applied to what is already on the screen.
  function retire() {
    thread.querySelectorAll(".turn .acts, .turn .chips").forEach((el) => el.remove());
  }

  async function send(form) {
    const body = new URLSearchParams(new FormData(form));
    const submitter = form.__submitter;
    if (submitter && submitter.name) body.set(submitter.name, submitter.value);

    const res = await fetch(form.action, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        "X-Thread": "fragment",
      },
      body,
    });
    if (!res.ok) throw new Error("the press did not land");
    return res.text();
  }

  document.addEventListener("submit", async (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement) || form.method.toLowerCase() !== "post") return;

    event.preventDefault();
    try {
      const html = await send(form);
      retire();
      thread.insertAdjacentHTML("beforeend", html);
      if (form.classList.contains("dock")) form.reset();
      toTheEnd();
      const last = thread.querySelector(".turn:last-child .bub");
      announce(last ? last.textContent : "");
    } catch (err) {
      // A press that did not land must not look like one that did. The form
      // goes through the ordinary way, which shows whatever the server says.
      form.submit();
    }
  });

  // Which button was pressed, since a form with three buttons posts different
  // things depending on which one. The submit event does not carry it in every
  // browser, so it is caught on the way down.
  document.addEventListener("click", (event) => {
    const button = event.target.closest("button[type=submit], button:not([type])");
    if (button && button.form) button.form.__submitter = button;
  }, true);

  toTheEnd();
})();
```

- [ ] **Step 5: Load it**

In `internal/web/templates/layout.html`, beside the existing `pile.js` tag:

```html
<script src="/static/thread.js?v={{.V}}" defer></script>
```

- [ ] **Step 6: Run the tests**

```bash
go test ./internal/web/ -run Fragment -v
go test -tags browser ./internal/web/ -run ThreadGrows -v
```

- [ ] **Step 7: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestAFragmentAndAPageRenderTheSameCard` | make `renderTurns` write `"<p>"+v.Words+"</p>"` directly instead of executing the template | fragment does not contain `<p class="bub">` |
| `TestAFragmentPostAnswersWithTheNewTurnsAndNoRedirect` | remove the `wantsFragment` branch | `expected 200, actual 303` |
| `TestAnOrdinaryPostStillRedirects` | make `wantsFragment` return `true` always | `expected 303, actual 200` |
| `TestTheThreadGrowsWithoutANavigation` | delete `event.preventDefault()` from `thread.js` | `window.__stillHere` is undefined after the navigation |

The browser mutation is the one that matters — it is the only proof the JavaScript is doing anything, and the naive version of this test passes without it.

- [ ] **Step 8: Commit**

```bash
git add internal/web/static/thread.js internal/web/thread.go \
        internal/web/thread_test.go internal/web/browser_test.go \
        internal/web/templates/layout.html
git commit -m "feat: a press stops being a page load"
```

---

### Task 8: Home comes out, and the records are amended

**Files:**
- Delete: `internal/web/home.go`, `internal/web/templates/home.html`, `internal/web/templates/askbuddy.html`, `internal/web/templates/coach.html`
- Modify: `internal/web/pile.go` (routes), `internal/web/render.go` (`pages`, `askBuddyAbout`, `placeName`, `elsewhere`), `internal/web/templates/layout.html` (the acorn), `internal/web/coach.go`, `internal/web/contrast_test.go`, `PRODUCT.md`, `DESIGN.md`, `docs/roadmap.md`

**Interfaces:** none new. This task removes.

- [ ] **Step 1: Delete the routes and the handler**

Remove from `internal/web/pile.go`: the `/buddy`, `/buddy/close` and `/coach` registrations. Keep `/buddy/say`, `/buddy/badly`, `/buddy/do` — the model plumbing behind them is reached from the thread in phase 2, and deleting it now would be deleting work that is about to be needed. Remove `homeHandler` entirely.

Remove from `internal/web/render.go`: the `"home"` and `"coach"` entries in `pages`, the `askBuddyAbout` function and the `AskBuddy` field, and every `{{template "askbuddy" .}}` call in the remaining templates.

In `placeName`, add `case "thread": return "home"` and leave the rest.

- [ ] **Step 2: Take the acorn out of the lid**

In `internal/web/templates/layout.html`, delete the two `lidbtn tobuddy` blocks (around lines 120 and 133). Buddy is the page now; a lid button to the page you are on is furniture — the same argument the lid already makes about not linking home from home.

- [ ] **Step 3: Run everything and fix what falls over**

```bash
go build ./... && go vet ./...
go test ./...
TEST_DATABASE_URL=... go test -tags integration ./...
go test -tags browser ./internal/web/
```

Expect failures in `internal/web/contrast_test.go` (it walks thirteen screens by name, and two of them are gone) and in any appearance snapshot naming `home`. Update the contrast walk's screen list to `thread` plus the eleven that remain — do not delete a screen from the list to make it pass.

- [ ] **Step 4: Amend `PRODUCT.md`**

Two edits, in the style the no-list-screen rule was amended in on line 280 — strike through, keep the original text readable, and date the change.

At line 113, the *Never a count* constraint:

```markdown
- ~~**Never a count.** No badge, no total, no "N to review", no page count. A
  capped list may say *that* there is more, never *how much* more. This is the
  single rule most likely to be broken by accident.~~

  **Retired by the owner on 24 August 2026.** Counts are permitted on every
  surface: the doors carry what is waiting behind them, and Buddy says how many
  of anything it likes. Principle 5's opening on 20 August had already permitted
  counts in *speech*; this is the other half.

  What the rule protected is real and is now unprotected, stated in the shape it
  will actually arrive in: a number beside a door, with an implied target of
  zero, that grows while nobody is looking and that a bad week makes worse. The
  argument it was traded for is that not knowing how much is waiting is its own
  weight. If the doors start reading as a scoreboard, this is the decision to
  reverse, and it reverses cleanly — the numbers are computed at render time and
  stored nowhere.
```

At line 393, Principle 2:

```markdown
2. ~~**Nothing accrues that can be destroyed.** No counter, no streak, no
   percentage — not on any surface, in any form.~~ **Retired by the owner on
   24 August 2026**, together with the *Never a count* constraint above. See
   there for what it protected and what reverses it. What survives: nothing in
   this product is ever marked late, and no number here is a score.
```

Then find the Positioning paragraph that rests on this principle — it says the rule is the thing a neighbouring product could not copy without abandoning its own metrics — and rewrite it rather than leaving it standing. What the product still refuses that a metrics-driven one cannot: nothing is marked late, stopping partway is a normal ending, and no number is a score.

- [ ] **Step 5: Amend `DESIGN.md`**

- Strike through and date the progressive-enhancement requirement. Add: *the thread requires JavaScript; handlers return HTML from the same templates whether the browser asked for a page or a fragment, so there is still exactly one description of a card.*
- Replace home's fixed order (line 748) and the four-doors item (line 760) with the rail: sticky under the lid, four equals at every width, art-beside-name above 620px and art-over-name below, the count as a pill on the door.
- Amend The One Title Rule (line 632): the thread has no `<h1>`, on home's existing exemption, and a turn that opens a place carries that place's name as an `<h2>`.
- Add **The live edge** as a named rule: only the newest Buddy turn carries controls.
- Relax the door-art guard rail that forbids depicting a count — it now covers the drawing only, since the pill beside it carries a number.

- [ ] **Step 6: Move the roadmap entry**

In `docs/roadmap.md`, move phase 1 of the thread from Open to Shipped, naming the version it ships in.

- [ ] **Step 7: Full run, then commit**

```bash
go build ./... && go vet ./... && go test ./...
TEST_DATABASE_URL=... go test -tags integration ./...
go test -tags browser ./internal/web/
```

```bash
git add -A
git commit -m "feat: home is the conversation now"
```

- [ ] **Step 8: Open the pull request**

```bash
gh pr create --base main --head feat/the-thread \
  --title "feat: the whole app is a conversation" \
  --body "..."
```

The body states the two retired rules with their dates, links the spec, and lists what a reviewer should mutation-test first: the live edge, the chore count going through `DueChores`, and the browser test's `preventDefault`.

---

## Self-Review

**Spec coverage.** Every phase-1 item in the spec's *Staging* entry maps to a task: rail with counts (2, 3), check-in (5), offer (6), dock (4), `turns` table (1), paging backwards (1, 3), fragment swap (7), home/`askbuddy`/acorn deleted (8), records amended (8). The spec's *Buddy has two voices* is phase 2 — `/buddy/say` is deliberately left routed in Task 8 so the model plumbing survives the deletion.

**Not in this plan, and deliberately:** the number × unit picker, the day/time picker, chores and tasks as messages, search as a turn, `/at/{id}`, and the notification's destination. All are phases 2–4.

**Type consistency.** `actView` changes shape in Task 6 (from `Name`/`Value` to `Fields map[string]string`); Task 6 says so and says to update Task 3's uses. Every other type — `Turn`, `Speaker`, `Waiting`, `turnView`, `cardView`, `doorView`, `shown` — is introduced once and used unchanged. `chipView` is pre-existing and Tasks 3 and 5 both say to extend it in place rather than declare a second.

**Known unverified names.** These are used above and must be checked against the tree before use, not assumed: `Routes` (the registration function in `internal/web/pile.go`), `fakeStore`, `testOptions`, `freshStore`, `insertItem`, `anotherPerson`, `squirrel.ParseMood`, the acknowledgement wording function, `offerFor`'s signature, `offerView`'s fields, and the browser helpers in `internal/web/browser_test.go`. Where a name here does not exist, use the real one — do not add a duplicate helper.
