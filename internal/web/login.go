package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Signing in, and out.
//
// Three routes and none of them is behind guard, which is the whole point: a
// person with no session has to be able to get one.

const (
	// sessionCookie is who you are.
	sessionCookie = "squirrel_session"
	// stateCookie carries the login in progress: the state to compare, the
	// PKCE verifier to spend, and where you were going. It lives for as long
	// as a trip to Authentik and back plausibly takes.
	stateCookie = "squirrel_login"
	stateLife   = 10 * time.Minute
	// sessionLife is how long a session lasts without being used. Long,
	// because being signed out of your own notes is the failure this product
	// can least afford — and every use pushes it out again, so this is the
	// gap after which you have stopped using Squirrel rather than a timer on
	// using it.
	sessionLife = 30 * 24 * time.Hour
)

// secret is n bytes from crypto/rand, base64url. Used for the session token,
// the state and the PKCE verifier: three things that must be unguessable and
// have no other requirement.
func secret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashOf is what goes in the table. Not for secrecy — the token has 256 bits
// of entropy — but so that read access to the sessions table is not read
// access to the product.
func hashOf(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// started is the login in progress, packed into one cookie.
//
// Three values in one cookie rather than three cookies, because they are one
// fact and expire together: a login half-abandoned should not leave a verifier
// behind for a state that is gone.
//
// Base64 because `next` is a path and a path is not cookie-safe. Go drops
// illegal bytes out of a cookie value rather than refusing it, so an unencoded
// separator does not fail here — it arrives silently truncated, and every
// login turns into "that was not our redirect".
func started(state, verifier, next string) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strings.Join([]string{state, verifier, next}, "\x1f")))
}

func unpack(v string) (state, verifier, next string, ok bool) {
	raw, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return "", "", "", false
	}
	parts := strings.Split(string(raw), "\x1f")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// beginHandler starts a login.
//
// A POST because it writes. It also means a prefetch, a crawler or a
// link-preview cannot begin one.
func beginHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := secret(32)
		if err == nil {
			var verifier string
			if verifier, err = secret(32); err == nil {
				var away string
				// The gate may not have found authentik yet — it is looked up
				// lazily and retried rather than at boot, so that an authentik
				// that is down costs the way in and not the whole product. The
				// screen already has a sentence for this.
				if away, err = opts.Gate.Away(state, verifier); err == nil {
					http.SetCookie(w, cookie(stateCookie,
						started(state, verifier, backTolerant(r.FormValue("next"))), stateLife))
					http.Redirect(w, r, away, http.StatusSeeOther)
					return
				}
			}
		}
		// No randomness, or no authentik. Neither is a login, and neither is
		// anything the person pressing the button can do something about.
		slog.Error("could not start a login", "error", err)
		http.Redirect(w, r, "/auth?said=down", http.StatusSeeOther)
	}
}

// backHandler is where Authentik sends you.
//
// Every failure lands on the gate rather than on an error page. This is the
// one screen a person can reach with no session, so it is the only place a
// failure here has to offer.
func backHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		down := func(why string, err error) {
			slog.Warn("a login did not land", "why", why, "error", err)
			http.SetCookie(w, cookie(stateCookie, "", -time.Second))
			http.Redirect(w, r, "/auth?said=down", http.StatusSeeOther)
		}

		carried, err := r.Cookie(stateCookie)
		if err != nil {
			// No state to check against. A bare /auth/callback?code=... that
			// anybody can type, and there is nothing here that began it.
			down("nothing was started", err)
			return
		}
		state, verifier, next, ok := unpack(carried.Value)
		// Constant time because it is a secret comparison, even though what it
		// protects against is a forged redirect rather than a stolen token.
		if !ok || !sameSecret(state, r.URL.Query().Get("state")) {
			down("that was not our redirect", nil)
			return
		}

		who, err := opts.Gate.Back(r.Context(), r.URL.Query().Get("code"), verifier)
		if errors.Is(err, ErrNotAllowed) {
			// Not a failure. Authentik did its job for an account that is
			// simply not for this product, and the gate says so in its own
			// words — which name no group.
			slog.Info("an account was refused")
			http.SetCookie(w, cookie(stateCookie, "", -time.Second))
			http.Redirect(w, r, "/auth?said=no", http.StatusSeeOther)
			return
		}
		if err != nil {
			down("the gate refused the code", err)
			return
		}

		personID, err := opts.Login(r.Context(), who.Sub, who.Handle)
		if err != nil {
			down("could not find or make the person", err)
			return
		}

		token, err := secret(32)
		if err != nil {
			down("no randomness for a session", err)
			return
		}
		// The session is written before the cookie is set. Failing open here
		// would hand somebody a cookie for a session that does not exist,
		// which reads as being signed in and behaves as being signed out.
		if err := opts.Sessions.Open(r.Context(), personID, who.Sub,
			hashOf(token), now(), lifeOf(opts)); err != nil {
			down("could not open a session", err)
			return
		}

		http.SetCookie(w, cookie(stateCookie, "", -time.Second))
		http.SetCookie(w, cookie(sessionCookie, token, lifeOf(opts)))
		http.Redirect(w, r, backTolerant(next), http.StatusSeeOther)
	}
}

// outHandler is signing out.
//
// It lands on the gate rather than on /, and that is not a detail: deleting
// the session and landing on / would redirect to Authentik, which still has
// its own session, and sign you straight back in.
//
// Signing out with no session is not an error. Pressing it twice is a thing
// somebody does.
func outHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if carried, err := r.Cookie(sessionCookie); err == nil {
			token := hashOf(carried.Value)
			if err := opts.Sessions.End(r.Context(), token); err != nil {
				// The cookie is cleared anyway. A sign-out that leaves you
				// looking signed in because the database was busy is the wrong
				// half to get right.
				slog.Warn("a session could not be ended", "error", err)
			}
			opts.Sessions.Forget(token)
		}
		http.SetCookie(w, cookie(sessionCookie, "", -time.Second))
		http.Redirect(w, r, "/auth?said=out", http.StatusSeeOther)
	}
}

// cookie is every cookie this product sets, and they all have the same shape.
//
// SameSite=Lax rather than Strict because the callback arrives as a top-level
// navigation from Authentik, and Strict drops the cookie on exactly that hop.
// Secure unconditionally: this is served over HTTPS behind Traefik and there
// is no development mode that is not.
func cookie(name, value string, life time.Duration) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		MaxAge: int(life.Seconds()),
	}
}

func lifeOf(opts Options) time.Duration {
	if opts.SessionLife > 0 {
		return opts.SessionLife
	}
	return sessionLife
}
