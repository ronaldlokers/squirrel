# A fixed point you can put things on — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A note can point at a fixed point; the appointment gets a screen showing what to take and the notes pointing at it; the leave-by notification lands there instead of the front door.

**Architecture:** One nullable column, `items.moment_id`. A note keeps everything it already has — capture, photograph, editing, search, undo — and gains a pointer. The pile hides notes whose appointment is still ahead, and shows them again once it is over, as a read rule with no write and no new state value. Two screens: `/at` lists what is coming, `/at/{id}` is one appointment.

**Tech Stack:** Go 1.26, pgx against Postgres 17, `html/template`, embedded migrations, no frontend framework. Tests: `testify`, integration behind `-tags=integration`, browser behind `-tags=browser`.

**Spec:** `docs/superpowers/specs/2026-08-24-fixed-point-detail-design.md`

## Global Constraints

- **Never a count.** No badge, no total, no "N coming", no page count — on either new screen, in the door's name, or in its sub-line. This is the product's hardest rule.
- **No new value in `items.state`.** The pointer is the disposition.
- **Nothing past on `/at`.** Only `starts_at > now and done_at is null`.
- **Never "late", never red, no urgency copy.** Nothing on these screens has been missed.
- **Every state transition reverses.** Detaching returns the note to the pile.
- **Cream, not white.** No surface is white or grey; card stock `#fdecd4`, field purple, type on cream is `#1c110b`, secondary type on cream is `#58413d`.
- **Type on Acorn Orange is the outline ink** `#1c110b`, never paper — The Orange Ink Rule.
- **3px outline on every object**, 2px only for a small control inside a card.
- **Every interactive element ≥44px**, and 16px minimum font on any field (iOS zooms below it).
- **Contrast ≥4.5:1**, enforced by `TestEveryWordCanBeRead` — add both new paths to its screen list.
- **`DESIGN.md` must change in the same PR** as any change to `internal/web/static/pile.css` or `internal/web/templates/*.html`, or CI blocks the merge.

---

## File Structure

| File | Responsibility |
| --- | --- |
| `internal/squirrel/migrations/0026_moment_notes.sql` | the column and its index |
| `internal/squirrel/moments.go` | `AttachNote`, `DetachNote`, `NotesFor`, `Upcoming` |
| `internal/squirrel/notes.go` | the pile's clause, and where a note is attached |
| `internal/web/at.go` | both handlers, new file — `held.go` is the model |
| `internal/web/templates/at.html` | `/at`, the list |
| `internal/web/templates/atone.html` | `/at/{id}`, one appointment |
| `internal/web/static/sw.js` | the notification's destination |
| `internal/web/templates/home.html` | the fourth door |
| `internal/web/static/pile.css` | four across, two-by-two on a phone |

---

### Task 1: The column

**Files:**
- Create: `internal/squirrel/migrations/0026_moment_notes.sql`
- Test: `internal/squirrel/momentnotes_test.go`

**Interfaces:**
- Produces: `items.moment_id bigint null references moments(id) on delete set null`

- [ ] **Step 1: Write the failing test**

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

// Deleting an appointment must never delete the owner's words. The note
// returns to the pile instead, which is what happens when it is over anyway.
func TestDeletingAFixedPointKeepsTheNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: time.Now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)
	id := taskOf(t, store, p, "the referral letter")

	_, err = store.Pool().Exec(ctx, `update items set moment_id = $1 where id = $2`, m.ID, id)
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx, `delete from moments where id = $1`, m.ID)
	require.NoError(t, err)

	var moment *int64
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select moment_id from items where id = $1`, id).Scan(&moment))
	require.Nil(t, moment, "the words survive the appointment")
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test go test -tags=integration -run TestDeletingAFixedPoint ./internal/squirrel/`

Expected: FAIL — `column "moment_id" of relation "items" does not exist`.

Start Postgres first if it is not running, and wait for it to accept queries rather than for the container to exist:

```bash
docker run --rm -d --name squirrel-test-db -p 55432:5432 \
  -e POSTGRES_USER=squirrel -e POSTGRES_PASSWORD=squirrel \
  -e POSTGRES_DB=squirrel_test postgres:17-alpine
until docker exec squirrel-test-db psql -U squirrel -d squirrel_test -c 'select 1' >/dev/null 2>&1; do sleep 1; done
```

- [ ] **Step 3: Write the migration**

```sql
-- A note can point at a fixed point.
--
-- The pointer rather than new columns on `moments`, and the difference is the
-- product's deepest rule: a thought that lives on an appointment instead of in
-- the pile is a thought `!find` cannot reach. This way a note keeps everything
-- it already has — capture, photograph, editing, search, undo — and only gains
-- somewhere to be.
--
-- No new value in `items.state`, deliberately. The pointer is the disposition:
-- a note with somewhere to be is not waiting to be decided about, and one whose
-- appointment has passed is waiting again with no transition to write.
--
-- `on delete set null` rather than cascade. Deleting an appointment must never
-- delete the owner's words; the note returns to the pile, which is the same
-- thing that happens when the appointment is simply over.
alter table items add column moment_id bigint references moments(id) on delete set null;

-- Partial, because the overwhelming majority of notes point at nothing and an
-- index over all of them would be mostly nulls.
create index items_moment_id_idx on items (moment_id) where moment_id is not null;
```

