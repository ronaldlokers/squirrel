package web

import (
	"log/slog"
	"net/http"
)

// Stopping, as a place you can arrive at. Argued in DESIGN.md under The Door
// Art and Principle 3.
//
// Chosen rather than triggered: a screen that appeared after four cards would
// have an opinion about how many cards are enough. Nothing is counted here and
// nothing is read — no session, no tally.
//
// It forgets where you got to, and that is all it touches. Choosing to stop is
// the clearest statement there is that there is nothing to come back to.
func enoughHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		// Being offered your place back after pressing this would make the
		// screen that says stopping is normal into one that argues about it.
		if err := s.EndRun(r.Context(), personID); err != nil {
			slog.Error("forgetting where you got to", "error", err)
		}
		// renderWith, not render: the menu needs a store to count with, and
		// this screen is reachable from the menu so it must carry one back.
		renderWith(w, r, s, opts, "enough", view{Here: "enough"})
	}
}
