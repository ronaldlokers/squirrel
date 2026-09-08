package web

import (
	"net/http"
)

func atOneHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