- [ ] **Step 4: Run it and watch it pass**

Run: the same command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/migrations/0026_moment_notes.sql internal/squirrel/momentnotes_test.go
git commit -m "feat: a note can point at a fixed point"
```

---

### Task 2: Attaching, detaching, and reading back

**Files:**
- Modify: `internal/squirrel/moments.go`
- Test: `internal/squirrel/momentnotes_test.go`

**Interfaces:**
- Consumes: `items.moment_id` from Task 1.
- Produces:
  - `func (s *Store) AttachNote(ctx context.Context, personID, itemID, momentID int64) (bool, error)`
  - `func (s *Store) DetachNote(ctx context.Context, personID, itemID int64) (bool, error)`
  - `func (s *Store) NotesFor(ctx context.Context, personID, momentID int64) ([]Item, error)`
  - Each returns `false` when the row is not the caller's, matching `HoldItem`'s existing shape.

- [ ] **Step 1: Write the failing tests**

```go
func TestANoteCanBePointedAtAFixedPointAndBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := taskOf(t, store, p, "the referral letter")

	ok, err := store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)
	require.True(t, ok)

	notes, err := store.NotesFor(ctx, p, m.ID)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, "the referral letter", notes[0].RawText)

	ok, err = store.DetachNote(ctx, p, id)
	require.NoError(t, err)
	require.True(t, ok)

	notes, err = store.NotesFor(ctx, p, m.ID)
	require.NoError(t, err)
	require.Empty(t, notes, "detaching is the reversal, and every transition here reverses")
}

// Somebody else's row is not yours to move, and saying so with a boolean
// rather than an error is the shape HoldItem already uses.
func TestPointingAtAFixedPointIsOnlyEverYourOwn(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	stranger := personNamed(t, store, "someone-else")

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := taskOf(t, store, p, "the referral letter")

	ok, err := store.AttachNote(ctx, stranger, id, m.ID)
	require.NoError(t, err)
	require.False(t, ok)
}

// aFixedPoint keeps the fixtures honest: a bare Moment{} has zero travel and
// ready, so WarnAt equals the start time and every window assertion is wrong.
func aFixedPoint(t *testing.T, store *squirrel.Store, personID int64, label string, in time.Duration) squirrel.Moment {
	t.Helper()
	m, err := store.CreateMoment(context.Background(), personID, squirrel.Moment{
		Label: label, Starts: time.Now().Add(in),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)
	return m
}
```

`personNamed` may not exist. If it does not, add it beside `owner` in `internal/squirrel/testsupport_test.go`, following `owner`'s own body — read it first and mirror it.

- [ ] **Step 2: Run and watch them fail**

Run: `TEST_DATABASE_URL=... go test -tags=integration -run 'TestANoteCanBePointed|TestPointingAtAFixedPoint' ./internal/squirrel/`

Expected: FAIL — `store.AttachNote undefined`.

- [ ] **Step 3: Implement**

Add to `internal/squirrel/moments.go`:

```go
// AttachNote points a note at a fixed point, and answers whether it was yours
// to point.
//
// The person is in the where clause rather than checked first, so there is no
// window between reading a row and writing it, and no way for a caller to
// forget the check.
func (s *Store) AttachNote(ctx context.Context, personID, itemID, momentID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set moment_id = $3
		 where id = $2 and person_id = $1
		   and exists (select 1 from moments where id = $3 and person_id = $1)`,
		personID, itemID, momentID)
	if err != nil {
		return false, fmt.Errorf("pointing a note at a fixed point: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// DetachNote puts it back in the pile. Every transition in this product
// reverses, and this is the reversal — there is no previous value to remember
// because the pointer was the whole of the change.
func (s *Store) DetachNote(ctx context.Context, personID, itemID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`update items set moment_id = null where id = $2 and person_id = $1`,
		personID, itemID)
	if err != nil {
		return false, fmt.Errorf("returning a note to the pile: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// NotesFor is what is pointing at one fixed point, newest first like every
// other list in this product.
func (s *Store) NotesFor(ctx context.Context, personID, momentID int64) ([]Item, error) {
	items, _, err := s.itemsWhere(ctx,
		`person_id = $1 and moment_id = $2`, notesForLimit, personID, momentID)
	return items, err
}

// notesForLimit is a bound rather than a page: nothing here says how many there
// are, and a limit that is never reached in practice is a limit nobody sees.
const notesForLimit = 50
```

- [ ] **Step 4: Run and watch them pass**

Run: the same command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/moments.go internal/squirrel/momentnotes_test.go internal/squirrel/testsupport_test.go
git commit -m "feat: point a note at a fixed point, and back"
```

---

### Task 3: The pile hides what has somewhere to be

**Files:**
- Modify: `internal/squirrel/notes.go:188-190` (`OpenItems`), `:258-267` (`OpenItemsAfter`)
- Test: `internal/squirrel/momentnotes_test.go`

**Interfaces:**
- Consumes: `AttachNote` from Task 2.
- Produces: the shared SQL fragment `stillMine`, used by both pile queries.

- [ ] **Step 1: Write the failing tests**

```go
// A note with somewhere to be is not waiting to be decided about.
func TestThePileHidesANoteThatIsOnAFixedPoint(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := noteOf(t, store, p, "the referral letter")

	before, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, before, 1)

	_, err = store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)

	after, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, after, "it has somewhere to be")
}

