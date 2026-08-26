package web

import (
	"net/http"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Things you cannot act on. heldTurn draws them in the conversation; what lives
// here are the writes — setting one aside, picking it back up, and saying it is
// still waiting.

// heldActHandler is the two things you can do: set something aside, and pick it
// back up.
//
// Picking it back up is `open`, which is the transition everything else in this
// product already reverses through — so undo works on it without anything new,
// and the pile it returns to is where it came from.
func heldActHandler(s Store, opts Options) http.HandlerFunc {
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
		back := backTolerant(r.FormValue("from"))

		// "still waiting" — the answer that costs nothing.
		//
		// It moves the clock and touches nothing else: same state, same
		// reason, and the note does not come back to the pile. Being able to
		// say "yes, still" without it becoming work is the whole reason
		// mentioning it is safe.
		if r.FormValue("act") == "still" {
			if _, err := s.StillHolding(r.Context(), personID, id, now()); err != nil {
				fail(w, err)
				return
			}
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: "still waiting"},
				{Who: squirrel.SpeakerBuddy, Words: "Right. I will leave it."},
			}), back)
			return
		}

		if r.FormValue("act") == "back" {
			if _, err := s.Unhold(r.Context(), personID, id, now()); err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, back, http.StatusSeeOther)
			return
		}

		// Its own field rather than another `act`, the way the ladder's chips
		// carry `why`: it keeps "an action button submits an act the server
		// understands" a countable invariant, which render_test.go leans on.
		//
		// Checked against the three anyway — a value that was never offered is
		// read the way a stranger's typing is read.
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
		// The same way back the other four dispositions have. This route used
		// to redirect with nothing at all, so a note set aside left the screen
		// with no offer to put it back — recoverable from the set-aside all
		// along, and it did not look it.
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
