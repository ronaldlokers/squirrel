package web

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// The application doing OIDC itself.
//
// Until 25 August 2026 this was a Traefik middleware calling an Authentik
// forward-auth outpost, and Squirrel compared one header to one configured
// string. The outpost could only ever say "somebody Authentik likes"; it could
// not say which somebody in a way Squirrel could act on, which is why a second
// person meant a redeploy.
//
// Everything about the protocol is go-oidc's: discovery, the JWKS, the
// signature, the issuer, the audience and the expiry. What is written here is
// only the part that is Squirrel's decision — whether this account is allowed
// in at all.

// ErrNotAllowed is an account Authentik authenticated and Squirrel will not
// admit. It is a different thing from a login that failed, and the door says
// something different about it.
var ErrNotAllowed = errors.New("that account cannot use Squirrel")

// Authentik is what the door needs to be built.
type Authentik struct {
	Issuer        string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	RequiredGroup string
}

// ErrNotReady is a gate that has not found authentik yet. It is a fact about
// this process rather than about the person pressing the button, which is why
// the screen says "I cannot reach the way in just now" and not anything about
// their account.
var ErrNotReady = errors.New("the way in has not been found yet")

// retryEvery is how often a gate that could not find authentik tries again.
//
// Not on every press: an authentik that is down would then cost a network
// round trip per button, and the answer would be the same one each time. Not
// once at boot either — see NewAuthentik.
const retryEvery = 30 * time.Second

// Gate is a configured way in and back.
//
// Not "door": a door in this product is a section of the pile — /pile,
// /chores, /kept — with its own art and its own heading. Reusing the word for
// the thing you sign in through would make every mention of it ambiguous.
type Gate struct {
	a Authentik

	mu       sync.Mutex
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	found    bool
	retryAt  time.Time
	group    string
}

// Person is who Authentik says just signed in.
type Person struct {
	// Sub is the OIDC subject: the only thing about an account that does not
	// change. Everything in the pile hangs off it.
	Sub string
	// Handle is a display name and nothing more. Two accounts may share one.
	Handle string
}

// NewAuthentik builds the gate. It makes no network call.
//
// Discovery used to happen here, and a provider that could not be reached was
// a boot that failed — "a Squirrel with no way in is not a working Squirrel".
// That reasoning was wrong and it took both clusters down on 25 August 2026:
// squirrel's pod had no egress to authentik's hostname, discovery failed, and
// the process refused to start. What went down with it was capture, the drain
// and the Campfire webhook, none of which have anything to do with the screen —
// and the room does not retry a delivery it could not make.
//
// This product's rule is that a failure costs a feature and never the product.
// The spool exists so that Postgres being unreachable does not lose a note. An
// identity provider the *screen* needs must not be a harder dependency than the
// database the *whole product* needs.
//
// So the split is by what the failure means:
//
//   - Configuration that is missing or dangerous is refused here, synchronously
//     and without a network. It cannot come right on its own.
//   - authentik being unreachable is not refused at all. The gate says so — the
//     screen already has a state for it — and tries again.
func NewAuthentik(_ context.Context, a Authentik) (*Gate, error) {
	// Refused rather than defaulted. Every other missing value in this product
	// degrades to less product — no coach, no camera, no push. An empty
	// required group would degrade to more access, which is the one direction
	// a default must never go.
	if a.RequiredGroup == "" {
		return nil, errors.New("refusing to build the gate: no required group")
	}
	return &Gate{a: a, group: a.RequiredGroup}, nil
}

// find is authentik's discovery document, fetched once and remembered.
//
// Retried rather than fatal, and not on every press: a down authentik would
// otherwise cost a round trip per button to give the same answer.
func (d *Gate) find(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.found {
		return nil
	}
	if now().Before(d.retryAt) {
		return ErrNotReady
	}
	d.retryAt = now().Add(retryEvery)

	provider, err := oidc.NewProvider(ctx, d.a.Issuer)
	if err != nil {
		slog.Warn("could not find the way in; the screen will say so and try again",
			"issuer", d.a.Issuer, "error", err)
		return fmt.Errorf("%w: %v", ErrNotReady, err)
	}
	d.oauth = oauth2.Config{
		ClientID:     d.a.ClientID,
		ClientSecret: d.a.ClientSecret,
		RedirectURL:  d.a.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	d.verifier = provider.Verifier(&oidc.Config{ClientID: d.a.ClientID})
	d.found = true
	slog.Info("the way in is open", "issuer", d.a.Issuer)
	return nil
}

// forget drops what was discovered, so a test can prove the lookup is not
// once-and-for-all.
func (d *Gate) forget() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.found, d.retryAt = false, time.Time{}
}

// Away is where to send somebody who is signing in.
//
// The verifier never leaves this machine — only its hash does, as the PKCE
// challenge — so a code intercepted on the way back cannot be spent by
// whoever intercepted it.
func (d *Gate) Away(state, verifier string) (string, error) {
	if err := d.find(context.Background()); err != nil {
		return "", err
	}
	d.mu.Lock()
	oauth := d.oauth
	d.mu.Unlock()

	sum := sha256.Sum256([]byte(verifier))
	return oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(sum[:])),
		oauth2.SetAuthURLParam("code_challenge_method", "S256")), nil
}

// Back turns a code into a person, or refuses.
//
// The two refusals are deliberately different errors. A token that does not
// verify is something wrong — a forgery, a clock, a rotated key — and says so.
// ErrNotAllowed is nothing wrong at all: it is Authentik doing its job for an
// account that is simply not for this product.
func (d *Gate) Back(ctx context.Context, code, verifier string) (Person, error) {
	if err := d.find(ctx); err != nil {
		return Person{}, err
	}
	d.mu.Lock()
	oauth, verify := d.oauth, d.verifier
	d.mu.Unlock()

	token, err := oauth.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return Person{}, fmt.Errorf("exchanging the code: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		// An access token without an ID token is an OAuth answer to an OIDC
		// question. There is nobody in it.
		return Person{}, errors.New("the gate answered without an id token")
	}
	id, err := verify.Verify(ctx, raw)
	if err != nil {
		return Person{}, fmt.Errorf("checking the id token: %w", err)
	}

	var said struct {
		Handle string   `json:"preferred_username"`
		Groups []string `json:"groups"`
	}
	if err := id.Claims(&said); err != nil {
		return Person{}, fmt.Errorf("reading the id token: %w", err)
	}
	// Checked here as well as bound in Authentik. The binding is the gate and
	// this is not a second gate — it is the check that a misconfigured gate
	// does not silently hand out piles. A claim that is absent is refused, not
	// waved through.
	if !allowed(said.Groups, d.group) {
		return Person{}, ErrNotAllowed
	}
	return Person{Sub: id.Subject, Handle: said.Handle}, nil
}

func allowed(groups []string, want string) bool {
	for _, g := range groups {
		if g == want {
			return true
		}
	}
	return false
}
