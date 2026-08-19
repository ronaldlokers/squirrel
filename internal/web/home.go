package web

import (
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The home screen reads almost nothing, and that absence is the design.
//
// A home screen that shows what is waiting greets you with what is waiting,
// however carefully it is dressed — so nothing here depends on what the pile
// holds. A full pile and an empty one render the same bytes, which also means
// there is nothing on this page to disagree with the chat about.
//
// The one thing it does read is how you are right now, which is not a fact
// about the pile: it is about the minute you are in, it is answered here, and
// it is shown back one reading at a time. If the database cannot answer, the
// question is simply asked — a check-in that cannot be read is not a reason to
// fail a page that otherwise needs nothing.
func homeHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		v := view{
			Home: true,
			// All three arrive from the address bar, so they are read the way
			// a stranger's typing is read: a present flag, and words that are
			// escaped on the way out like any other text on this screen.
			Kept:   q.Get("kept") != "",
			NoKeep: q.Get("nokeep") != "",
			Held:   q.Get("held") != "",
			Said:   q.Get("said"),
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
		}

		renderWith(w, r, s, opts, "home", v)
	}
}
