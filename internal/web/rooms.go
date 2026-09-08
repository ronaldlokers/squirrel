package web

import "net/http"

var theBays = map[string]string{
	"notes":  "/?bay=notes",
	"chores": "/?bay=daily",
	"at":     "/?bay=agenda",
	"tasks":  "/?bay=tasks",
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
