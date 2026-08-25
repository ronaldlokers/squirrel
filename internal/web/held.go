package web

import (
	"net/http"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Things you cannot act on.
//
// A page until 25 August 2026, and a message since: the screen is gone and
// heldTurn draws the same rows in the conversation. What stayed here is the
// pair of writes — setting one aside, and picking it back up.

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
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
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
		// The same way back the other four dispositions have always had.
		//
		// This route redirected with nothing at all, so a note set aside left
		// the screen with no stamp, no hold and no offer to put it back — the
		// one transition in the product where undo was not one press away, on
		// a card that had simply gone. It was recoverable from /held all
		// along; it did not look it.
		answerWith(w, r, keepSaid(r.Context(), s, personID, append(
			[]squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: squirrel.HeldWords[state]},
				{Who: squirrel.SpeakerBuddy, Words: "Set aside. It is in the set-aside until it is not."},
			},
			pileTurn(r.Context(), s, opts, personID, 0, ""),
		)), "/")
	}
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
