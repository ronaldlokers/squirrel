package web

import (
	"context"
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
		id, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		h(w, withWho(r, id, opts.Identity))
	}
}

type whoKey struct{}

type who struct {
	personID int64
	sub      string
}

// withWho puts the person on the request. Only guard calls it, and only after
// it has decided.
func withWho(r *http.Request, personID int64, sub string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), whoKey{}, who{personID: personID, sub: sub}))
}

// personOf is whose pile this request is about.
//
// It replaced Options.Owner on 25 August 2026. That was a process-global
// atomic.Int64 read through personOf(r), and every handler read it. It could
// not survive two people: a second person's request would have read the first
// one's owner and drawn their pile.
//
// A request nobody has been put on is nobody, and never a default. A fallback
// here would be exactly the silent cross-pile read this change exists to make
// impossible.
func personOf(r *http.Request) (int64, bool) {
	w, ok := r.Context().Value(whoKey{}).(who)
	return w.personID, ok && w.personID != 0
}

// subOf is the OIDC subject, which only the capture path needs: it writes the
// sub as a sender string so the drain can resolve a spooled capture's owner.
func subOf(r *http.Request) string {
	w, _ := r.Context().Value(whoKey{}).(who)
	return w.sub
}
