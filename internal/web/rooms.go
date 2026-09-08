package web

import "net/http"

var theBays = map[string]string{
	"notes":  "/?bay=notes",
	"chores": "/?bay=daily",
	// The agenda stopped being a door on 9 September 2026 and became the
	// sidebar's lower half, which is on the board itself.
	"at":    "/",
	"tasks": "/?bay=tasks",
}

var placesGone = map[string]string{
	"buddy": "/",
	"pile":  "/?bay=notes",
	"held":  "/?shelf=held",
	"kept":  "/?shelf=kept",
}

func roomRoute(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		where := r.PathValue("room")
		if bay, ok := theBays[where]; ok {
			http.Redirect(w, r, bay, http.StatusMovedPermanently)
			return
		}
		if to, ok := placesGone[where]; ok {
			http.Redirect(w, r, to, http.StatusMovedPermanently)
			return
		}
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	}
}
