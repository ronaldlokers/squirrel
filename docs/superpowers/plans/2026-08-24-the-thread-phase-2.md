# The Thread, Phase 2 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Absorb the chores and the tasks into the conversation — a door press becomes something you said, what is behind it arrives as cards in Buddy's reply, and every action on them answers as a turn. The four fixed interval chips become a number × unit picker.

**Architecture:** The rail's doors stop being links and become one-button POST forms to `/open`, so opening a place is an utterance that goes into the record like everything else. `/chores/act`, `/chores/new`, `/tasks/act` and `/tasks/new` keep their URLs and their writes and answer through `answerWith` instead of redirecting to a screen. `/chores`, `/tasks` and `/tasks/done` are deleted.

**Tech Stack:** Go 1.26, pgx v5, `html/template`, vanilla ES. Tests: `go test`, plus tags `integration` (needs `TEST_DATABASE_URL`) and `browser` (needs chromium).

**Spec:** `docs/superpowers/specs/2026-08-24-the-thread-design.md` — phase 2 under *Staging*. Read it and phase 1's plan (`2026-08-24-the-thread-phase-1.md`) before starting; this builds directly on the machinery that landed there.

## Global Constraints

- **Counts are permitted.** Principle 2 was retired on 24 August 2026. Do not add a test asserting the absence of a number.
- **JavaScript is required**, and there is **one rendering path**: handlers return HTML from `internal/web/templates/*`, as a page or as the turns a press produced. No JSON, no client-side template.
- **History is never rewritten.** A stored turn holds text, never a foreign key it re-reads. A card in scrollback keeps its words and loses its buttons.
- **Only the newest Buddy turn carries controls**, and **Buddy does not talk over himself** — `endsOpen` in `internal/web/thread.go` is the existing guard, and every new turn-writing path has to respect it.
- **Zero renders as no number**, never as `0`.
- **A turn that opens a place carries an `<h2>`** with that place's name.
- **Mutation proof is required for every test.** Revert the behaviour, record the exact assertion-failure text you observed, restore. A compile error is **not** proof — the failure must be an assertion failing in code that compiles and runs. If a mutation does not compile (`declared and not used`), rewrite the mutation, not the test.
- **Work on `feat/the-thread-2`, branched from `feat/the-thread`.** Never commit to `main`.
- Time comes from `now()` in `internal/web`.

---

## What phase 1 left for this one to use

Read these before writing anything; the whole plan is built on them.

| thing | where | what it does |
| --- | --- | --- |
| `drawn` | `internal/web/thread.go` | what a turn drew, as JSON: `place`, `cards`, `chips`, `faces` |
| `cardView`, `actView`, `turnChip` | same | a card, a button (with a map of hidden fields), a link |
| `keepSaid` | same | writes turns, returns the ones that were written |
| `answerWith` | same | redirect, or the new turns as HTML when the press carried `X-Thread: fragment` |
| `endsOpen` | same | is something Buddy put on the table still unanswered |
| `railFor` | same | the four doors and their numbers |
| `with(fields, k, v)` | same | a row's fields plus one, copied |

---

## File Structure

**Modified**

| file | change |
| --- | --- |
| `internal/web/thread.go` | `/open`, the chore and task turns, `cardView` gains `Photo` |
| `internal/web/templates/thread.html` | the rail's doors become forms |
| `internal/web/templates/turn.html` | a card may carry a photograph and a picker |
| `internal/web/chores.go` | the three actions answer as turns; `choresHandler` deleted |
| `internal/web/tasks.go` | the same for tasks; `tasksHandler` and `archiveHandler` deleted |
| `internal/web/pile.go` | `/open` added; `/chores`, `/tasks`, `/tasks/done` removed |
| `internal/web/render.go` | `pages` loses `chores`, `tasks`, `archive` |
| `internal/web/static/pile.css` | the picker's two rows, the card's photograph |
| `PRODUCT.md`, `DESIGN.md`, `docs/roadmap.md` | the interval vocabulary and the two screens' removal |

**Deleted**

`internal/web/templates/chores.html`, `internal/web/templates/tasks.html`, `internal/web/templates/archive.html`, and `internal/web/templates/every.html` once nothing includes it.

---

### Task 1: Opening a door is something you said

**Files:**
- Modify: `internal/web/thread.go`, `internal/web/templates/thread.html`, `internal/web/pile.go`
- Test: `internal/web/thread_test.go`

**Interfaces:**
- Consumes: `keepSaid`, `answerWith`, `endsOpen`, `drawn`, `cardView`, `railFor`.
- Produces:
  ```go
  func openHandler(s Store, opts Options) http.HandlerFunc   // POST /open, field `where`
  func placeTurn(ctx context.Context, s Store, opts Options, personID int64, where string) []squirrel.Turn
  ```
  `where` is one of `pile`, `tasks`, `chores`, `at`. Anything else writes nothing.

**Why a POST and not a link.** Opening a place is an utterance — the mockup drew it as your own bubble — so it belongs in the record, and a GET that writes to the record would write again on every reload and on every walk back through the past. What it costs, stated rather than discovered: **a door can no longer be opened in a new tab, and the back button does not step through doors.** In a single-page app that is the ordinary trade; it is worth writing down because it is the one thing the rail lost.

- [ ] **Step 1: Write the failing tests**

Add to `internal/web/thread_test.go`:

