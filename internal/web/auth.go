package web

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"
)

// guard is this product's authentication.
//
// It reads a session cookie and puts two things on the request: the person id,
// which almost everything uses, and the sub, which only capture needs.
//
// A GET for a page redirects; nothing else does. An unauthenticated POST, and
// any request carrying X-Thread, gets 401 with no body. A redirect there would
// swallow a form's words into a login screen, and thread.js would paste the
// gate into the conversation as a turn.
func guard(opts Options, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Development is always signed in as person 1. devDir is set only by
		// EnableDevelopment, which lives behind the `dev` build tag — a shipped
		// binary does not contain the code that could make this true.
		if devDir != "" {
			h(w, withWho(r, 1, "dev"))
			return
		}
		carried, err := r.Cookie(sessionCookie)
		if err != nil {
			refuse(w, r, "no cookie")
			return
		}
		session, known, err := opts.Sessions.For(r.Context(), hashOf(carried.Value), now())
		if err != nil {
			// Past the cache's memory and the database cannot answer. This is
			// the one place in the product that fails closed on an unreachable
			// Postgres, and it has to: the alternative is guessing who is
			// asking.
			slog.Error("could not tell who is asking", "error", err)
			refuse(w, r, "cannot say")
			return
		}
		if !known || session.PersonID == 0 {
			refuse(w, r, "no session")
			return
		}
		h(w, withWho(r, session.PersonID, session.Sub))
	}
}

// wantsAPage is a plain browser navigation rather than a form post or a fetch
// from thread.js. Only one of those can be sent to a login screen.
func wantsAPage(r *http.Request) bool {
	return r.Method == http.MethodGet && r.Header.Get("X-Thread") == ""
}

// refuse is what a request with nobody behind it gets.
//
// No body on the 401: a refusal that describes what it is refusing tells an
// unauthenticated caller that there is something here.
func refuse(w http.ResponseWriter, r *http.Request, why string) {
	slog.Warn("refused the pile", "why", why, "path", r.URL.Path)
	if !wantsAPage(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	to := url.URL{Path: "/auth"}
	if r.URL.Path != "/" {
		// Where you were going, so signing in puts you back rather than at the
		// top of the pile. Only the path: a query string can carry anything,
		// and the gate checks what it is given anyway.
		to.RawQuery = url.Values{"next": {r.URL.Path}}.Encode()
	}
	http.Redirect(w, r, to.String(), http.StatusSeeOther)
}

// sameSecret compares two secrets without leaking which byte differed.
func sameSecret(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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
// A request nobody has been put on is nobody, and never a default: a fallback
// here would be a silent cross-pile read.
func personOf(r *http.Request) (int64, bool) { return personIn(r.Context()) }

// personIn is the same read from a context that has no request around it.
func personIn(ctx context.Context) (int64, bool) {
	w, ok := ctx.Value(whoKey{}).(who)
	return w.personID, ok && w.personID != 0
}

// subOf is the OIDC subject, which only the capture path needs: it writes the
// sub as a sender string so the drain can resolve a spooled capture's owner.
func subOf(r *http.Request) string {
	w, _ := r.Context().Value(whoKey{}).(who)
	return w.sub
}
