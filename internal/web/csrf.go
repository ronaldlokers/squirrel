package web

import (
	"log/slog"
	"net/http"
	"net/url"
)

// sameOrigin refuses a write that was submitted from another site.
//
// A cross-site form post travels with this site's session cookie like any other
// request, so without this check a page the owner happens to visit could post
// to /pile/act and move notes around. Nothing is lost — every transition
// reverses — but nothing asked for it either.
//
// Origin, falling back to Referer for browsers that still omit Origin on a
// same-origin form post; neither present is a refusal. Deliberately not a
// token: a browser will not let a page lie about its own origin, which is the
// whole guarantee this needs.
//
// It compares against r.Host, so the proxy in front must pass the original Host
// through.
func sameOrigin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claimed := r.Header.Get("Origin")
		if claimed == "" {
			claimed = r.Header.Get("Referer")
		}
		u, err := url.Parse(claimed)
		if err != nil || u.Host == "" || u.Host != r.Host {
			slog.Warn("refused a cross-site write",
				"origin", claimed, "host", r.Host, "path", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
