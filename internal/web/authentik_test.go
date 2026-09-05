package web

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A fake Authentik.
//
// It signs real ID tokens with a real key and serves a real JWKS, so go-oidc
// runs the verification it will run in production. Stubbing that out would
// leave the one part of this that is delegated to a library untested, and it
// is the part that decides whether a token is a login.
type fakeIdP struct {
	*httptest.Server
	key    *rsa.PrivateKey
	claims map[string]any
	// signWith, when set, signs with a different key than the JWKS publishes.
	signWith *rsa.PrivateKey
	// discoveries counts how often it was asked who it is, so a test can prove
	// the answer is remembered rather than fetched per press.
	discoveries int
	// refuse makes discovery fail, which is an authentik that is down.
	refuse bool
}

func anIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIdP{key: key, claims: map[string]any{
		"sub": "sub-123", "preferred_username": "ronald",
		"groups": []any{"squirrel-users"},
	}}
	mux := http.NewServeMux()
	idp.Server = httptest.NewServer(mux)
	t.Cleanup(idp.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		idp.discoveries++
		if idp.refuse {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                idp.URL,
			"authorization_endpoint":                idp.URL + "/authorize",
			"token_endpoint":                        idp.URL + "/token",
			"jwks_uri":                              idp.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "one",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at", "token_type": "Bearer",
			"id_token": idp.sign(t),
		})
	})
	return idp
}

// sign builds an RS256 ID token from the fake's current claims.
func (f *fakeIdP) sign(t *testing.T) string {
	t.Helper()
	claims := map[string]any{
		"iss": f.URL, "aud": "squirrel",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	for k, v := range f.claims {
		claims[k] = v
	}

	part := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	signing := part(map[string]any{"alg": "RS256", "typ": "JWT", "kid": "one"}) + "." + part(claims)

	key := f.key
	if f.signWith != nil {
		key = f.signWith
	}
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	require.NoError(t, err)
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func aGate(t *testing.T, idp *fakeIdP, group string) *Gate {
	t.Helper()
	d, err := NewAuthentik(context.Background(), Authentik{
		Issuer: idp.URL, ClientID: "squirrel", ClientSecret: "shh",
		RedirectURL: "https://squirrel.example/auth/callback", RequiredGroup: group,
	})
	require.NoError(t, err)
	return d
}

func TestAGoodLoginGivesBackWhoItWas(t *testing.T) {
	d := aGate(t, anIdP(t), "squirrel-users")

	who, err := d.Back(context.Background(), "a-code", "a-verifier", "")
	require.NoError(t, err)
	require.Equal(t, "sub-123", who.Sub)
	require.Equal(t, "ronald", who.Handle)
}

// The group is the gate, and it is checked here as well as bound in Authentik.
// A misconfigured binding would otherwise hand out piles silently.
func TestAnAccountWithoutTheGroupIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["groups"] = []any{"somebody-elses-app"}
	d := aGate(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier", "")
	require.ErrorIs(t, err, ErrNotAllowed)
}

// No groups at all is refused too, rather than treated as unrestricted. This
// is the one place in the product where a missing value would mean more access
// rather than less.
func TestAnAccountWithNoGroupsIsRefused(t *testing.T) {
	idp := anIdP(t)
	delete(idp.claims, "groups")
	d := aGate(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier", "")
	require.ErrorIs(t, err, ErrNotAllowed)
}

// A token signed by somebody else is not a login. This is the check delegated
// to go-oidc, and it is worth proving it is actually wired rather than
// assuming the library was configured correctly.
func TestATokenSignedByTheWrongKeyIsRefused(t *testing.T) {
	idp := anIdP(t)
	d := aGate(t, idp, "squirrel-users")

	other, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp.signWith = other // the JWKS still publishes the real public key

	_, err = d.Back(context.Background(), "a-code", "a-verifier", "")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed, "a forged token was refused as a group problem")
}

func TestAnExpiredTokenIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["exp"] = time.Now().Add(-time.Hour).Unix()
	d := aGate(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier", "")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed)
}

// A token minted for another application is not a login here.
func TestATokenForAnotherAudienceIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["aud"] = "somebody-elses-app"
	d := aGate(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier", "")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed)
}

// The way out carries the nonce too, so the callback can check the token it
// gets back was minted for this trip and not replayed from another.
func TestTheWayOutCarriesANonce(t *testing.T) {
	d := aGate(t, anIdP(t), "squirrel-users")

	raw, err := d.Away("the-state", "the-nonce", "the-verifier")
	require.NoError(t, err)
	away, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "the-nonce", away.Query().Get("nonce"))
}

// A token whose nonce does not match the one this trip sent is not a login —
// it was minted for a different round trip, replayed or forged.
func TestATokenWithTheWrongNonceIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["nonce"] = "the-real-nonce"
	d := aGate(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier", "somebody-elses-nonce")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed, "a nonce mismatch was refused as a group problem")
}

// And the matching nonce is a good login, same as no nonce ever was before
// this existed.
func TestATokenWithTheMatchingNonceSignsIn(t *testing.T) {
	idp := anIdP(t)
	idp.claims["nonce"] = "the-real-nonce"
	d := aGate(t, idp, "squirrel-users")

	who, err := d.Back(context.Background(), "a-code", "a-verifier", "the-real-nonce")
	require.NoError(t, err)
	require.Equal(t, "sub-123", who.Sub)
}

// The way out carries state and a PKCE challenge, or the callback cannot tell
// its own redirect from somebody else's.
func TestTheWayOutCarriesStateAndAChallenge(t *testing.T) {
	d := aGate(t, anIdP(t), "squirrel-users")

	raw, err := d.Away("the-state", "", "the-verifier")
	require.NoError(t, err)
	away, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "the-state", away.Query().Get("state"))
	require.Equal(t, "S256", away.Query().Get("code_challenge_method"))
	require.NotEmpty(t, away.Query().Get("code_challenge"))
	require.NotContains(t, away.RawQuery, "the-verifier", "the verifier went out in the URL")
}

