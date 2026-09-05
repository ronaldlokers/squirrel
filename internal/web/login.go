package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
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
// Four values in one cookie rather than four cookies, because they are one
// fact and expire together: a login half-abandoned should not leave a verifier
// behind for a state that is gone.
//
// Base64 because `next` is a path and a path is not cookie-safe. Go drops
// illegal bytes out of a cookie value rather than refusing it, so an unencoded
// separator does not fail here — it arrives silently truncated, and every
// login turns into "that was not our redirect".
func started(state, nonce, verifier, next string) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strings.Join([]string{state, nonce, verifier, next}, "\x1f")))
}

func unpack(v string) (state, nonce, verifier, next string, ok bool) {
	raw, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return "", "", "", "", false
	}
	parts := strings.Split(string(raw), "\x1f")
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

// beginHandler starts a login.
//
// A POST because it writes. It also means a prefetch, a crawler or a
// link-preview cannot begin one.
func beginHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := secret(32)
		if err == nil {
			var nonce string
			if nonce, err = secret(32); err == nil {
				var verifier string
				if verifier, err = secret(32); err == nil {
					var away string
					// The gate may not have found authentik yet — it is looked up
					// lazily and retried rather than at boot, so that an authentik
					// that is down costs the way in and not the whole product. The
					// screen already has a sentence for this.
					if away, err = opts.Gate.Away(state, nonce, verifier); err == nil {
						http.SetCookie(w, cookie(stateCookie,
							started(state, nonce, verifier, backTolerant(r.FormValue("next"))), stateLife))
						http.Redirect(w, r, away, http.StatusSeeOther)
						return
					}
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
		state, nonce, verifier, next, ok := unpack(carried.Value)
		// Constant time because it is a secret comparison, even though what it
		// protects against is a forged redirect rather than a stolen token.
		if !ok || !sameSecret(state, r.URL.Query().Get("state")) {
			down("that was not our redirect", nil)
			return
		}

		who, err := opts.Gate.Back(r.Context(), r.URL.Query().Get("code"), verifier, nonce)
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

		// Best-effort, and after the person exists: what you are called and
		// what you look like are not what lets you in, and failing to fetch a
		// picture must never be a failed sign-in.
		rememberWho(r.Context(), opts, personID, who)

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

// aPicture is every type this will store and serve, and the list is short on
// purpose.
//
// image/svg+xml is an image type and is deliberately absent: an SVG is a
// document, it runs script, and one served from this origin at /me/face would
// be stored cross-site scripting wearing somebody's avatar. Nothing here needs
// vector art, so nothing here accepts it.
var aPicture = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// faceLimit is the most of a picture worth keeping. Authentik serves an avatar,
// not a photograph, and a provider that answers with something enormous is a
// provider to ignore rather than to store.
const faceLimit = 512 << 10

// remember keeps the display name and, once, the picture. Everything here is
// best-effort: it runs after the session's person exists and before the cookie,
// and every failure is a log line rather than a refused login.
func rememberWho(ctx context.Context, opts Options, personID int64, who Person) {
	if opts.RememberWho == nil {
		return
	}
	// The username stands in when the provider sends no name, because the
	// stored handle cannot: handleFor makes it unique with a hash of the sub,
	// and "ronald-cf1cab94" is a row, not a person.
	name := who.Name
	if name == "" {
		name = who.Handle
	}
	face, kind := fetchFace(ctx, opts, who.Face)
	if err := opts.RememberWho(ctx, personID, name, face, kind); err != nil {
		slog.Error("remembering who signed in", "error", err)
	}
}

// fetchFace pulls the avatar once, so nothing on the screen depends on the
// provider still being reachable. Empty on any doubt at all: a wrong picture is
// worse than none, and none is drawn as a monogram.
func fetchFace(ctx context.Context, opts Options, from string) ([]byte, string) {
	if from == "" {
		return nil, ""
	}
	u, err := url.Parse(from)
	if err != nil || u.Scheme != "https" {
		slog.Warn("ignoring a picture that is not https", "url", from)
		return nil, ""
	}
	ctx, stop := context.WithTimeout(ctx, 5*time.Second)
	defer stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, from, nil)
	if err != nil {
		return nil, ""
	}
	client := opts.Fetch
	if client == nil {
		client = onlyTheOpenInternet()
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Warn("could not fetch your picture", "error", err)
		return nil, ""
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slog.Warn("the picture answered badly", "status", res.StatusCode)
		return nil, ""
	}
	// The declared type, without its parameters: "image/png; charset=utf-8" is
	// a picture with something to say about itself.
	kind, _, _ := strings.Cut(res.Header.Get("Content-Type"), ";")
	kind = strings.TrimSpace(strings.ToLower(kind))
	if !aPicture[kind] {
		slog.Warn("that is not a picture worth serving", "type", kind)
		return nil, ""
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, faceLimit+1))
	if err != nil || len(body) == 0 || len(body) > faceLimit {
		slog.Warn("ignoring a picture that would not fit", "bytes", len(body))
		return nil, ""
	}
	// And the bytes have to agree with the header, because the header is the
	// remote server's opinion and the bytes are what a browser will act on.
	sniffed, _, _ := strings.Cut(http.DetectContentType(body), ";")
	if sniffed != kind {
		slog.Warn("a picture that is not what it says it is", "said", kind, "is", sniffed)
		return nil, ""
	}
	return body, kind
}

// faceHandler serves your picture from this origin.
//
// From here rather than from Authentik, so the one face on the screen is not a
// third-party request: no referrer to the provider on every render, nothing to
// go missing when the network does, and the browser's own cache can hold it.
func faceHandler(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, known := personOf(r)
		if !known {
			fail(w, errNoOwner)
			return
		}
		face, kind, found, err := s.PersonFace(r.Context(), personID)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			// The monogram is drawn, not served: a missing picture is a shape
			// this product already knows how to make.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Never the stored string on trust: a row written before this list
		// existed could say anything, and what a browser does with a document
		// is decided by this header.
		if !aPicture[kind] {
			kind = "application/octet-stream"
		}
		w.Header().Set("Content-Type", kind)
		// And no sniffing, so a mislabelled body cannot be promoted into
		// something that runs.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Yours, so it never belongs to a shared cache, and short so a picture
		// changed at the provider arrives on the next sign-in rather than never.
		w.Header().Set("Cache-Control", "private, max-age=300")
		_, _ = w.Write(face)
	}
}

