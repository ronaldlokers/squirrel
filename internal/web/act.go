package web

import (
	"log/slog"
	"net/http"
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

// actionStates is the state a card says it is showing, read the way every
// other query and form value here is read: as though a stranger typed it. A
// value that is not one of the four is no claim rather than a bad one, and the
// write falls back to the unconditional one.
var actionStates = map[string]squirrel.ItemState{
	string(squirrel.ItemOpen):    squirrel.ItemOpen,
	string(squirrel.ItemDone):    squirrel.ItemDone,
	string(squirrel.ItemKept):    squirrel.ItemKept,
	string(squirrel.ItemDropped): squirrel.ItemDropped,
}

// intervalSentinel stands in for a chore name while an interval is parsed. The
// literal is copied from apply.go rather than exported: two callers agreeing on a
// word is cheaper than a package's API growing to say it.
const intervalSentinel = "chore-name-placeholder"

// back sends the browser to the pile with a 303, so a reload after triaging
// re-reads rather than re-submitting. The undo hint travels in the query string,
// because this binary has no sessions.
func actHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		personID, known := personOf(r)
		if !known {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Deciding is not disposing, so it takes its own branch rather than
		// joining a map of states and pretending to be one. The three
		// disposals end a note; this moves it, and it stays open because
		// deciding to do a thing is not doing it.
		if act := r.FormValue("act"); act == "task" || act == "note" {
			// Deciding is not disposing, so it takes its own branch. The note's state never
			// moved, so `act=open` would undo nothing: what changed was its kind.
			kind := squirrel.ItemTask
			if act == "note" {
				kind = squirrel.ItemNote
			}
			if _, err := s.SetItemKind(r.Context(), personID, id, kind); err != nil {
				fail(w, err)
				return
			}
			it, _, _ := s.ItemByID(r.Context(), personID, id)
			if act == "note" {
				// Undoing is not a thing to be offered an undo for, so it says
				// so and hands you the pile again rather than offering a way
				// back to where you just came from.
				answerWith(w, r, keepSaid(r.Context(), s, personID, append(
					[]squirrel.Turn{
						{Who: squirrel.SpeakerYou, Words: "put it back"},
						{Who: squirrel.SpeakerBuddy, Words: "It is a note again."},
					},
					pileTurn(r.Context(), s, opts, personID, 0, ""),
				)), "/")
				return
			}
			answerInThread(w, r, s, opts, personID, "task", it.RawText, id, string(it.State))
			return
		}

		state, ok := actions[r.FormValue("act")]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// What the card said the note was when the press happened, if it said. An absent
		// one means the old unconditional write.
		//
		// Writing the state a note already holds is a no-op rather than an error, and
		// this handler must not add a check that turns a retry into a failure.
		from, decided := actionStates[r.FormValue("was")]
		if decided {
			moved, err := s.MoveItemState(r.Context(), it.ID, from, state, time.Now())
			if err != nil {
				fail(w, err)
				return
			}
			if !moved {
				// It moved under you, from the room, while the card was still
				// on the screen. Saying nothing would be the two views
				// disagreeing and neither of them mentioning it.
				answerWith(w, r, keepSaid(r.Context(), s, personID, append(
					[]squirrel.Turn{{
						Who:   squirrel.SpeakerBuddy,
						Words: "That one moved while you were looking at it. It is not where the card said.",
					}},
					pileTurn(r.Context(), s, opts, personID, 0, ""),
				)), "/")
				return
			}
		} else if err := s.SetItemState(r.Context(), it.ID, state, time.Now()); err != nil {
			fail(w, err)
			return
		}
		answerInThread(w, r, s, opts, personID, r.FormValue("act"), it.RawText, it.ID, string(it.State))
	}
}

func choreHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		personID, known := personOf(r)
		if !known {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// ParseEvery returns the name out of the same string, so here a sentinel word is
		// appended and only the duration kept — without it, "every" alone borrows the
		// next word as its unit and silently creates a chore nobody asked for.
		every, ok := offered(r.FormValue("every"))
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, found, err := s.PromoteItem(r.Context(), personID, id, every)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, append(
			[]squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: r.FormValue("count") + " " + r.FormValue("unit")},
				{Who: squirrel.SpeakerBuddy, Words: "It comes back now."},
			},
			pileTurn(r.Context(), s, opts, personID, 0, ""),
		)), "/")
	}
}

// fixHandler changes what a note says and nothing else, through the same store
// call the chat's !fix makes.
func fixHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		text := strings.TrimSpace(r.FormValue("text"))
		// Empty is not a correction. A note cannot be emptied into nothing —
		// that is what dropping it is for, and it is reversible.
		if err != nil || id < 1 || text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}
		if _, err := s.Reword(r.Context(), personID, id, text); err != nil {
			fail(w, err)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, append(
			[]squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: text},
				{Who: squirrel.SpeakerBuddy, Words: "That is what it says now."},
			},
			pileTurn(r.Context(), s, opts, personID, 0, ""),
		)), "/")
	}
}

// answerInThread says what happened and hands you the next note: having decided
// is the moment you are most able to decide again.
func answerInThread(w http.ResponseWriter, r *http.Request, s Store, opts Options,
	personID int64, act, text string, id int64, was string) {
	said := saidAboutANote(act, text, id, was)
	if len(said) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	// Marked on every answer rather than once at the start, so the clock measures
	// silence: a long afternoon of triage never goes stale, and one you walked away
	// from ages out.
	//
	// Best-effort: failing to remember where you got to must not fail the decision
	// you just made.
	if err := s.MarkRun(r.Context(), personID, squirrel.RunPile, now()); err != nil {
		slog.Error("keeping your place", "error", err)
	}

	said = append(said, pileTurn(r.Context(), s, opts, personID, 0, ""))
	answerWith(w, r, keepSaid(r.Context(), s, personID, said), "/")
}

// laterHandler is skipping one, which is not a decision: the note stays where it
// was. Your half is said too — a record that only kept the decisions would be a
// record of a different afternoon.
func laterHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		after, err := strconv.ParseInt(r.FormValue("after"), 10, 64)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "later"},
			pileTurn(r.Context(), s, opts, personID, after, ""),
		}), "/")
	}
}

// undoHandler is changing your mind, from the chip that travelled with the
// answer. The same write as any other act, so there is no second way to move a
// note.
func undoHandler(s Store, opts Options) http.HandlerFunc {
	act := actHandler(s, opts)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		r.Form.Set("from", "thread")
		act(w, r)
	}
}

// askAbout is the three questions a note can be asked, in the conversation.
//
// Each reads the note first, which the words need anyway and which scopes the
// press: a row that is not yours is not yours to ask about.
func askAbout(s Store, opts Options, ask func(it squirrel.Item) squirrel.Turn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
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
		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: it.RawText},
			ask(it),
		}), "/")
	}
}

// moreHandler is `something else?` — the three questions a note can be asked,
// arriving as a turn rather than a panel expanding.
//
// The card above keeps its place, so you can see which note is being discussed,
// and the press goes into the conversation: you paused on that one.
//
// Room appears here for `break it up`, which on the card could only be offered
// when a free check guessed it was worth it.
func moreHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil || id < 1 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		// Checked against what this person actually has, rather than trusted:
		// the id arrives from a form, and the same check every other handler
		// here makes.
		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		of := map[string]string{"id": strconv.FormatInt(id, 10)}
		chips := []turnChip{
			{Label: "make it a chore", Action: "/pile/often", Fields: of},
			{Label: "say it another way", Action: "/pile/reword", Fields: of},
			{Label: "i can't act on this", Action: "/pile/why", Fields: of},
		}
		if splittable(opts, it.RawText) {
			chips = append(chips, turnChip{
				Label: "break it up", Action: "/pile/split",
				Fields: map[string]string{
					"id": strconv.FormatInt(id, 10), "act": "propose", "from": "thread",
				},
			})
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "something else?"},
			sayWithChips("about that one:", chips),
		}), "/")
	}
}