func TestTheChallengeIsTheHashOfTheVerifier(t *testing.T) {
	d := aGate(t, anIdP(t), "squirrel-users")

	raw, err := d.Away("s", "", "the-verifier")
	require.NoError(t, err)
	away, err := url.Parse(raw)
	require.NoError(t, err)

	sum := sha256.Sum256([]byte("the-verifier"))
	require.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]),
		away.Query().Get("code_challenge"))
}

// A door with no required group is refused at construction. Everything else
// missing degrades to less product; this would degrade to more access.
func TestAGateWithNoRequiredGroupIsRefused(t *testing.T) {
	idp := anIdP(t)
	_, err := NewAuthentik(context.Background(), Authentik{
		Issuer: idp.URL, ClientID: "squirrel", ClientSecret: "shh",
		RedirectURL: "https://squirrel.example/auth/callback",
	})
	require.Error(t, err, "a gate was built that would let anybody in")
}

// This is the test that would have prevented the outage on 25 August 2026.
// Discovery ran at boot and a failure was a boot that failed — which took down
// capture, the drain and the Campfire webhook, none of which have anything to
// do with the screen. Squirrel's rule is that a failure costs a feature and not
// the product; the spool exists so a Postgres outage does not lose a note. An
// identity provider the screen needs must not be a harder dependency than the
// database the whole product needs.
func TestAGateBuildsWithoutReachingAuthentik(t *testing.T) {
	g, err := NewAuthentik(context.Background(), Authentik{
		Issuer:   "https://nothing.invalid/application/o/squirrel/",
		ClientID: "squirrel", ClientSecret: "shh",
		RedirectURL:   "https://squirrel.example/auth/callback",
		RequiredGroup: "squirrel-users",
	})
	require.NoError(t, err, "an unreachable authentik stopped the whole product from booting")
	require.NotNil(t, g)
}

// It refuses to act until it has, rather than pretending.
func TestAGateThatHasNotFoundAuthentikRefuses(t *testing.T) {
	g := unreachableGate(t)

	_, err := g.Away("the-state", "", "the-verifier")
	require.Error(t, err)

	_, err = g.Back(context.Background(), "a-code", "a-verifier", "")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed, "a gate that is not ready blamed the account")
}

// And it starts working once authentik answers, without a restart. An outage
// during boot must not need a human to notice it ended.
func TestAGateFindsAuthentikLater(t *testing.T) {
	idp := anIdP(t)
	g := aGate(t, idp, "squirrel-users")
	// aGate builds against a live idp; take discovery away and put it back to
	// prove the lookup is not once-and-for-all.
	g.forget()

	who, err := g.Back(context.Background(), "a-code", "a-verifier", "")
	require.NoError(t, err, "a gate could not find authentik a second time")
	require.Equal(t, "sub-123", who.Sub)
}

// Discovery is not repeated on every press. It is a network call, and the
// request path in this product does not make one it can avoid.
func TestAuthentikIsFoundOnceAndRemembered(t *testing.T) {
	idp := anIdP(t)
	g := aGate(t, idp, "squirrel-users")
	require.Zero(t, idp.discoveries, "building the gate went to the network")

	for range 3 {
		_, err := g.Away("s", "n", "v")
		require.NoError(t, err)
	}
	require.Equal(t, 1, idp.discoveries, "it asked authentik who it was on every press")
}

// A failed lookup is retried, but not on every request: an authentik that is
// down would otherwise take a network round trip per press to say so.
func TestAFailedLookupIsNotRetriedOnEveryPress(t *testing.T) {
	// The clock, moved rather than the gate's own field, so the backoff is
	// what is under test rather than a hole poked in it.
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	was := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = was })

	idp := anIdP(t)
	idp.refuse = true
	g := aGate(t, idp, "squirrel-users")

	for range 3 {
		_, err := g.Away("s", "n", "v")
		require.Error(t, err)
	}
	require.Equal(t, 1, idp.discoveries, "a down authentik cost a round trip per press")

	// Past the backoff it tries again, and by then authentik is up.
	at = at.Add(retryEvery + time.Second)
	idp.refuse = false
	_, err := g.Away("s", "n", "v")
	require.NoError(t, err, "it never tried again")
	require.Equal(t, 2, idp.discoveries)
}

func unreachableGate(t *testing.T) *Gate {
	t.Helper()
	g, err := NewAuthentik(context.Background(), Authentik{
		Issuer:   "https://nothing.invalid/application/o/squirrel/",
		ClientID: "squirrel", ClientSecret: "shh",
		RedirectURL:   "https://squirrel.example/auth/callback",
		RequiredGroup: "squirrel-users",
	})
	require.NoError(t, err)
	return g
}