// And it is waiting again the moment the appointment is over, with no write
// and nothing running on a schedule. This is the half that is a read rule, so
// it is the half most worth a test.
func TestThePileShowsItAgainOnceTheFixedPointIsPast(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	past := aFixedPoint(t, store, p, "dentist", -2*time.Hour)
	id := noteOf(t, store, p, "the referral letter")
	_, err := store.AttachNote(ctx, p, id, past.ID)
	require.NoError(t, err)

	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, items, 1, "the occasion happened; it is a thought again")
}

// The same rule, on the pile's own paging query. Two queries that disagree
// about what the pile holds is the bug this product's records warn about most.
func TestPagingThroughThePileObeysTheSameRule(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	first := noteOf(t, store, p, "buy milk")
	attached := noteOf(t, store, p, "the referral letter")
	_, err := store.AttachNote(ctx, p, attached, m.ID)
	require.NoError(t, err)

	items, _, err := store.OpenItemsAfter(ctx, p, first, 20)
	require.NoError(t, err)
	require.Empty(t, items)
}
```

`noteOf` is `taskOf` without the `SetItemKind` call — add it beside `taskOf` in `internal/squirrel/pick_test.go`, mirroring its body exactly and leaving `kind` at the column default.

- [ ] **Step 2: Run and watch them fail**

Run: `TEST_DATABASE_URL=... go test -tags=integration -run 'TestThePileHides|TestThePileShowsItAgain|TestPagingThroughThePile' ./internal/squirrel/`

Expected: the first and third FAIL (the note is still listed); the second passes already, and that is fine — it is the guard, not the change.

- [ ] **Step 3: Implement**

In `internal/squirrel/notes.go`, above `OpenItems`:

```go
// stillMine is the pile's clause about fixed points, written once because two
// queries read the pile and a pile that disagrees with itself is the bug this
// package's comments warn about most.
//
// A note pointing at an appointment that is still ahead has somewhere to be, so
// it is not waiting to be decided about. Once the appointment is over it is a
// thought again — and that is a read rule rather than a write, so nothing runs
// on a schedule, nothing needs a migration to catch up, and an appointment
// deleted outright leaves its notes here by the same sentence.
const stillMine = `(
	moment_id is null
	or not exists (
		select 1 from moments
		 where moments.id = items.moment_id
		   and moments.done_at is null
		   and moments.starts_at > now()
	)
)`
```

Then thread it into both:

```go
func (s *Store) OpenItems(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx,
		`person_id = $1 and kind = 'note' and state = 'open' and `+stillMine, limit, personID)
}
```

and in `OpenItemsAfter`, append `and `+stillMine to its existing where clause, leaving the `afterID` subquery untouched.

- [ ] **Step 4: Run and watch all three pass**

Run: the same command. Expected: PASS.

Then the whole package, because this changes the query behind the most-read screen in the product:

Run: `TEST_DATABASE_URL=... go test -tags=integration ./internal/squirrel/`
Expected: ok.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/notes.go internal/squirrel/momentnotes_test.go internal/squirrel/pick_test.go
git commit -m "feat: the pile hides a note that has somewhere to be"
```

---

### Task 4: What is coming

**Files:**
- Modify: `internal/squirrel/moments.go`
- Test: `internal/squirrel/momentnotes_test.go`

**Interfaces:**
- Produces: `func (s *Store) Upcoming(ctx context.Context, personID int64, now time.Time, limit int) ([]Moment, error)`

- [ ] **Step 1: Write the failing test**

