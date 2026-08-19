package web

import (
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// One answer, recorded, and then the page says the one word back.
//
// It records rather than replaces: the history is what lets a nudge be
// gentler, and it is the reason for asking at all. Nothing here reads it back
// as a series, and there is no store function that could.
func moodHandler(s Store, opts Options) http.HandlerFunc {
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
		m, ok := squirrel.ParseMood(r.FormValue("mood"))
		if !ok {
			// Not one of the five. This arrives from a form, so it is read the
			// way a stranger's typing is read: no answer rather than a wrong
			// one.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := s.RecordCheckin(r.Context(), personID, m, "screen", now()); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