```go
// Pressing a door says its name, and Buddy answers with what is behind it.
func TestOpeningADoorSaysItsName(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "water the plants", EveryDays: 7, SinceDays: 8, Active: true, EverDone: true},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=chores"))

	require.Len(t, f.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, f.appended[0].Who)
	require.Equal(t, "the chores", f.appended[0].Words)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[1].Who)
	require.Contains(t, string(f.appended[1].Shown), "water the plants")
}

// And the turn carries the place's name as a heading, so heading navigation
// still walks the app.
func TestTheReplyToADoorCarriesItsHeading(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=chores"))

	require.Contains(t, string(f.appended[1].Shown), `"place":"the chores"`)
}

// A door nobody offered does nothing and says nothing. It arrives from a form,
// so it is read the way a stranger's typing is read.
func TestADoorThatDoesNotExistDoesNothing(t *testing.T) {
	f := &fakeStore{}
	w := routed(t, f).call(t, "POST", "/open", strings.NewReader("where=cellar"))

	require.Equal(t, 303, w.Code)
	require.Empty(t, f.appended)
}

// Nothing behind it is a sentence, not an empty card list — the same rule the
// empty states followed.
func TestAnEmptyPlaceSaysSoAndDrawsNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=chores"))

	require.Len(t, f.appended, 2)
	require.NotContains(t, string(f.appended[1].Shown), `"cards"`)
	require.NotEmpty(t, f.appended[1].Words)
}

// The rail posts rather than links, because opening a place is something you
// said and a GET that writes would write again on every reload.
func TestTheRailPostsRatherThanLinks(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Contains(t, body, `action="/open"`)
	require.NotContains(t, body, `<a class="rdoor`)
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
go test ./internal/web/ -run 'TestOpeningADoor|TestTheReplyToADoor|TestADoorThatDoesNot|TestAnEmptyPlace|TestTheRailPosts' -v
```

Expected: `404` from the recorder, so `require.Len(t, f.appended, 2)` fails with `expected 2, actual 0`; the rail test fails on `does not contain action="/open"`.

- [ ] **Step 3: Write the handler**

Add to `internal/web/thread.go`:

```go
// openHandler is a door being pressed.
//
// A POST, and not a link, because opening a place is an utterance: it goes into
// the record like anything else you say. A GET that wrote to the record would
// write again on every reload and on every walk back through the past.
//
// What it costs, stated rather than discovered: a door cannot be opened in a
// new tab, and the back button does not step through doors. That is the
// ordinary trade for one page, and it is the only thing the rail gave up.
func openHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		said := placeTurn(r.Context(), s, opts, personID, r.FormValue("where"))
		if len(said) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, said), "/")
	}
}

// doorNames is the vocabulary, as a map rather than a switch so an unknown
// door is a lookup miss instead of a default branch someone later fills in
// with something destructive. The same device the offer's kinds use.
var doorNames = map[string]string{
	"pile": "the pile", "tasks": "the tasks", "chores": "the chores", "at": "the agenda",
}

// placeTurn is what you said and what Buddy answered, or nothing at all.
func placeTurn(ctx context.Context, s Store, opts Options, personID int64, where string) []squirrel.Turn {
	name, ok := doorNames[where]
	if !ok {
		return nil
	}
	var reply squirrel.Turn
	switch where {
	case "chores":
		reply = choresTurn(ctx, s, personID, name)
	default:
		// The pile and the agenda are phase 3. Until then the doors that are
		// not built say so rather than answering with silence, which reads as
		// a press that did not land.
		reply = squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Not yet. That one is still a page."}
	}
	return []squirrel.Turn{{Who: squirrel.SpeakerYou, Words: name}, reply}
}
```

- [ ] **Step 4: Write the chores turn**

Still in `internal/web/thread.go`:

```go
// listLimit is how many cards one turn draws.
//
// A bound rather than a page, and it matters more here than anywhere else: a
// turn is frozen the moment it is written, so a turn holding forty cards is
// forty cards in the record forever.
const listLimit = 12

// choresTurn is what comes back, as cards.
func choresTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	chores, err := s.ActiveChores(ctx, personID)
	if err != nil {
		slog.Error("reading what comes back", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the chores just now."}
	}
	if len(chores) == 0 {
		// A fact, not a nudge. It says where chores come from and offers the
		// other way of making one — the same words the empty state used.
		body, err := json.Marshal(drawn{Place: name, Chips: []turnChip{
			{Label: "a new chore", Href: "/?new=chore"},
		}})
		if err != nil {
			slog.Error("drawing the chores", "error", err)
			return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Nothing comes back on its own."}
		}
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing comes back on its own. When a note becomes a chore, it lives here.",
			Shown: body,
		}
	}

	more := false
	if len(chores) > listLimit {
		chores, more = chores[:listLimit], true
	}
	sh := drawn{Place: name}
	for _, c := range chores {
		v := toChoreView(c)
		card := cardView{Title: v.Name, Meta: choreMeta(v)}
		row := map[string]string{"id": strconv.FormatInt(v.ID, 10), "label": v.Name}
		card.Acts = []actView{
			{Label: "DID IT", Action: "/chores/act", Style: "did", Fields: with(row, "act", "done")},
			{Label: "HOW OFTEN", Action: "/chores/often", Style: "go", Fields: row},
			{Label: "STOP ASKING", Action: "/chores/act", Style: "stop", Fields: with(row, "act", "retire")},
		}
		sh.Cards = append(sh.Cards, card)
	}
	sh.Chips = []turnChip{{Label: "a new chore", Href: "/?new=chore"}}
	if more {
		sh.Chips = append(sh.Chips, turnChip{Label: "the rest", Href: "/?open=chores&after=" + strconv.Itoa(listLimit)})
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the chores", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the chores just now."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: choreLead(len(sh.Cards)), Shown: body}
}

