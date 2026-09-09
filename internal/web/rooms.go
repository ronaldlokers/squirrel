package web

import "net/http"

var theBays = map[string]string{
	"notes":  "/notes",
	"chores": "/?bay=daily",
	// The agenda stopped being a door on 9 September 2026 and became the
	// sidebar's lower half, which is on the board itself.
	"at":    "/",
	"tasks": "/?bay=once",
}

var placesGone = map[string]string{
	"buddy": "/",
	"pile":  "/notes",
	"held":  "/notes",
	"kept":  "/notes",
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
