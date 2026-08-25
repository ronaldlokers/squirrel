package web

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

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

// Gate is a configured way in and back.
//
// Not "door": a door in this product is a section of the pile — /pile,
// /chores, /kept — with its own art and its own heading. Reusing the word for
// the thing you sign in through would make every mention of it ambiguous.
type Gate struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
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

// NewAuthentik discovers the provider and builds the gate.
//
// Discovery is a network call, made once at boot rather than per request. A
// provider that cannot be reached at boot is a boot that fails, which is the
// honest outcome: a Squirrel with no way in is not a working Squirrel.
func NewAuthentik(ctx context.Context, a Authentik) (*Gate, error) {
	// Refused rather than defaulted. Every other missing value in this product
	// degrades to less product — no coach, no camera, no push. An empty
	// required group would degrade to more access, which is the one direction
	// a default must never go.
	if a.RequiredGroup == "" {
		return nil, errors.New("refusing to build the gate: no required group")
	}
	provider, err := oidc.NewProvider(ctx, a.Issuer)
	if err != nil {
		return nil, fmt.Errorf("finding the gate at %s: %w", a.Issuer, err)
	}
	return &Gate{
		oauth: oauth2.Config{
			ClientID:     a.ClientID,
			ClientSecret: a.ClientSecret,
			RedirectURL:  a.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: a.ClientID}),
		group:    a.RequiredGroup,
	}, nil
}

// Away is where to send somebody who is signing in.
//
// The verifier never leaves this machine — only its hash does, as the PKCE
// challenge — so a code intercepted on the way back cannot be spent by
// whoever intercepted it.
func (d *Gate) Away(state, verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return d.oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(sum[:])),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

// Back turns a code into a person, or refuses.
//
// The two refusals are deliberately different errors. A token that does not
// verify is something wrong — a forgery, a clock, a rotated key — and says so.
// ErrNotAllowed is nothing wrong at all: it is Authentik doing its job for an
// account that is simply not for this product.
func (d *Gate) Back(ctx context.Context, code, verifier string) (Person, error) {
	token, err := d.oauth.Exchange(ctx, code,
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
	id, err := d.verifier.Verify(ctx, raw)
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
