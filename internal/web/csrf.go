package web

import (
	"log/slog"
	"net/http"
	"net/url"
)

// sameOrigin refuses a write that was submitted from another site.
//
// The identity on this screen arrives in a header that Traefik fills from an
// Authentik session cookie, and a cross-site form post travels with that
// cookie like any other request. Without this check, a page the owner happens
// to visit could post to /pile/act and move notes around. Nothing is lost —
// every transition reverses — but nothing asked for it either.
//
// The check is Origin, falling back to Referer for the browsers that still
// omit Origin on a same-origin form post, and a request carrying neither is
// refused. Deliberately not a token: a token needs somewhere to live, and this
// binary has no session, no cookie and no secret to mint one from. A browser
// will not let a page lie about its own origin, which is the whole guarantee
// this needs.
//
// It compares against r.Host, so the proxy in front must pass the original
// Host through — Traefik does by default; see docs/pile-screen.md.
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
