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

func aDoor(t *testing.T, idp *fakeIdP, group string) *Door {
	t.Helper()
	d, err := NewAuthentik(context.Background(), Authentik{
		Issuer: idp.URL, ClientID: "squirrel", ClientSecret: "shh",
		RedirectURL: "https://squirrel.example/auth/callback", RequiredGroup: group,
	})
	require.NoError(t, err)
	return d
}

func TestAGoodLoginGivesBackWhoItWas(t *testing.T) {
	d := aDoor(t, anIdP(t), "squirrel-users")

	who, err := d.Back(context.Background(), "a-code", "a-verifier")
	require.NoError(t, err)
	require.Equal(t, "sub-123", who.Sub)
	require.Equal(t, "ronald", who.Handle)
}

// The group is the gate, and it is checked here as well as bound in Authentik.
// A misconfigured binding would otherwise hand out piles silently.
func TestAnAccountWithoutTheGroupIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["groups"] = []any{"somebody-elses-app"}
	d := aDoor(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier")
	require.ErrorIs(t, err, ErrNotAllowed)
}

// No groups at all is refused too, rather than treated as unrestricted. This
// is the one place in the product where a missing value would mean more access
// rather than less.
func TestAnAccountWithNoGroupsIsRefused(t *testing.T) {
	idp := anIdP(t)
	delete(idp.claims, "groups")
	d := aDoor(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier")
	require.ErrorIs(t, err, ErrNotAllowed)
}

// A token signed by somebody else is not a login. This is the check delegated
// to go-oidc, and it is worth proving it is actually wired rather than
// assuming the library was configured correctly.
func TestATokenSignedByTheWrongKeyIsRefused(t *testing.T) {
	idp := anIdP(t)
	d := aDoor(t, idp, "squirrel-users")

	other, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp.signWith = other // the JWKS still publishes the real public key

	_, err = d.Back(context.Background(), "a-code", "a-verifier")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed, "a forged token was refused as a group problem")
}

// An expired token is not a login either.
func TestAnExpiredTokenIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["exp"] = time.Now().Add(-time.Hour).Unix()
	d := aDoor(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed)
}

// A token minted for another application is not a login here.
func TestATokenForAnotherAudienceIsRefused(t *testing.T) {
	idp := anIdP(t)
	idp.claims["aud"] = "somebody-elses-app"
	d := aDoor(t, idp, "squirrel-users")

	_, err := d.Back(context.Background(), "a-code", "a-verifier")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotAllowed)
}

// The way out carries state and a PKCE challenge, or the callback cannot tell
// its own redirect from somebody else's.
func TestTheWayOutCarriesStateAndAChallenge(t *testing.T) {
	d := aDoor(t, anIdP(t), "squirrel-users")

	away, err := url.Parse(d.Away("the-state", "the-verifier"))
	require.NoError(t, err)
	require.Equal(t, "the-state", away.Query().Get("state"))
	require.Equal(t, "S256", away.Query().Get("code_challenge_method"))
	require.NotEmpty(t, away.Query().Get("code_challenge"))
	require.NotContains(t, away.RawQuery, "the-verifier", "the verifier went out in the URL")
}

// The challenge is the hash of the verifier, not the verifier renamed.
func TestTheChallengeIsTheHashOfTheVerifier(t *testing.T) {
	d := aDoor(t, anIdP(t), "squirrel-users")

	away, err := url.Parse(d.Away("s", "the-verifier"))
	require.NoError(t, err)

	sum := sha256.Sum256([]byte("the-verifier"))
	require.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]),
		away.Query().Get("code_challenge"))
}

// A door with no required group is refused at construction. Everything else
// missing degrades to less product; this would degrade to more access.
func TestADoorWithNoRequiredGroupIsRefused(t *testing.T) {
	idp := anIdP(t)
	_, err := NewAuthentik(context.Background(), Authentik{
		Issuer: idp.URL, ClientID: "squirrel", ClientSecret: "shh",
		RedirectURL: "https://squirrel.example/auth/callback",
	})
	require.Error(t, err, "a door was built that would let anybody in")
}
