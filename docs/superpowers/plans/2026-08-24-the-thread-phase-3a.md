# The Thread, Phase 3a — the agenda and the day picker

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans.

**Goal:** The agenda joins the conversation and gains the day/time picker — a day grid, time chips, and the composed line — so an appointment can be made for any day rather than only today or tomorrow.

**Why this is 3a and not 3.** The spec's phase 3 bundles the pile with the agenda. The pile is the deck: one-card triage with split, undo, paging, search and seven states, and it is the hardest surface in the product. The day/time picker is the thing that was actually asked for. Splitting them lands the picker now and leaves the deck as its own piece (3b) rather than hiding it inside a larger change.

**Spec:** `docs/superpowers/specs/2026-08-24-the-thread-design.md`, phase 3.

## Global Constraints

Everything from phase 2's plan carries: counts permitted, JavaScript required, one rendering path, history never rewritten, only the newest Buddy turn carries controls, `endsOpen` respected, zero renders as no number, a place-opening turn carries an `<h2>`, and **every test proved by mutation with the observed assertion text recorded**. Branch `feat/the-thread-3a`, never `main`.

## The core problem, stated first

`ParseMoment` builds `starts` from *today's* date and adds a day for "tomorrow" or for a time already gone. There is no way to say a date. The interval picker could compose a sentence because `ParseEveryAsking` already understood any rhythm; the day picker cannot, because "Thursday 27 August" is not sayable.

Two bad answers and one good one:

- **Extend the sentence grammar to take dates.** Widens the bar `momentPattern` deliberately sets — "the marks of a deliberate fixed point are the word *at* or the word *tomorrow*" — and the bar exists so a stray thought is never silently turned into something that interrupts you.
- **Let the picker build a `Moment` itself.** A second place that decides what `14:30` means, which is exactly what `composeEvery` was written to avoid.
- **Anchor the same parser to a different day.** One parser, one definition of what a time means, and the picker gets any day.

---

### Task 1: `MomentOn` — the same sentence, on a chosen day

**Files:** `internal/squirrel/moments.go`, `internal/squirrel/moments_test.go`

**Produces:**
```go
// MomentOn reads the same sentence ParseMoment reads, anchored to a chosen day.
func MomentOn(day time.Time, s string, now time.Time) (Moment, bool)
```
and `ParseMoment(s, now)` becomes `MomentOn(now, s, now)` — literally, so there is one body.

- [ ] **Step 1: failing tests**

```go
func TestMomentOnPutsItOnTheChosenDay(t *testing.T) {
	now := time.Date(2026, 8, 24, 9, 0, 0, 0, time.Local)
	day := time.Date(2026, 8, 27, 0, 0, 0, 0, time.Local)

	m, ok := squirrel.MomentOn(day, "at 14:30 dentist", now)
	require.True(t, ok)
	require.Equal(t, 2026, m.Starts.Year())
	require.Equal(t, time.August, m.Starts.Month())
	require.Equal(t, 27, m.Starts.Day())
	require.Equal(t, 14, m.Starts.Hour())
	require.Equal(t, 30, m.Starts.Minute())
}

// A time already gone rolls to tomorrow when the day is today, and does not
// when the day was chosen: choosing the 27th means the 27th.
func TestAChosenDayIsNotRolledForward(t *testing.T) {
	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.Local)
	day := time.Date(2026, 8, 27, 0, 0, 0, 0, time.Local)

	m, ok := squirrel.MomentOn(day, "at 09:00 dentist", now)
	require.True(t, ok)
	require.Equal(t, 27, m.Starts.Day())
}

// And ParseMoment is unchanged: today, or tomorrow when the time has gone.
func TestParseMomentStillRollsForward(t *testing.T) {
	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.Local)

	m, ok := squirrel.ParseMoment("at 09:00 dentist", now)
	require.True(t, ok)
	require.Equal(t, 25, m.Starts.Day())
}

// The bar is unchanged too. A chosen day does not make a bare time a fixed
// point: the picker composes a sentence that clears the bar, and anything that
// does not clear it is still a note.
func TestAChosenDayDoesNotLowerTheBar(t *testing.T) {
	now := time.Now()
	_, ok := squirrel.MomentOn(now.AddDate(0, 0, 3), "14:30 dentist", now)
	require.False(t, ok)
}
```

- [ ] **Step 2: run — expect `undefined: squirrel.MomentOn`**
- [ ] **Step 3: implement.** Rename the body of `ParseMoment` to `MomentOn`, take `day`, and change the anchor:

```go
	starts := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, now.Location())
	// Tomorrow if it was said, and tomorrow if the time has already gone —
	// which is what someone means when they type a time that has passed.
	//
	// Only when the day is the one we are standing in. A day that was chosen
	// was chosen: rolling the 27th forward because 09:00 is behind us today
	// would book the 28th, which nobody asked for.
	sameDay := day.Year() == now.Year() && day.YearDay() == now.YearDay()
	if m[1] != "" || (sameDay && !starts.After(now)) {
		starts = starts.AddDate(0, 0, 1)
	}
```