// choreMeta is the rhythm, and what has happened, in the card's own line.
// What has not happened is not reported: a chore nobody has ever done shows its
// rhythm and stops there.
func choreMeta(v choreView) string {
	out := v.Every
	if v.Last != "" {
		out += " · last done " + v.Last
	}
	if v.When != "" {
		out += " · " + v.When
	}
	return out
}

// choreLead is Buddy counting, which he is allowed to do since 24 August 2026 —
// Principle 5 permitted it in speech on 20 August and Principle 2's retirement
// permitted it everywhere else.
func choreLead(n int) string {
	if n == 1 {
		return "One comes back."
	}
	return fmt.Sprintf("%d come back.", n)
}
```

Add `"fmt"` to the imports.

- [ ] **Step 5: The rail posts**

In `internal/web/templates/thread.html`, replace the door anchor with a form:

```html
    {{range .Rail}}
    {{/* A form rather than a link: opening a place is something you said, and
         it goes into the record. See openHandler for what that costs. */}}
    <form class="rdoorform" method="post" action="/open">
      <input type="hidden" name="where" value="{{.Where}}">
      <button class="rdoor{{if .Here}} here{{end}}" type="submit">
        <img alt="" src="/static/{{.Art}}?v={{$.V}}">
        <span class="rname">{{.Label}}</span>
        {{if .Count}}<span class="doorcount">{{.Count}}</span>{{end}}
      </button>
    </form>
    {{end}}
```

`doorView` gains `Where string` and loses nothing — keep `Href` off it entirely, so nothing can render a door as a link by accident:

```go
type doorView struct {
	// Where is the door's own word, posted to /open. Not an href: a door is
	// pressed rather than followed, and a field that could be used as one
	// would invite exactly that.
	Where string
	Label string
	Art   string
	Count int
	Here  bool
}
```

Update `railFor` to set `Where: "pile"` and so on.

Give the form no box of its own in `internal/web/static/pile.css`:

```css
  .rdoorform { display: contents; }
```

- [ ] **Step 6: Route it**

In `internal/web/pile.go`:

```go
	// A door being pressed. See openHandler for why it is a POST.
	m.Post("/open", guard(opts, sameOrigin(openHandler(s, opts))))
```

- [ ] **Step 7: Run the tests**

```bash
go test ./internal/web/ -run 'TestOpeningADoor|TestTheReplyToADoor|TestADoorThatDoesNot|TestAnEmptyPlace|TestTheRailPosts' -v
go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 8: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestOpeningADoorSaysItsName` | return only Buddy's turn from `placeTurn` | `expected 2, actual 1` |
| `TestTheReplyToADoorCarriesItsHeading` | drop `Place: name` from the `drawn` literal | does not contain `"place":"the chores"` |
| `TestADoorThatDoesNotExistDoesNothing` | make `doorNames` lookup default to `"the chores"` | `Empty` fails with two turns |
| `TestAnEmptyPlaceSaysSoAndDrawsNothing` | remove the `len(chores) == 0` branch | the marshalled `drawn` contains `"cards"` |
| `TestTheRailPostsRatherThanLinks` | put the anchor back | does not contain `action="/open"` |

Record the failure texts in the commit body.

- [ ] **Step 9: Commit**

```bash
git add internal/web/thread.go internal/web/thread_test.go \
        internal/web/templates/thread.html internal/web/pile.go internal/web/static/pile.css
git commit -m "feat: a door is something you say"
```

---

### Task 2: The chores answer as turns

**Files:**
- Modify: `internal/web/chores.go`, `internal/web/thread.go`
- Test: `internal/web/chores_test.go`

**Interfaces:**
- Consumes: `keepSaid`, `answerWith`, `saidAboutTheOffer`'s shape (do not reuse it — chores have their own words), `(Store).RecordCompletion`, `(Store).DeactivateChore`, `(Store).UpsertChoreAsking`.
- Produces: `func saidAboutAChore(act, label string) []squirrel.Turn`.

- [ ] **Step 1: Write the failing tests**

```go
// Doing a chore says so, and the saying is in the record beside the doing.
func TestDoingAChoreIsSaid(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&act=done&label=water+the+plants"))

	require.Equal(t, []int64{1}, f.completed)
	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[0].Words, "water the plants")
}

// Stopping one is not the same sentence as doing it — which answer you gave is
// the whole of what happened.
func TestRetiringAChoreSaysSomethingElse(t *testing.T) {
	did, stopped := &fakeStore{}, &fakeStore{}
	routed(t, did).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=done&label=bins"))
	routed(t, stopped).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=retire&label=bins"))

	require.Equal(t, []int64{1}, stopped.retired)
	require.NotEqual(t, did.appended[1].Words, stopped.appended[1].Words)
}

// A write that failed must not be reported as one that happened.
func TestAChoreThatCannotBeWrittenIsNotSaid(t *testing.T) {
	f := &fakeStore{err: errTest}
	routed(t, f).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=done&label=bins"))

	require.Empty(t, f.appended)
}

// An act nobody offered does nothing and says nothing.
func TestAChoreActThatWasNeverOfferedDoesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=burn&label=bins"))

	require.Empty(t, f.appended)
	require.Empty(t, f.completed)
	require.Empty(t, f.retired)
}

