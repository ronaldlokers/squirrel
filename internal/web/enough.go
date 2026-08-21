package web

import "net/http"

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
// Nothing is written here and nothing is read. There is no session, no tally,
// no "you did three" — this route takes no store and touches no row, which is
// the cheapest possible guarantee that it never grows one.
func enoughHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := opts.person(); !ok {
			fail(w, errNoOwner)
			return
		}
		render(w, "enough", view{Here: "enough"})
	}
}
