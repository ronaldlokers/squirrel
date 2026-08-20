package web

import (
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The home screen reads two things, and both are about the minute you are in
// rather than about what is waiting.
//
// It used to read almost nothing, and that absence was the design: a home
// screen that shows what is waiting greets you with what is waiting, however
// carefully it is dressed. That argument still holds and is why the doors and
// the slot depend on nothing — a full pile and an empty one render the same
// bytes below the offer.
//
// What changed is that "nothing about the pile" was doing a second job it was
// never meant to do: it also meant Squirrel never chose anything, so the front
// door asked which of three boxes you would like to open. The offer is one
// thing the product picked, with an action on it. It is not a preview of the
// pile and it is not a count of anything; when there is nothing to hand you it
// is absent rather than empty, which is the same rule the deck's own empty
// state follows.
//
// The check-in and the offer share one region, and the sharing is the point:
// the question configures the page rather than filing a diary entry, so home
// has exactly one interactive thing above the doors. Stale reading, and it
// asks; fresh reading, and it offers.
func homeHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		v := view{
			Home: true,
			// All four arrive from the address bar, so they are read the way a
			// stranger's typing is read: a present flag, and words that are
			// escaped on the way out like any other text on this screen.
			Kept:    q.Get("kept") != "",
			NoKeep:  q.Get("nokeep") != "",
			NoPhoto: q.Get("nophoto") != "",
			Held:    q.Get("held") != "",
			Said:    q.Get("said"),
			// "Show me anyway" lifts the capacity gate for this render and
			// nothing else. It lives in the address bar rather than in the
			// database on purpose: a person who says they are wiped and then
			// works anyway has not changed their answer, and storing it would
			// make them re-decide tomorrow that they meant it.
			Anyway: q.Get("anyway") != "",
		}

		// `?ask=1` is "say something else": the question again, without
		// forgetting what was said. Changing your mind is not a special case,
		// it is the same answer given twice.
		if q.Get("ask") == "" {
			if personID, ok := opts.person(); ok {
				if c, found, err := s.LatestCheckin(r.Context(), personID); err == nil && found && c.Fresh(now()) {
					v.Mood, v.MoodWord = string(c.Mood), squirrel.Words[c.Mood]
				}
			}
		}
		if v.Mood == "" {
			for _, m := range squirrel.Moods {
				v.Faces = append(v.Faces, faceView{Mood: string(m), Word: squirrel.Words[m]})
			}
		} else {
			// Only once the question has been answered. Asking how you are and
			// then handing you a job in the same breath is the interruption
			// this product exists to reduce, and the answer is what shapes the
			// offer anyway.
			// Home may pay. It is the screen the decision is *for*, and the
			// cache is what keeps a run of opens down to one call.
			v.Offer = offerFor(s, opts, r, v.Anyway, true)
			if v.Offer != nil {
				// The ladder's answer, when one has been asked for. It hangs
				// off the offer rather than replacing it: the thing you could
				// not start is still the thing.
				v.Offer.Unstuck = withStep(unstuckFrom(q), s, opts, r)
			}
		}

		renderWith(w, r, s, opts, "home", v)
	}
}