and

```go
// ParseMoment reads a fixed point out of a sentence, today or tomorrow.
func ParseMoment(s string, now time.Time) (Moment, bool) { return MomentOn(now, s, now) }
```

- [ ] **Step 4: run, then mutate.** Each of: drop `sameDay` from the condition (the chosen-day test rolls forward); anchor `starts` to `now` instead of `day` (the day is ignored); make `MomentOn` accept a bare time (the bar test passes something it should not).
- [ ] **Step 5: commit** — `feat: the same sentence, on a day you chose`

---

### Task 2: the agenda door

**Files:** `internal/web/thread.go`, `internal/web/thread_test.go`

**Produces:** `agendaTurn(ctx, s, personID, name)`, wired into `placeTurn`'s `"at"` case.

Each fixed point is a card: the label as the title, `squirrel.LeaveWords(m)` as the meta — the core's own sentence, so the card, chat and the notification cannot drift about when to leave. One act, `OPEN`, posting to `/at/open` with the id; and `LEAVING` as well when the window is open, posting the same `/now/act` the offer's does.

Empty says *when something has a time you can be late for, it will be here* — the words the empty state used, unchanged.

**Never a count of what is past, and nothing marked late.** The guard rails that let this list exist at all are in `PRODUCT.md` and are not touched by moving it into a turn.

- [ ] Tests: the door draws what is coming; the meta is `LeaveWords`; `LEAVING` is absent outside the window and present inside it; empty says so; nothing past is drawn.
- [ ] Mutations: drop the `"at"` case; use the label instead of `LeaveWords`; render `LEAVING` unconditionally; delete the empty branch; widen `Upcoming`'s window.
- [ ] Commit — `feat: what has a time, as something buddy said`

---

### Task 3: one fixed point, as a turn

**Files:** `internal/web/at.go`, `internal/web/thread.go`

`/at/open` posts an id and answers with a turn: the fixed point's card, what to take on its own line, and the notes pointing at it as further cards, each with BACK IN THE PILE.

`/at/{id}` **stays a real page** until phase 4. A notification sent yesterday is still on a lock screen, and phase 4 is what retires the URL.

- [ ] Tests: opening one draws its notes; what to take is on its own line; a fixed point that is not yours draws nothing; detaching says so.
- [ ] Commit — `feat: a fixed point, and what is pointing at it`

---

### Task 4: the day and time picker

**Files:** `internal/web/thread.go`, `internal/web/at.go`, `internal/web/templates/turn.html`, `internal/web/static/pile.css`

**Produces:** `drawn` gains `Cal *calView`; `POST /at/new` asks, `POST /at/make` answers.

```go
type calView struct {
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
	Month  string            `json:"month"`   // "August 2026"
	Prev   string            `json:"prev"`    // the month before, as 2026-07
	Next   string            `json:"next"`
	Pad    int               `json:"pad"`     // blanks before the 1st, Monday-first
	Days   int               `json:"days"`
	Chosen int               `json:"chosen,omitempty"`
	Times  []string          `json:"times"`
	At     string            `json:"at,omitempty"`
}
```

**One form, one answer**, exactly as the interval picker: the day is a radio, the time is a radio, and one submit. Moving between months is the one thing that posts on its own — it is not an answer, it is turning a page, and it re-asks rather than writing a second question.

**The composed line goes through `MomentOn`.** `/at/make` builds `at 14:30 dentist` and hands it to the parser with the chosen day. No arithmetic in the web package.

**43px cells at 390px**, one pixel under the touch floor, accepted deliberately: seven columns cannot do better without cropping the month or scrolling it sideways, and both are worse.

- [ ] Tests: the question offers a month and times; a chosen day is marked; answering makes a fixed point on that day; a day nobody offered does nothing; the picker and a typed `at 14:30 dentist` agree about the time; moving a month re-asks without writing an appointment.
- [ ] Mutations: build the `time.Time` in the handler instead of calling `MomentOn`; ignore the chosen day; accept an unoffered time; make paging write an appointment.
- [ ] Commit — `feat: which day, and what time`

---

### Task 5: `/at` comes out, records amended

- [ ] Delete `atHandler` and `templates/at.html`; drop the route and the `at` page. Keep `/at/{id}`.
- [ ] Update the fences: the route table, the heading list, the appearance and contrast walks.
- [ ] `PRODUCT.md` — the fixed-point guard rails restated for a turn; the picker is a calendar the product refused, and what keeps it allowed is unchanged: only what is ahead, nothing past, nothing late.
- [ ] `DESIGN.md` — the agenda's section; the day picker's grid and its 43px cell.
- [ ] `docs/roadmap.md` — a v0.25.0 entry.
- [ ] Commit and open the PR, stacked on `feat/the-thread-2`.

## Deliberately not in 3a

The pile and the deck, search as a turn, `/kept`, `/held` and `/moods` as turns, and the notification's destination. 3b and phase 4.
