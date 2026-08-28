package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A new one, at every door.
//
// A note is a thought you had; a chore or an appointment is one you decided to
// have. So each door's turn carries one chip, and pressing it asks.
//
// The pile's chip posts to the same capture route as the dock, deliberately:
// everything typed lands in the pile, and a second way in with weaker
// durability guarantees would be a second way in nobody could tell apart.

// newChipFor is the chip a door's turn carries, or nothing.
func newChipFor(where string) []turnChip {
	switch where {
	case "chores":
		return []turnChip{{Label: "a new chore", Action: "/chores/ask"}}
	case "tasks":
		return []turnChip{{Label: "a new task", Action: "/tasks/ask"}}
	case "at":
		// Straight to the day picker: an appointment is a day and a time
		// before it is anything else, and /at/new already asks exactly that.
		return []turnChip{{Label: "a new appointment", Action: "/at/new"}}
	case "pile":
		return []turnChip{{Label: "put something down", Action: "/pile/ask"}}
	}
	return nil
}

// alsoOffer adds chips to a turn that has already been drawn.
//
// The alternative was threading the door's name through every turn function so
// each could add its own, which is four call sites and four chances to forget
// one — and forgetting one is invisible, because a missing chip looks exactly
// like a door that has nothing to add.
//
// It reads the turn back and writes it again. A turn whose record cannot be
// read keeps its words and loses the chip, which is the same direction
// turnViews takes: losing the chip is better than losing the turn.
func alsoOffer(t squirrel.Turn, chips ...turnChip) squirrel.Turn {
	if len(chips) == 0 {
		return t
	}
	var sh drawn
	if len(t.Shown) > 0 {
		if err := json.Unmarshal(t.Shown, &sh); err != nil {
			slog.Error("reading a turn back to offer more", "error", err)
			return t
		}
	}
	// Never over a question. A turn waiting for an answer is not the moment to
	// offer starting something else — the same rule the two chips at the foot
	// of the thread follow, for the same reason.
	if sh.Pick != nil || sh.Cal != nil || sh.Say != nil || sh.Cut != nil {
		return t
	}
	sh.Chips = append(sh.Chips, chips...)
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("offering more on a turn", "error", err)
		return t
	}
	t.Shown = body
	return t
}

// askNameHandler is every one of those chips: it asks for words, and says
// which route the words go to.
//
// One handler rather than three, because the only thing that differs is the
// question and where the answer is posted — and three copies of the same six
// lines is three places for one of them to drift.
func askNameHandler(s Store, opts Options, said, question, action, field, does string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: said},
			askInWordsNamed(question, action, field, does, nil),
		}), "/")
	}
}

// choreNameHandler takes the name and asks the second half of the question.
//
// Two turns, because a chore is two answers: what it is, and how often. The
// name travels in the picker's own hidden fields, the way a proposal travels
// in the form that renders it — nothing is written until the interval is
// answered, so abandoning it halfway leaves no half-made chore behind.
func choreNameHandler(s Store, opts Options) http.HandlerFunc {
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
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			// Nothing to make. Silence rather than a scolding: an empty box
			// submitted by accident is not a mistake worth a sentence.
			answerWith(w, r, nil, "/")
			return
		}
		if len(name) > choreNameLimit {
			name = name[:choreNameLimit]
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: name},
			askHowOften("/chores/new", map[string]string{"name": name}, "", "", ""),
		}), "/")
	}
}

// saidRhythm is how often, in the words the picker uses. What you said, said
// back — rather than echoing the form field, which after the picker is two
// fields and reads as neither.
func saidRhythm(every time.Duration) string {
	count, unit := rhythmOf(every)
	if count == "" || unit == "" {
		return "when you asked"
	}
	return "every " + count + " " + unit
}

// One page of a list, and whether there is another.
//
// "The rest" was a link — `/?open=tasks` — and the thread has never read a
// query like that, so it reloaded the conversation and did nothing. Search's
// was worse: `/pile?q=…`, a route deleted with the deck. All three had been
// dead since the doors became messages, and dropping a door's reply from
// twelve cards to five is what would have made somebody find out.
//
// Generic because the three lists are three types and the arithmetic is the
// same. An offset rather than a cursor: these are short lists read from the
// top, and a cursor would be a stable identity that a chore list reordered by
// what is due does not have.
func slice[T any](all []T, from int) (page []T, more bool) {
	if from >= len(all) {
		return nil, false
	}
	to := from + listLimit
	if to >= len(all) {
		return all[from:], false
	}
	return all[from:to], true
}

// theRest is the chip that asks for the next page. A form, and it stays one
// after rooms became links: this is paging inside a room rather than
// navigation to one, and asking for the rest is something you said. See
// openHandler, which is these two jobs behind one route.
func theRest(where string, from int) turnChip {
	return turnChip{
		Label: "the rest", Action: "/open",
		Fields: map[string]string{"where": where, "from": strconv.Itoa(from)},
	}
}