// A new chore arrives as a turn with the chore in it, so the thing you just
// made is on the screen rather than somewhere you have to go and look.
func TestANewChoreComesBackAsACard(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/new",
		strings.NewReader("name=descale+the+kettle&every=every+2+weeks"))

	require.Equal(t, "descale the kettle", f.reinterval.name)
	require.Len(t, f.appended, 2)
	require.Contains(t, string(f.appended[1].Shown), "descale the kettle")
}
```

Extend `fakeStore` only if a field is genuinely missing; `completed`, `retired` and `reinterval` already exist.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run 'TestDoingAChore|TestRetiringAChore|TestAChoreThat|TestANewChore' -v
```

Expected: `expected 2, actual 0` on the turn assertions — the writes already work, the saying does not.

- [ ] **Step 3: Write the words**

Add to `internal/web/thread.go`:

```go
// saidAboutAChore is what the two of you said about one.
//
// Its own function rather than the offer's, because the answers are different
// facts: an offer is a thing you were handed, and a chore is a thing that comes
// back whatever you do about it. "Stop asking" especially — it is the one press
// here that ends something, and it must not read like finishing it.
func saidAboutAChore(act, label string) []squirrel.Turn {
	if label == "" {
		label = "that"
	}
	switch act {
	case "done":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "Good. It will come back."},
		}
	case "retire":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "stop asking — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "It will not come back. Tell me if you want it again."},
		}
	}
	return nil
}
```

- [ ] **Step 4: Answer with them**

In `internal/web/chores.go`, in `choreActHandler`, after the write succeeds and before the redirect:

```go
		answerWith(w, r, keepSaid(r.Context(), s, personID,
			saidAboutAChore(r.FormValue("act"), r.FormValue("label"))), "/")
		return
```

and in `newChoreHandler`, after the chore is created:

```go
		// The chore you just made, as a card, so it is on the screen rather
		// than somewhere you have to go and look at.
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: name + " — " + every},
			madeAChore(c),
		}), "/")
		return
```

with:

```go
// madeAChore is the new chore, drawn the way the list draws one.
//
// The same cardView the list builds, so a chore made from nothing and a chore
// read back out of the store cannot look different — which is the sort of
// difference nobody notices until one of them grows a button the other has not.
func madeAChore(c squirrel.Chore) squirrel.Turn {
	v := toChoreView(c)
	row := map[string]string{"id": strconv.FormatInt(v.ID, 10), "label": v.Name}
	body, err := json.Marshal(drawn{Cards: []cardView{{
		Title: v.Name, Meta: choreMeta(v),
		Acts: []actView{
			{Label: "DID IT", Action: "/chores/act", Style: "did", Fields: with(row, "act", "done")},
			{Label: "HOW OFTEN", Action: "/chores/often", Style: "go", Fields: row},
			{Label: "STOP ASKING", Action: "/chores/act", Style: "stop", Fields: with(row, "act", "retire")},
		},
	}}})
	if err != nil {
		slog.Error("drawing a new chore", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Kept. It comes back " + v.Every + "."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Kept.", Shown: body}
}
```

Both handlers must keep their existing guards: an act that is not in the vocabulary, and a name that is empty, still do nothing.

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/web/ -run 'TestDoingAChore|TestRetiringAChore|TestAChoreThat|TestANewChore' -v
```

- [ ] **Step 6: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestDoingAChoreIsSaid` | drop the `keepSaid` call | `expected 2, actual 0` |
| `TestRetiringAChoreSaysSomethingElse` | make `retire` return the `done` sentences | `Should not be: "Good. It will come back."` |
| `TestAChoreThatCannotBeWrittenIsNotSaid` | move `keepSaid` above the write's error check | `Empty` fails with two turns |
| `TestAChoreActThatWasNeverOfferedDoesNothing` | remove the act guard | `Empty` fails |
| `TestANewChoreComesBackAsACard` | return a wordless turn instead of `madeAChore` | does not contain `descale the kettle` |

- [ ] **Step 7: Commit**

```bash
git add internal/web/chores.go internal/web/chores_test.go internal/web/thread.go
git commit -m "feat: the chores answer back"
```

---

### Task 3: How often — number × unit

**Files:**
- Modify: `internal/web/thread.go`, `internal/web/chores.go`, `internal/web/templates/turn.html`, `internal/web/pile.go`, `internal/web/static/pile.css`
- Test: `internal/web/chores_test.go`

**Interfaces:**
- Consumes: `squirrel.ParseEvery` (`internal/squirrel/intent.go`) — it already accepts any count against any unit, and `unitDurations` already holds day, week, fortnight, month, quarter and year. **No core work.**
- Produces:
  ```go
  func oftenHandler(s Store, opts Options) http.HandlerFunc  // POST /chores/often
  func askHowOften(id int64, label, current string) squirrel.Turn
  ```
  and `drawn` gains `Pick *pickView`:
  ```go
  type pickView struct {
      Action string            `json:"action"`
      Fields map[string]string `json:"fields,omitempty"`
      Rows   []pickRow         `json:"rows"`
      Do     string            `json:"do"`
  }
  type pickRow struct {
      Lead    string   `json:"lead"`
      Name    string   `json:"name"`
      Options []string `json:"options"`
      Chosen  string   `json:"chosen,omitempty"`
  }
  ```

**One form, one answer.** The two rows are radio groups inside a single form with one submit. Pressing a number must not write a turn of its own: a picker that wrote a turn per fiddle would fill a record that is never rewritten with the sound of somebody deciding. You are asked once and you answer once.

**The picker composes a sentence.** It posts `every 2 weeks` — the exact string the four fixed chips posted — so the chore path underneath is untouched and the sentence lane and the picker cannot disagree, because they produce the same string for the same rhythm.