```go
// Only what is still coming. Nothing past, nothing done — because a thing you
// have not reached yet is not a thing you are late for, and that is the whole
// of what makes this list allowed to exist.
func TestUpcomingHoldsOnlyWhatIsStillAhead(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	soon := aFixedPoint(t, store, p, "dentist", 2*time.Hour)
	later := aFixedPoint(t, store, p, "school run", 30*time.Hour)
	aFixedPoint(t, store, p, "yesterday's thing", -26*time.Hour)

	done := aFixedPoint(t, store, p, "already left", 4*time.Hour)
	require.NoError(t, store.MomentDone(ctx, p, done.ID, now))

	got, err := store.Upcoming(ctx, p, now, 20)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, soon.ID, got[0].ID, "soonest first")
	require.Equal(t, later.ID, got[1].ID)
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `TEST_DATABASE_URL=... go test -tags=integration -run TestUpcomingHoldsOnly ./internal/squirrel/`
Expected: FAIL — `store.Upcoming undefined`.

- [ ] **Step 3: Implement**

```go
// Upcoming is what is still ahead, soonest first.
//
// The list this product spent its whole life refusing, and the refusal is worth
// keeping in view: a browsable set of your appointments is a calendar, and a
// calendar is a thing you are behind on. What makes this one allowed is that it
// holds nothing you can be behind on — `starts_at > now` and `done_at is null`,
// so everything in it is still ahead of you.
//
// It returns rows and never a total. Nothing above it may count them.
func (s *Store) Upcoming(ctx context.Context, personID int64, now time.Time, limit int) ([]Moment, error) {
	const q = `
		select id, person_id, label, starts_at, travel_secs, ready_secs,
		       coalesce(bring, ''), said_at is not null
		  from moments
		 where person_id = $1 and done_at is null and starts_at > $2
		 order by starts_at limit $3`

	rows, err := s.pool.Query(ctx, q, personID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("reading what is coming: %w", err)
	}
	defer rows.Close()

	out := []Moment{}
	for rows.Next() {
		m, err := scanMomentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
```

`scanMoment` takes a query string and scans one row; this needs the same field order against a `pgx.Rows`. Extract the scan body of `scanMoment` into `scanMomentRow(row interface{ Scan(...any) error }) (Moment, error)` — including the `defaultTravel` / `Guessed` defaulting — and have `scanMoment` call it, so the two cannot drift.

- [ ] **Step 4: Run and watch it pass**

Run: the same command, then the whole package. Expected: PASS, ok.

- [ ] **Step 5: Commit**

```bash
git add internal/squirrel/moments.go internal/squirrel/momentnotes_test.go
git commit -m "feat: what is coming, and only what is coming"
```

---

### Task 5: `/at/{id}` — one fixed point

**Files:**
- Create: `internal/web/at.go`, `internal/web/templates/atone.html`
- Modify: `internal/web/render.go` (view fields), `internal/web/pile.go` (routes), `internal/web/static/pile.css`, `DESIGN.md`
- Test: `internal/web/at_test.go`

**Interfaces:**
- Consumes: `NotesFor`, `DetachNote`, `AttachNote` (Task 2); `Moment.LeaveAt`, `Moment.Open`, `LeaveWords`.
- Produces: routes `GET /at/{id}`, `POST /at/{id}/note`, `POST /at/{id}/detach`; view fields `Moment *momentView`, `Attached []itemView`.

- [ ] **Step 1: Write the failing tests**

```go
package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestAFixedPointShowsWhenToLeaveAndWhatToTake(t *testing.T) {
	f := withMoment(&squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute, Bring: "keys, wallet",
	})
	body := mounted(t, f).call(t, "GET", "/at/4", nil).Body.String()

	require.Contains(t, body, "dentist")
	require.Contains(t, body, "keys, wallet")
	require.NotContains(t, body, "LEAVING", "hours out, there is nothing to press")
}

func TestLeavingIsOfferedInsideTheWindow(t *testing.T) {
	f := withMoment(&squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(10 * time.Minute),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	body := mounted(t, f).call(t, "GET", "/at/4", nil).Body.String()

	require.Contains(t, body, "LEAVING")
}

func TestTheNotesPointingAtItAreShown(t *testing.T) {
	f := withMoment(&squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	f.attached = []squirrel.Item{{ID: 9, RawText: "the referral letter"}}

	body := mounted(t, f).call(t, "GET", "/at/4", nil).Body.String()
	require.Contains(t, body, "the referral letter")
}

// Anything typed here is an ordinary note that happens to point at this
// appointment. No picker, because a picker needs a browsable list of
// appointments to pick from, which is the thing the record refuses.
func TestTypingIntoAFixedPointKeepsANotePointingAtIt(t *testing.T) {
	f := withMoment(&squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	m := mounted(t, f)

	form := url.Values{"words": {"the referral letter"}}
	res := m.call(t, "POST", "/at/4/note", strings.NewReader(form.Encode()))

	require.Equal(t, 303, res.Code)
	require.Equal(t, []string{"the referral letter"}, f.inserted)
	require.Equal(t, []int64{4}, f.attachedTo, "it points at the appointment it was typed on")
}

// There is no count on this screen, in any form. The hardest rule in the
// product, on its newest surface.
func TestAFixedPointNeverCountsAnything(t *testing.T) {
	f := withMoment(&squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	f.attached = []squirrel.Item{
		{ID: 9, RawText: "one"}, {ID: 10, RawText: "two"}, {ID: 11, RawText: "three"},
	}
	body := mounted(t, f).call(t, "GET", "/at/4", nil).Body.String()

	require.NotContains(t, body, "3 ")
	require.NotContains(t, body, "three notes")
}
```

`withMoment` and the `fakeStore` fields `attached`, `attachedTo` need adding to `internal/web/testsupport_test.go`. Follow the existing fakes exactly: a field per thing written, so a test asserts on the write rather than on a rendering of it.

```go
func withMoment(m *squirrel.Moment) *fakeStore { return &fakeStore{moment: m} }

func (f *fakeStore) MomentByID(_ context.Context, _, id int64) (squirrel.Moment, bool, error) {
	if f.err != nil {
		return squirrel.Moment{}, false, f.err
	}
	if f.moment == nil || f.moment.ID != id {
		return squirrel.Moment{}, false, nil
	}
	return *f.moment, true, nil
}

func (f *fakeStore) NotesFor(_ context.Context, _, _ int64) ([]squirrel.Item, error) {
	return f.attached, f.err
}

func (f *fakeStore) AttachNote(_ context.Context, _, _, momentID int64) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.attachedTo = append(f.attachedTo, momentID)
	return true, nil
}

func (f *fakeStore) DetachNote(_ context.Context, _, itemID int64) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.detached = append(f.detached, itemID)
	return true, nil
}
```

`MomentByID` does not exist yet — add it to `internal/squirrel/moments.go` alongside `NextMoment`, using `scanMoment` with `where id = $2 and person_id = $1`, and add it to the `Store` interface in `internal/web/web.go`.

- [ ] **Step 2: Run and watch them fail**

Run: `go test ./internal/web/ -run 'TestAFixedPoint|TestLeavingIsOffered|TestTheNotesPointing|TestTypingIntoAFixedPoint'`
Expected: FAIL — 404, because the route does not exist.

- [ ] **Step 3: Implement the handlers**

Create `internal/web/at.go` following `held.go`'s shape exactly — `opts.person()`, `fail(w, err)` on a store error, `renderWith(w, r, s, opts, "atone", view{...})`. Three handlers:

- `atOneHandler` — reads `MomentByID` and `NotesFor`, and when the moment is not found answers `w.WriteHeader(http.StatusNotFound)` and returns. That is the convention in `act.go:126` and `chores.go:81`; there is no `notFound` helper and this plan is not the place to invent one.
- `atNoteHandler` — the slot's POST. Reuse the capture path the home slot uses, then `AttachNote` with the id from the route. Redirect 303 to `/at/{id}`.
- `atDetachHandler` — `DetachNote`, redirect 303 to `/at/{id}`.

Register in `internal/web/pile.go` beside `/held`:

```go
// One fixed point, and what is pointing at it. The notification lands here:
// see sw.js, and DESIGN.md for why this replaces the front door.
m.Get("/at/{id}", guard(opts, atOneHandler(s, opts)))
m.Post("/at/{id}/note", guard(opts, sameOrigin(atNoteHandler(s, opts))))
m.Post("/at/{id}/detach", guard(opts, sameOrigin(atDetachHandler(s, opts))))
```

Both POSTs go through `sameOrigin`, like every other write in this file.

- [ ] **Step 4: Write the template**

`internal/web/templates/atone.html`, following `held.html`'s structure: `{{define "content"}}`, one `<h1 class="deckhead">` carrying the label under The One Title Rule, then when-it-starts and when-to-leave, then what to take on **its own line** rather than after a middot, then the slot, then the attached notes as `.rcard`s each with a "back in the pile" control, then `{{template "stopping" .}}`.

`LEAVING` is rendered only when the view says the window is open — compute that in the handler with `m.Open(now())` rather than in the template, because a template that does arithmetic is a template that can disagree with the core.

- [ ] **Step 5: Style it**

Reuse existing classes wherever the shape already exists — `.rcard` for an attached note, `.slot` for the box, `.abtn` for `LEAVING`. Add only what is genuinely new: the take-line. Give it the Meta role in headphone brown, and the label uppercase:

```css
  /* What to take, on its own line rather than as a clause after a middot.
     It is the thing you are standing in the hall without, so it gets to be an
     element rather than a tail. */
  .takeline {
    margin: 6px 0 0;
    font-size: 11.5px; letter-spacing: .1em; text-transform: uppercase;
    font-variation-settings: var(--precise), "wght" 750;
    color: var(--brown);
  }
  .takeline b {
    font-variation-settings: var(--casual), "wght" 520;
    letter-spacing: 0; text-transform: none; font-size: 15px;
  }
```

- [ ] **Step 6: Amend DESIGN.md**

Add `/at/{id}` to the screen list and record the take-line as a role. CI blocks the merge without this — a template changed and `DESIGN.md` did not.

- [ ] **Step 7: Run everything**

```bash
make check
go test -tags=browser -count=1 ./internal/web/
```

Expected: ok. The appearance snapshot will fail if it records anything that moved — read the diff before regenerating, and regenerate deliberately with `APPEARANCE=rewrite`.

- [ ] **Step 8: Commit**

```bash
git add internal/web/at.go internal/web/at_test.go internal/web/templates/atone.html \
        internal/web/testsupport_test.go internal/web/render.go internal/web/pile.go \
        internal/web/web.go internal/web/static/pile.css internal/squirrel/moments.go DESIGN.md
git commit -m "feat: a screen for one fixed point"
```

---

### Task 6: `/at` — the list

**Files:**
- Modify: `internal/web/at.go`, `internal/web/pile.go`, `DESIGN.md`
- Create: `internal/web/templates/at.html`
- Test: `internal/web/at_test.go`

**Interfaces:**
- Consumes: `Upcoming` (Task 4).
- Produces: route `GET /at`.

- [ ] **Step 1: Write the failing tests**

```go
func TestWhatIsComingListsTheSoonestFirst(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	body := mounted(t, f).call(t, "GET", "/at", nil).Body.String()

	require.Less(t, strings.Index(body, "dentist"), strings.Index(body, "school run"))
	require.Contains(t, body, `href="/at/4"`)
}

// Never a count, and never a word about being behind. Everything here is still
// ahead of you, which is the only reason this list is allowed to exist.
func TestWhatIsComingCountsNothingAndScoldsNobody(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	body := mounted(t, f).call(t, "GET", "/at", nil).Body.String()

	for _, banned := range []string{"late", "overdue", "2 coming", "you have"} {
		require.NotContains(t, strings.ToLower(body), banned)
	}
}

func TestNothingComingIsAnEmptyStateAndNotAnEncouragement(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/at", nil).Body.String()

	require.Contains(t, body, "empty")
	require.NotContains(t, strings.ToLower(body), "plan")
}
```

- [ ] **Step 2: Run and watch them fail**

Run: `go test ./internal/web/ -run 'TestWhatIsComing|TestNothingComing'`
Expected: FAIL — 404.

- [ ] **Step 3: Implement**

`atHandler` in `internal/web/at.go`: `Upcoming(ctx, personID, now(), upcomingLimit)`, then `renderWith(..., "at", view{Here: "at", Scrolling: true, Upcoming: rows})`. Register `m.Get("/at", guard(opts, atHandler(s, opts)))` before the `/at/{id}` route.

`internal/web/templates/at.html`: `<h1 class="deckhead">` with the screen's name, then one `<a class="choreHit" href="/at/{{.ID}}">` per row — the Door's grammar at one line high, which a chore in search results already borrows. Empty renders the shared empty-state block with the mark, in the Headline role.

- [ ] **Step 4: Run and watch them pass**

Run: the same command. Expected: PASS.

- [ ] **Step 5: Add both screens to the contrast walk**

In `internal/web/contrast_test.go`, add `"/at"` and `"/at/4"` to `auditPaths`, and give `everyScreen()` an upcoming moment and an attached note so both render something.

Run: `go test -tags=browser -count=1 -run TestEveryWordCanBeRead ./internal/web/`
Expected: ok. Any failure names the element, the words and both colours — fix the colour, do not widen the test.

- [ ] **Step 6: Commit**

```bash
git add internal/web/at.go internal/web/at_test.go internal/web/templates/at.html \
        internal/web/pile.go internal/web/contrast_test.go internal/web/testsupport_test.go DESIGN.md
git commit -m "feat: what is coming"
```

---

### Task 7: The notification lands on the appointment

**Files:**
- Modify: `internal/web/static/sw.js`, `internal/squirrel/schedule.go:715-726`, `DESIGN.md`
- Test: `internal/web/notificationclick_test.go`, `internal/squirrel/push_test.go`

**Interfaces:**
- Consumes: `/at/{id}` (Task 5).
- Produces: `Push.URL` carries `/at/{id}`; the worker navigates there.

- [ ] **Step 1: Write the failing test**

```go
// The payload names where to go, and the worker goes there. Both halves,
// because the field has existed and been read by nothing since it was added.
func TestALeaveByPushPointsAtTheFixedPoint(t *testing.T) {
	// in internal/squirrel, beside the other push tests
	m := Moment{ID: 4, Label: "dentist", Starts: at(14, 30), Travel: 20 * time.Minute}
	require.Equal(t, "/at/4", pushFor(m).URL)
}
```

and in `internal/web/notificationclick_test.go`, following the fakes already there, assert the worker navigates to the payload's `url` rather than to `/`.

- [ ] **Step 2: Run and watch them fail**

Run: `go test ./internal/squirrel/ -run TestALeaveByPush` and `go test -tags=browser -count=1 -run Notification ./internal/web/`
Expected: FAIL — `URL` is empty, and the worker navigates to `/`.

- [ ] **Step 3: Implement**

In `internal/squirrel/schedule.go`, set `URL: fmt.Sprintf("/at/%d", m.ID)` on the `Push` built in `MomentTick`.

In `internal/web/static/sw.js`, keep both carve-outs and change only the destination:

```js
// Where the payload says, which is the fixed point this is about.
//
// This used to be the front door, on the argument that a link to something
// already done is worse than a page saying what is true now. That reasoning is
// kept rather than dropped: a fixed point inside its leave-by window is the one
// thing here that cannot be stale, because the notification and the window are
// the same fact. See DESIGN.md.
const target = said.url || "/";
```

and use `target` in place of `"/"` in both the "already there" comparison and the `navigate()` call.

- [ ] **Step 4: Run and watch them pass**

Run: both commands. Expected: PASS.

- [ ] **Step 5: Amend DESIGN.md**

Replace the front-door paragraph with the amended rule and the reason it changed. Quote the old rule; do not delete it.

- [ ] **Step 6: Commit**

```bash
git add internal/web/static/sw.js internal/squirrel/schedule.go internal/squirrel/push_test.go \
        internal/web/notificationclick_test.go DESIGN.md
git commit -m "feat: a leave-by notification lands on the fixed point"
```

---

### Task 8: The fourth door

**Files:**
- Modify: `internal/web/templates/home.html:235-258`, `internal/web/static/pile.css:274`, `:1674-1682`, `DESIGN.md`, `PRODUCT.md`
- Test: `internal/web/home_test.go`, `internal/web/appearance_test.go`

**Interfaces:**
- Consumes: `/at` (Task 6).

- [ ] **Step 1: Write the failing test**

```go
func TestHomeHasAWayToWhatIsComing(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `href="/at"`)
	require.Equal(t, 4, strings.Count(body, `class="door"`))
}

// The doors are equals. Nothing on home may say how many of anything there is,
// and a door is the most tempting place in the product to put a number.
func TestTheDoorsStillCountNothing(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour)},
	}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	doors := body[strings.Index(body, `class="doors"`):]
	require.NotContains(t, doors, "1")
}
```

- [ ] **Step 2: Run and watch them fail**

Run: `go test ./internal/web/ -run 'TestHomeHasAWayToWhatIsComing|TestTheDoorsStillCountNothing'`
Expected: FAIL — three doors, no `/at`.

- [ ] **Step 3: Add the door**

In `home.html`, after the chores door, mirroring the three exactly. The working name is `what is coming` and the sub-line follows the others' grammar. Art: `door-at.png`, drawn by the owner — until it exists, reuse `door-tasks.png` and **say so in the PR** rather than shipping a silent placeholder. Update the `aria-label` from "The three things the box holds".

- [ ] **Step 4: Four across, two-by-two on a phone**

```css
  .doors { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
```

and inside `@media (max-width: 620px)`:

```css
    /* Two by two, because four across is not four equals.
       At 390px the frame is 364px: three cells with a 7px gap measure about
       115px and four measure about 84px, where "what you decided" already wraps
       to two lines at 115px and the art has already stepped down to 50px. So
       the row breaks rather than the cells cramping — it trades "side by side
       at every width" for rearranging instead of shrinking, which keeps the
       four equal to each other, which is what the rule was protecting. */
    .doors { grid-template-columns: 1fr 1fr; }
```

The `:first-child` / `:last-child` tilt rules still apply and still give two of the four a lean; leave them, and check the result rather than assuming.

- [ ] **Step 5: Run and look**

Run: `go test ./internal/web/ -run 'TestHomeHasAWayToWhatIsComing|TestTheDoorsStillCountNothing'` — expect PASS.

Then render home at 1280 and 390 and **look at it**, because this is the only task that changes a screen the owner already likes:

```bash
PREVIEW_DIR=/tmp/doors go test -tags=browser -run TestPreviewDump ./internal/web/
```

(If `TestPreviewDump` no longer exists, stand the screen up however the browser tests do and screenshot with `chromium --headless --screenshot --window-size=390,844`.)

- [ ] **Step 6: Amend both records**

- `DESIGN.md`: the fourth door and the two-by-two phone layout, with the arithmetic.
- `PRODUCT.md`: the no-list-screen rule, amended rather than deleted — quote the old rule, record that the owner overturned it on 24 August 2026, and state the guard rails that replace it.

- [ ] **Step 7: Regenerate the appearance snapshot deliberately**

```bash
APPEARANCE=rewrite go test -tags=browser -run TestTheScreensLookLike ./internal/web/
git diff internal/web/testdata/appearance.json
```

Read the diff. Every line should be a door width or the doors' `grid-template-columns`. Anything else is a change you did not intend.

- [ ] **Step 8: Commit**

```bash
git add internal/web/templates/home.html internal/web/static/pile.css internal/web/home_test.go \
        internal/web/testdata/appearance.json DESIGN.md PRODUCT.md
git commit -m "feat: a fourth door, and two by two on a phone"
```

---

### Task 9: Chat reads the same

**Files:**
- Test: `internal/squirrel/momentnotes_test.go`
- Modify: `internal/squirrel/render.go` only if a test proves it needs it

**Interfaces:**
- Consumes: everything above.

- [ ] **Step 1: Write the tests**

```go
// Two views, one pile. `!notes` and the screen run the same query, so a note
// with somewhere to be is in neither.
func TestChatsPileAgreesWithTheScreen(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := noteOf(t, store, p, "the referral letter")
	_, err := store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)

	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, items)
}

