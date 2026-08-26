package web

import (
	"log/slog"
	"net/http"
)

// Stopping, as a place you can arrive at.
//
// The product has always said stopping is a normal ending, and the sentence
// carrying that — "stop whenever you like" — was twelve pixels at the bottom
// of the pile. Meanwhile the two screens that do end well, the bottom of the
// pile and the empty states, are places almost nobody reaches: they need you
// to have cleared everything, which is the exact thing the product says you
// should not be trying to do.
//
// So the ending the ordinary session has — three notes in, that will do — gets
// somewhere to be. It is chosen rather than triggered, and that is the whole
// design: a screen that appeared after four cards would be a screen with an
// opinion about how many cards are enough, and the number would be a count
// wearing a kind face.
//
// Nothing is counted here and nothing is read. There is no session, no tally,
// no "you did three".
//
// It took no store at all until 26 August 2026, which was the cheapest possible
// guarantee that it never grew one. It forgets where you got to now, and that
// is the whole of what it touches: choosing to stop is the clearest statement
// there is that there is nothing to come back to.
func enoughHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		// Stopping ends the run. Choosing to stop is the clearest statement
		// there is that there is nothing to come back to, and being offered
		// your place back after pressing this would make the screen that says
		// stopping is normal into one that argues with you about it.
		//
		// This is the one thing this route touches, and it is worth saying
		// that the comment above about taking no store is now half true: it
		// reads nothing and counts nothing. It forgets one row.
		if err := s.EndRun(r.Context(), personID); err != nil {
			slog.Error("forgetting where you got to", "error", err)
		}
		render(w, "enough", view{Here: "enough"})
	}
}