**No `…` and no keypad.** Six numbers cover what anyone reaches for; `every 9 weeks` is a sentence you type.

- [ ] **Step 1: Write the failing tests**

```go
// Asking how often puts the question on the table with both rows on it.
func TestAskingHowOftenOffersNumbersAndUnits(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/often",
		strings.NewReader("id=1&label=water+the+plants"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	for _, want := range []string{`"1"`, `"2"`, `"3"`, `"4"`, "days", "weeks", "months"} {
		require.Contains(t, shown, want)
	}
}

// The rhythm it has now is marked, so the question is answerable rather than a
// blank form: you are changing something, not inventing it.
func TestTheQuestionMarksTheRhythmItHasNow(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "water the plants", Every: 14 * 24 * time.Hour, EveryDays: 14, Active: true},
	}}
	routed(t, f).call(t, "POST", "/chores/often", strings.NewReader("id=1&label=water+the+plants"))

	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, `"chosen":"2"`)
	require.Contains(t, shown, `"chosen":"weeks"`)
}

// Answering composes the same sentence the fixed chips used to post, so the
// chore path underneath is untouched.
func TestAnsweringHowOftenComposesTheSentence(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&label=water+the+plants&count=3&unit=weeks"))

	require.Equal(t, 21*24*time.Hour, f.reinterval.every)
}

// And says it back in the same words a person would use.
func TestAnsweringHowOftenIsSaid(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&label=water+the+plants&count=3&unit=weeks"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[0].Words, "every 3 weeks")
}

// A number and a unit nobody offered do nothing. They arrive from a form.
func TestARhythmThatWasNeverOfferedDoesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&label=bins&count=99&unit=fortnights"))

	require.Empty(t, f.appended)
	require.Zero(t, f.reinterval.every)
}

// The picker and a typed sentence produce the same interval for the same
// rhythm. Asserted on the duration, not on the string.
func TestThePickerAndTheSentenceAgree(t *testing.T) {
	_, typed, ok := squirrel.ParseEvery("every 3 months: water the plants")
	require.True(t, ok)

	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&label=water+the+plants&count=3&unit=months"))

	require.Equal(t, typed, f.reinterval.every)
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run 'TestAskingHowOften|TestTheQuestionMarks|TestAnsweringHowOften|TestARhythmThat|TestThePickerAndTheSentence' -v
```

Expected: 404 on `/chores/often`, and `f.reinterval.every` zero.

- [ ] **Step 3: Write the question**

Add to `internal/web/thread.go`:

```go
// pickNumbers and pickUnits are what the picker offers.
//
// Six numbers and three units, and no way to type one: six covers what anyone
// reaches for, and `every 9 weeks` is a sentence rather than a control. The
// units are the three a chore is actually said in — ParseEvery accepts
// fortnights, quarters and years too, and those stay available through the
// sentence at no cost in buttons.
var (
	pickNumbers = []string{"1", "2", "3", "4", "6", "8"}
	pickUnits   = []string{"days", "weeks", "months"}
)

// askHowOften is the question, as one form with two rows.
//
// One form and one submit, deliberately: a picker that wrote a turn every time
// you pressed a number would fill a record that is never rewritten with the
// sound of somebody deciding. You are asked once and you answer once.
func askHowOften(id int64, label, count, unit string) squirrel.Turn {
	body, err := json.Marshal(drawn{Pick: &pickView{
		Action: "/chores/act",
		Fields: map[string]string{"id": strconv.FormatInt(id, 10), "label": label},
		Do:     "that's it",
		Rows: []pickRow{
			{Lead: "every", Name: "count", Options: pickNumbers, Chosen: count},
			{Lead: "of these", Name: "unit", Options: pickUnits, Chosen: unit},
		},
	}})
	if err != nil {
		slog.Error("drawing the question", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Tell me how often, in words."}
	}
	return squirrel.Turn{
		Who: squirrel.SpeakerBuddy, Words: "How often should it come back?", Shown: body,
	}
}

// rhythmOf is the interval a chore has now, as the picker's own two answers,
// so the question opens on what is true rather than on a blank form.
//
// Anything that does not land on one of the offered pairs — a fortnight typed
// as a sentence, a quarter — leaves both unmarked rather than rounding to the
// nearest offered thing. Marking the wrong one is worse than marking none: it
// would tell you the chore is something it is not.
func rhythmOf(every time.Duration) (count, unit string) {
	for _, u := range pickUnits {
		step := unitStep(u)
		if step == 0 || every%step != 0 {
			continue
		}
		n := strconv.FormatInt(int64(every/step), 10)
		for _, offered := range pickNumbers {
			if offered == n {
				return n, u
			}
		}
	}
	return "", ""
}

func unitStep(unit string) time.Duration {
	switch unit {
	case "days":
		return 24 * time.Hour
	case "weeks":
		return 7 * 24 * time.Hour
	case "months":
		// Thirty days, exactly as the core reads it. This is a nudge, not a
		// calendar — see unitDurations in internal/squirrel/intent.go, and do
		// not let the two drift.
		return 30 * 24 * time.Hour
	}
	return 0
}
```

**Note for the implementer:** `rhythmOf` tries days first, so 14 days matches `2 weeks` only if `days` fails — which it does not, since 14 is not in `pickNumbers`. Check that with the test above before moving on; if `pickNumbers` ever gains `14`, the order stops being safe and the loop needs the largest unit first instead.

- [ ] **Step 4: Write the handler**

Add to `internal/web/chores.go`:

