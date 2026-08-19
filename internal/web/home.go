package web

import "net/http"

// The home screen takes no Store, and that absence is the design.
//
// A home screen that shows what is waiting greets you with what is waiting,
// however carefully it is dressed — so this one shows nothing that depends on
// what the pile holds. A full pile and an empty one render the same bytes,
// which also means there is nothing here to disagree with the chat about and
// nothing on the page to triage.
//
// It answers when Postgres does not, for the same reason: there is nothing on
// it to fail.
func homeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		render(w, "home", view{
			Home: true,
			// All three arrive from the address bar, so they are read the way
			// a stranger's typing is read: a present flag, and words that are
			// escaped on the way out like any other text on this screen.
			Kept:   q.Get("kept") != "",
			NoKeep: q.Get("nokeep") != "",
			Said:   q.Get("said"),
		})
	}
}