// And the whole justification for this shape over new columns: nothing becomes
// unfindable. A test is what makes that true rather than intended.
func TestFindStillReachesANoteOnAFixedPoint(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := noteOf(t, store, p, "the referral letter")
	_, err := store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)

	hits, _, err := store.SearchItems(ctx, p, "referral", 20)
	require.NoError(t, err)
	require.Len(t, hits, 1, "a note with somewhere to be is still a note you can find")
}
```

- [ ] **Step 2: Run them**

Run: `TEST_DATABASE_URL=... go test -tags=integration -run 'TestChatsPileAgrees|TestFindStillReaches' ./internal/squirrel/`

Expected: both PASS, because Task 3 changed only the pile's query and `SearchItems` deliberately filters by nothing. **If `TestFindStillReaches` fails, stop** — the shape's whole justification has broken and the design needs revisiting, not the test.

- [ ] **Step 3: Commit**

```bash
git add internal/squirrel/momentnotes_test.go
git commit -m "test: chat and the screen agree about a note with somewhere to be"
```

---

### Task 10: The whole path, and the records

**Files:**
- Modify: `docs/roadmap.md`, `docs/pile-screen.md`

- [ ] **Step 1: Run every suite**

```bash
make check
docker run --rm -d --name squirrel-test-db -p 55432:5432 \
  -e POSTGRES_USER=squirrel -e POSTGRES_PASSWORD=squirrel \
  -e POSTGRES_DB=squirrel_test postgres:17-alpine