```go
// oftenHandler puts the question on the table.
//
// It writes rather than renders, like everything else here: a question that is
// not in the record is a question the record cannot show you answering.
func oftenHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		label := r.FormValue("label")

		// What it is now, so the question opens on what is true. A chore that
		// cannot be read is still a question worth asking — it opens unmarked.
		var count, unit string
		if chores, err := s.ActiveChores(r.Context(), personID); err == nil {
			for _, c := range chores {
				if c.ID == id {
					count, unit = rhythmOf(c.Every)
					break
				}
			}
		}

		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "how often — " + label},
			askHowOften(id, label, count, unit),
		}), "/")
	}
}
```

In `choreActHandler`, before the `act` switch, take the picker's answer:

```go
		// The picker's answer: a number and a unit, composed into the same
		// sentence the four fixed chips used to post. The two lanes cannot
		// disagree because they produce the same string.
		if count, unit := r.FormValue("count"), r.FormValue("unit"); count != "" || unit != "" {
			every, ok := composeEvery(count, unit)
			if !ok {
				// Neither was offered. Nothing is done and nothing is said.
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			name := r.FormValue("label")
			if _, err := s.UpsertChore(r.Context(), personID, name, every, every/2); err != nil {
				fail(w, err)
				return
			}
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: "every " + count + " " + unit},
				{Who: squirrel.SpeakerBuddy, Words: name + " comes back every " + count + " " + unit + " now."},
			}), "/")
			return
		}
```

and:

```go
// composeEvery turns the picker's two answers into an interval, through the
// same parser a typed sentence goes through.
//
// Not arithmetic here: ParseEvery is where "every 3 weeks" means something, and
// a second place that decided what a week was would be a second place to be
// wrong. Both are checked against what was offered first, because they arrive
// from a form.
func composeEvery(count, unit string) (time.Duration, bool) {
	if !offered(pickNumbers, count) || !offered(pickUnits, unit) {
		return 0, false
	}
	_, every, ok := squirrel.ParseEvery("every " + count + " " + unit + ": x")
	return every, ok
}

func offered(list []string, v string) bool {
	for _, o := range list {
		if o == v {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Draw it**

`drawn` gains `Pick *pickView`, `turnView` gains `Pick *pickView`, and `turnViews` copies it across. In `internal/web/templates/turn.html`, after the cards:

```html
  {{/* The question, as one form with two rows and one submit. Radios rather
       than buttons: pressing a number is not an answer, it is half of one, and
       a picker that posted on every press would write a turn per fiddle into a
       record that is never rewritten. */}}
  {{if and .Live .Pick}}
  <form class="pick" method="post" action="{{.Pick.Action}}">
    {{range $name, $value := .Pick.Fields}}<input type="hidden" name="{{$name}}" value="{{$value}}">{{end}}
    {{range .Pick.Rows}}
    <p class="picklead">{{.Lead}}</p>
    <div class="pickrow">
      {{$row := .}}
      {{range .Options}}
      <label class="chip{{if eq . $row.Chosen}} current{{end}}">
        <input type="radio" name="{{$row.Name}}" value="{{.}}"{{if eq . $row.Chosen}} checked{{end}}>{{.}}
      </label>
      {{end}}
    </div>
    {{end}}
    <button class="make" type="submit">{{.Pick.Do}}</button>
  </form>
  {{end}}
```

Style it in `internal/web/static/pile.css`, reusing the chips the new-chore form already uses:

```css
  .pick { display: flex; flex-direction: column; gap: 8px; width: 100%;
    padding: 13px 15px 15px; background: var(--card); color: var(--outline);
    border: var(--line) solid var(--outline); border-radius: var(--r);
    box-shadow: 0 5px 0 0 var(--outline), 0 16px 26px -16px rgba(0, 0, 0, .7); }
  .picklead { margin: 0; font-size: 11.5px; letter-spacing: .1em;
    text-transform: uppercase; color: var(--brown);
    font-variation-settings: var(--precise), "wght" 750; }
  .pickrow { display: flex; flex-wrap: wrap; gap: 8px; }
  .pick .make { align-self: flex-start; margin-top: 4px; }
```

Route it in `internal/web/pile.go`:

```go
	m.Post("/chores/often", guard(opts, sameOrigin(oftenHandler(s, opts))))
```

- [ ] **Step 6: Run the tests**

```bash
go test ./internal/web/ -run 'TestAskingHowOften|TestTheQuestionMarks|TestAnsweringHowOften|TestARhythmThat|TestThePickerAndTheSentence' -v
```

- [ ] **Step 7: Prove them by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestAskingHowOftenOffersNumbersAndUnits` | drop the units row from `askHowOften` | does not contain `weeks` |
| `TestTheQuestionMarksTheRhythmItHasNow` | make `rhythmOf` always return `"", ""` | does not contain `"chosen":"2"` |
| `TestAnsweringHowOftenComposesTheSentence` | multiply the count by the unit in Go instead of calling `ParseEvery` **and use 24×7×count for months** | the months test fails on the duration |
| `TestARhythmThatWasNeverOfferedDoesNothing` | remove the `offered` checks from `composeEvery` | `Zero` fails |
| `TestThePickerAndTheSentenceAgree` | make `unitStep("months")` 31 days | the durations differ |

The third one is the one that matters: it is what stops a later author replacing `ParseEvery` with arithmetic and quietly redefining a month.

- [ ] **Step 8: Commit**

