package web

import (
	"log/slog"
	"net/http"
)

// guard is the whole of this product's authentication, and that is the point.
// Traefik calls an Authentik outpost, Authentik decides, and Squirrel compares
// one header to one configured value. No sessions, no cookies, no redirect
// flow, no OIDC library in a binary that has none of that anywhere else.
//
// The comparison is exact. Trimming or lower-casing the header would be this
// file quietly deciding that two identities are the same one.
func guard(opts Options, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who := r.Header.Get(opts.IdentityHeader)
		if who == "" || who != opts.Identity {
			// No body: a refusal that describes what it is refusing tells an
			// unauthenticated caller that there is something here.
			slog.Warn("refused the pile", "identity", who, "path", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
