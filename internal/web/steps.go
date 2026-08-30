package web

import (
	"net/http"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A thing broken into steps, on the screen.
//
// One step is shown. Never the sequence, never a position out of a total,
// never a progress bar — a bar is a count in a costume, and a count of what
// you have left to do is the accruing number this product refuses.
//
// The store is what makes that more than an intention: there is no function on
// it that returns the list, so this file could not render one.

// stepFor is the step to show, or nil.
//
// Nil for every failure, like the picker's own reader: a sequence that cannot
// be read must not take down a page that rendered without one for the
// product's whole life.
func stepFor(s Store, opts Options, r *http.Request) *stepView {
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	st, found, err := s.NextStep(r.Context(), personID)
	if err != nil || !found {
		return nil
	}
	return &stepView{ID: st.ID, Label: st.Label, Body: st.Body, Last: st.Last}
}

// smallerFor breaks the thing being offered into steps and hands back the
// first one, or nil.
//
// Called on the path where the ladder has just answered "too big". The fixed
// line is already what renders when this returns nil, so a model that is slow,
// absent or wrong costs nothing anyone can see.
func smallerFor(s Store, opts Options, r *http.Request, b squirrel.Blocker, o squirrel.Offer) *stepView {
	personID, ok := personOf(r)
	if !ok || opts.Smaller == nil || !squirrel.BreakingHelps(b) || o.Text == "" {
		return nil
	}
	steps, ok := opts.Smaller(r.Context(), personID, o.Text, squirrel.BlockerWords[b])
	if !ok {
		return nil
	}

	var itemID *int64
	if o.Kind == squirrel.OfferTask && o.RefID != 0 {
		id := o.RefID
		itemID = &id
	}
	if err := s.SaveSteps(r.Context(), personID, itemID, o.Text, steps); err != nil {
		return nil
	}
	return stepFor(s, opts, r)
}

// stepsHandler is the two things you can do with a sequence: finish the step
// you were shown, or throw the whole thing away.
//
// Throwing it away costs one press, has no consequence and asks nothing back —
// the same shape "not now" already has. A breakdown you did not want must be
// as cheap to dismiss as it was to produce.
func stepsHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		back := backTolerant(r.FormValue("from"))

		switch r.FormValue("act") {
		case "done":
			id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
			if err != nil {
				http.Redirect(w, r, back, http.StatusSeeOther)
				return
			}
			if err := s.StepDone(r.Context(), personID, id, now()); err != nil {
				fail(w, err)
				return
			}
		case "clear":
			if err := s.ClearSteps(r.Context(), personID); err != nil {
				fail(w, err)
				return
			}
		}
		http.Redirect(w, r, back, http.StatusSeeOther)
	}
}

// stepView is one step and the two presses that belong to it. There is
// deliberately nowhere here to put a second step: the failure being avoided is
// the twelve-step plan, and a struct that cannot hold one cannot render one.
type stepView struct {
	ID    int64
	Label string
	Body  string
	// Last says nothing comes after, so the surface can say so rather than
	// leaving you waiting for a step that never arrives. Not a position and
	// not a total — only whether this is the end.
	Last bool
}