until docker exec squirrel-test-db psql -U squirrel -d squirrel_test -c 'select 1' >/dev/null 2>&1; do sleep 1; done
TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test make test-integration
make test-browser
docker rm -f squirrel-test-db
```

All must be ok.

- [ ] **Step 2: Run the design detector**

```bash
node ~/.claude/plugins/cache/impeccable/impeccable/4.1.1/skills/impeccable/scripts/detect.mjs --json \
  internal/web/static/pile.css internal/web/templates
```

Expected: `[]`.

- [ ] **Step 3: Update the route table**

`docs/pile-screen.md` carries a route table that has been wrong before — `/enough` was missing from it. Add `/at`, `/at/{id}` and both POSTs, and diff the table against `internal/web/pile.go` line by line while you are in there.

- [ ] **Step 4: Update the roadmap**

Move this out of Open into Shipped, dated, naming the two rules it overturned.

- [ ] **Step 5: Walk it by hand**

With the app running: open home, press the fourth door, open an appointment, type a note into its slot, check it left the pile, put it back, and check it returned. Then check `/at/{id}` on a phone-width window.

- [ ] **Step 6: Commit**

```bash
git add docs/roadmap.md docs/pile-screen.md
git commit -m "docs: what is coming, and the rules it changed"
```

---

## Self-Review

**Spec coverage.** Data → Task 1. Attach/detach/read → Task 2. Pile clause → Task 3. `/at` list and its guard rails → Tasks 4, 6. `/at/{id}`, the slot, take-line, LEAVING-in-window → Task 5. Notification destination → Task 7. Fourth door and two-by-two → Task 8. Chat parity and `!find` → Task 9. Records → Tasks 5, 7, 8, 10.

**One spec item deliberately not a task:** the door's real name and art are the owner's. Task 8 ships the working name and an explicitly-declared placeholder image rather than blocking.

**Type consistency.** `AttachNote(ctx, personID, itemID, momentID) (bool, error)`, `DetachNote(ctx, personID, itemID) (bool, error)`, `NotesFor(ctx, personID, momentID) ([]Item, error)`, `Upcoming(ctx, personID, now, limit) ([]Moment, error)`, `MomentByID(ctx, personID, id) (Moment, bool, error)` — used identically in every task that references them, and each added to `web.Store` in the task that first needs it.

**Known sharp edge:** Task 4 extracts `scanMomentRow` out of `scanMoment`. If that extraction turns out to be awkward against pgx's `Row` and `Rows` types, write the scan out twice with a comment saying why rather than inventing an interface — two honest copies beat a wrong abstraction, and this plan would rather say so than have it discovered mid-task.