```bash
git add internal/web/thread.go internal/web/chores.go internal/web/chores_test.go \
        internal/web/templates/turn.html internal/web/pile.go internal/web/static/pile.css
git commit -m "feat: how often, as a number and a unit"
```

---

### Task 4: The tasks answer as turns

**Files:**
- Modify: `internal/web/thread.go`, `internal/web/tasks.go`
- Test: `internal/web/tasks_test.go`

**Interfaces:**
- Consumes: `(Store).Tasks`, `(Store).SetItemState`, `(Store).SetItemKind`, `toView` (`internal/web/render.go`), `cardView`, `keepSaid`, `answerWith`.
- Produces: `func tasksTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn`, `func saidAboutATask(act, label string) []squirrel.Turn`, and `cardView` gains `Photo string`.

**A task card carries its photograph.** A note with no words at all is a perfectly good note, and a task made from one would otherwise be a card saying nothing. `cardView` gains `Photo`, and `turn.html` draws it above the words the way the pile's card does.

- [ ] **Step 1: Write the failing tests**

```go
// The tasks arrive as cards, with what you decided in them.
func TestOpeningTheTasksDrawsThem(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 3, RawText: "ring the bank", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks"))

	require.Len(t, f.appended, 2)
	require.Contains(t, string(f.appended[1].Shown), "ring the bank")
	require.Contains(t, string(f.appended[1].Shown), `"place":"the tasks"`)
}

// A task made from a photograph is a card with the photograph on it. Without
// this it is a card saying nothing, which is what a note with no words is.
func TestATaskWithNoWordsKeepsItsPhotograph(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 3, RawText: "", Kind: squirrel.ItemTask, State: squirrel.ItemOpen,
			PhotoPath: squirrel.Ptr("letter.jpg")},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks"))

	require.Contains(t, string(f.appended[1].Shown), "photo")
}

// Doing one says so.
func TestDoingATaskIsSaid(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 3, RawText: "ring the bank", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	routed(t, f).call(t, "POST", "/tasks/act",
		strings.NewReader("id=3&act=done&label=ring+the+bank"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[0].Words, "ring the bank")
}

// And putting one back is a different sentence: it is not finishing it.
func TestUntaskingSaysSomethingElse(t *testing.T) {
	did, back := &fakeStore{}, &fakeStore{}
	routed(t, did).call(t, "POST", "/tasks/act", strings.NewReader("id=3&act=done&label=x"))
	routed(t, back).call(t, "POST", "/tasks/act", strings.NewReader("id=3&act=untask&label=x"))

	require.NotEqual(t, did.appended[1].Words, back.appended[1].Words)
}

// Nothing decided yet is a sentence rather than an empty list.
func TestNoTasksSaysSo(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks"))

	require.NotContains(t, string(f.appended[1].Shown), `"cards"`)
	require.NotEmpty(t, f.appended[1].Words)
}
```

Use the real field name for a note's photograph — check `squirrel.Item` before writing `PhotoPath`, and use whatever `toView` reads.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/web/ -run 'TestOpeningTheTasks|TestATaskWithNoWords|TestDoingATask|TestUntasking|TestNoTasksSaysSo' -v
```

- [ ] **Step 3: Write it**

Add to `internal/web/thread.go`:

```go
// tasksTurn is what you decided and have not done, as cards.
//
// Newest first, like the pile: a task decided this morning is the one you still
// remember deciding.
func tasksTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	items, more, err := s.Tasks(ctx, personID, listLimit)
	if err != nil {
		slog.Error("reading what you decided", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the tasks just now."}
	}
	if len(items) == 0 {
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing decided yet. A task is a note you said yes to.",
		}
	}

	sh := drawn{Place: name}
	for _, it := range items {
		v := toView(it)
		row := map[string]string{"id": strconv.FormatInt(v.ID, 10), "label": v.Text}
		sh.Cards = append(sh.Cards, cardView{
			Title: v.Text, Meta: "decided " + v.When, Photo: v.Photo,
			Acts: []actView{
				{Label: "did it", Action: "/tasks/act", Style: "did", Fields: with(row, "act", "done")},
				{Label: "not a task", Action: "/tasks/act", Style: "later", Fields: with(row, "act", "untask")},
			},
		})
	}
	if more {
		sh.Chips = []turnChip{{Label: "the rest", Href: "/?open=tasks&after=" + strconv.FormatInt(items[len(items)-1].ID, 10)}}
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the tasks", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the tasks just now."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: taskLead(len(sh.Cards)), Shown: body}
}

func taskLead(n int) string {
	if n == 1 {
		return "One thing you decided."
	}
	return fmt.Sprintf("%d things you decided.", n)
}

// saidAboutATask is what the two of you said about one.
//
// "Not a task" is not a failure and must not read like one: it is a note that
// went back to being a note, which is a decision reversed rather than a thing
// given up on.
func saidAboutATask(act, label string) []squirrel.Turn {
	if label == "" {
		label = "that"
	}
	switch act {
	case "done":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "Done."},
		}
	case "untask":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "not a task — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "Back in the pile."},
		}
	}
	return nil
}
```

Add `Photo string \`json:"photo,omitempty"\`` to `cardView`, and in `turn.html`, inside the card and above the title:

```html
    {{if .Photo}}<p class="cardphoto"><img src="{{.Photo}}" alt="the photograph on this note" loading="lazy"></p>{{end}}
```

Wire `placeTurn`'s switch to call `tasksTurn` for `"tasks"`, and `taskActHandler` to answer with `saidAboutATask` after its write succeeds.

- [ ] **Step 4: Run, then prove by mutation**

