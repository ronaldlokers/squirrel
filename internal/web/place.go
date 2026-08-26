package web

import (
	"log/slog"
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Starting fresh.
//
// The other half of being offered your place back, and it has to be as easy to
// say as carrying on. After forty minutes away either answer can be the honest
// one — you may have come back to finish, or the afternoon may have moved on —
// so neither is drawn louder than the other and this one costs a single press.
//
// It forgets the run and opens the pile from the top, which is what "fresh"
// means. It does not touch a single note: nothing you decided is undone, and
// nothing you skipped is un-skipped. The only thing thrown away is the memory
// that you were part way through.
func freshHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := s.EndRun(r.Context(), personID); err != nil {
			// Best effort, and the failure is mild: the run ages out on its
			// own, so the worst case is being offered it once more.
			slog.Error("forgetting where you got to", "error", err)
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "start fresh"},
			pileTurn(r.Context(), s, opts, personID, 0, ""),
		}), "/")
	}
}