// onlyTheOpenInternet is the client that fetches a picture, and the reason it
// is not http.DefaultClient.
//
// The URL is somebody else's. It arrives in the `picture` claim of a verified
// id token, which proves Authentik said it and proves nothing about where it
// points: a misconfigured or compromised provider could aim it at
// 169.254.169.254, at a service on the cluster network, or at localhost, and
// this server would fetch it, store the bytes and serve them back at /me/face.
// A signed claim is not a safe URL.
//
// The check is on the address actually being dialled rather than on a
// hostname resolved beforehand, because a name resolved twice can answer twice
// — public on the check and private on the connection. Control runs after
// resolution and before connect, so there is no gap to win.
func onlyTheOpenInternet() *http.Client {
	dial := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("refusing an address that will not split: %w", err)
			}
			if ip := net.ParseIP(host); !onTheOpenInternet(ip) {
				return fmt.Errorf("refusing a picture from %s", host)
			}
			return nil
		},
	}
	return &http.Client{
		Transport: &http.Transport{DialContext: dial.DialContext},
		// A redirect is a second URL nobody checked. Same two rules, every hop.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects for a picture")
			}
			if req.URL.Scheme != "https" {
				return errors.New("refusing a redirect that is not https")
			}
			return nil
		},
	}
}

// onTheOpenInternet is whether an address is somewhere on the open internet, which is
// the only place a picture may come from. Everything else is either this
// machine, this network, or the cloud metadata service.
func onTheOpenInternet(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	// Shared address space (RFC 6598), which is neither private by Go's
	// reckoning nor anywhere a picture lives.
	if four := ip.To4(); four != nil && four[0] == 100 && four[1] >= 64 && four[1] <= 127 {
		return false
	}
	return true
}