| test | mutation | expected failure |
| --- | --- | --- |
| `TestOpeningTheTasksDrawsThem` | leave `"tasks"` on the default branch of `placeTurn` | does not contain `ring the bank` |
| `TestATaskWithNoWordsKeepsItsPhotograph` | drop `Photo: v.Photo` | does not contain `photo` |
| `TestDoingATaskIsSaid` | drop the `keepSaid` call | `expected 2, actual 0` |
| `TestUntaskingSaysSomethingElse` | return the `done` sentences for `untask` | `Should not be: "Done."` |
| `TestNoTasksSaysSo` | remove the empty branch | the marshalled `drawn` contains `"cards"` |

- [ ] **Step 5: Commit**

```bash
git add internal/web/thread.go internal/web/tasks.go internal/web/tasks_test.go \
        internal/web/templates/turn.html
git commit -m "feat: what you decided, as something buddy said"
```

---

### Task 5: The two screens come out

**Files:**
- Delete: `internal/web/templates/chores.html`, `internal/web/templates/tasks.html`, `internal/web/templates/archive.html`, `internal/web/templates/every.html`
- Modify: `internal/web/chores.go`, `internal/web/tasks.go`, `internal/web/pile.go`, `internal/web/render.go`, `internal/web/heading_test.go`, `internal/web/appearance_test.go`, `internal/web/contrast_test.go`, `PRODUCT.md`, `DESIGN.md`, `docs/roadmap.md`

- [ ] **Step 1: Remove the routes and the handlers**

Drop `/chores`, `/tasks`, `/tasks/done` and `/pile/chores` from `internal/web/pile.go`, and `choresHandler`, `tasksHandler` and `archiveHandler` with them. Keep every write handler.

`every.html` goes only once nothing includes it — the pile's card and the search results both do today. **If they still do, keep it and say so in the commit message**: the pile is phase 3, and deleting a template two live screens include is not this task's business.

- [ ] **Step 2: Update the fences**

- `heading_test.go`: remove the deleted templates from `notATitle` if they are named there.
- `appearance_test.go`: delete the `/chores`, `/tasks` and `/tasks/done` entries; add the picker's selectors to `/` — `.pick`, `.picklead`, `.pickrow`, `.pick .make`.
- `contrast_test.go`: the same — the screen list loses two and the thread gains the states that now carry cards.

**Do not delete a screen from a fence to make it pass.** If a walk covers something no longer reachable, the entry moves to the state of the thread that shows it.

- [ ] **Step 3: Run everything**

```bash
go build ./... && go vet ./... && go test ./...
TEST_DATABASE_URL=... go test -count=1 -p 1 -tags integration ./...
go test -count=1 -tags browser ./internal/web/
```

`-p 1` on the integration run: `internal/boot` and `internal/squirrel` truncate the same live Postgres, and their binaries must not run concurrently. This is in the Makefile and is not optional.

Regenerate the appearance snapshot **only after reading the diff**:

```bash
APPEARANCE=rewrite go test -tags=browser -run TestTheScreensLookLike ./internal/web/
```

Read every line that is not `is new:`. A `→` on a screen this task did not touch is a regression, not a rewrite — the last phase found `/tasks .tcard` changing because a new class collided with an old one.

- [ ] **Step 4: Amend the records**

- `PRODUCT.md` — the chores' four fixed intervals become a number and a unit. Find every place that names the four and say what replaced them, struck through and dated.
- `DESIGN.md` — the chores and tasks screens' sections replaced by what a card in a turn looks like; the interval picker's disclosure replaced by the two-row form; the `.obtn`/`.abtn` note kept.
- `docs/roadmap.md` — a `v0.24.0` entry naming what landed and what it cost.

- [ ] **Step 5: Commit and open the pull request**

```bash
git add -A
git commit -m "feat: the chores and the tasks are messages now"
gh pr create --base main --head feat/the-thread-2 --title "feat: the chores and the tasks join the conversation"
```

The body names what a reviewer should mutation-test first: `composeEvery` going through `ParseEvery` rather than arithmetic, the picker writing one turn rather than one per press, and `rhythmOf` leaving both rows unmarked rather than rounding.

---

## Self-Review

**Spec coverage.** Phase 2's entry says *"the chores and the tasks are absorbed, with the number × unit picker arriving alongside them because it is a chore control; `/chores`, `/tasks`, `/tasks/done` and their templates are deleted."* Task 1 opens a door, Task 2 and Task 4 answer for each place, Task 3 is the picker, Task 5 deletes the screens and amends the records.

**Deliberately not in this plan:** the day/time picker, the pile and the agenda as messages, search as a turn, `/at/{id}`, and the notification's destination. Phases 3 and 4. The chore's day-part question (*when is it worth asking*) and the "go for a bit" timer stay on their existing routes and are not drawn in a turn yet — they are a second question on top of the first, and the interval is the one the owner asked for.

**Type consistency.** `pickView`/`pickRow` are introduced in Task 3 and used only there. `cardView` gains `Photo` in Task 4, which is the one type that changes shape after being introduced — Task 4 says so. `listLimit`, `doorNames`, `choreMeta` and `choreLead` come from Task 1 and are used by Tasks 2 and 4 unchanged.

**Known unverified names**, to check against the tree rather than assume: `toChoreView` and `choreView`'s fields, `toView` and `noteView`'s photograph field, `squirrel.Item`'s photograph field, `newChoreHandler`'s local variable names, `taskActHandler`'s act vocabulary (`done`, `open`, `untask`), and whether `every.html` still has live includers.
