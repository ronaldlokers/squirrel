package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Things you cannot act on, on the screen.
//
// Its own page rather than a fourth door on home. Home carries three doors and
// an offer, and a door for the things you have explicitly set aside is the
// opposite of what setting them aside was for — it would put them back in front
// of you, which is the one thing they exist to stop.
//
// Reached instead from the bottom of the tasks, in the same quiet line that
// already reaches the archive. That is where you look when you are wondering
// what happened to something.

// heldLimit caps the list, like every other list here. The cap is what makes
// "there is more" truthful.
const heldLimit = 20

func heldHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		held, more, err := s.HeldItems(r.Context(), personID, heldLimit)
		if err != nil {
			fail(w, err)
			return
		}
		renderWith(w, r, s, opts, "held", view{
			Here: "held", Scrolling: true, More: more, SetAside: heldGroups(held),
		})
	}
}

// heldActHandler is the two things you can do: set something aside, and pick it
// back up.
//
// Picking it back up is `open`, which is the transition everything else in this
// product already reverses through — so undo works on it without anything new,
// and the pile it returns to is where it came from.
func heldActHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/held", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Redirect(w, r, "/held", http.StatusSeeOther)
			return
		}
		back := backTolerant(r.FormValue("from"))

		if r.FormValue("act") == "back" {
			if _, err := s.Unhold(r.Context(), personID, id, now()); err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, back, http.StatusSeeOther)
			return
		}

		// Setting one aside carries its own field rather than another `act`,
		// the same way the ladder's four chips carry `why`. It keeps "an
		// action button submits an act the server understands" true as a
		// countable invariant, which render_test.go leans on.
		//
		// Checked against the three anyway: the value arrives from a button
		// this screen drew, and a value that was never offered is read the way
		// a stranger's typing is read.
		state, ok := squirrel.ParseHeld(r.FormValue("aside"))
		if !ok {
			http.Redirect(w, r, back, http.StatusSeeOther)
			return
		}
		if _, err := s.HoldItem(r.Context(), personID, id, state,
			r.FormValue("because"), now()); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, back, http.StatusSeeOther)
	}
}

// heldGroups is the list, grouped by which of the three.
//
// Grouped because they are different questions: the waiting ones have somebody
// to chase, the blocked ones have something to arrive, and the somedays have
// neither and want leaving alone. A single list would make you read the reason
// on every row to know which kind you were looking at.
//
// A group with nothing in it does not render, so the page is never a set of
// empty headings telling you what you have not got.
func heldGroups(held []squirrel.HeldItem) []heldGroup {
	groups := make([]heldGroup, 0, len(squirrel.Held))
	for _, state := range squirrel.Held {
		var rows []heldView
		for _, h := range held {
			if h.State != state {
				continue
			}
			rows = append(rows, heldView{
				ID: h.ID, Text: h.Text, Because: h.Because,
				Task: h.Kind == squirrel.ItemTask,
			})
		}
		if len(rows) == 0 {
			continue
		}
		groups = append(groups, heldGroup{
			Word: strings.ToUpper(squirrel.HeldWords[state]),
			Rows: rows,
		})
	}
	return groups
}

// heldGroup is one of the three, with what is in it. No count on the heading:
// a number beside stalled work is a reproach, and the point of setting it
// aside was to stop being asked about it.
type heldGroup struct {
	Word string
	Rows []heldView
}

// heldView is one thing you cannot act on. Because is what would move it, and
// is often empty — someday is not waiting on anything.
type heldView struct {
	ID      int64
	Text    string
	Because string
	// Task says this was something you decided to do rather than a thought you
	// parked. The two read differently and the row says which.
	Task bool
}

// heldChips is the three, for the card that offers them. The order is the
// core's own: the two something outside you will end, then the one only you
// can.
func heldChips() []chipView {
	out := make([]chipView, 0, len(squirrel.Held))
	for _, state := range squirrel.Held {
		out = append(out, chipView{Why: string(state), Word: squirrel.HeldWords[state]})
	}
	return out
}
