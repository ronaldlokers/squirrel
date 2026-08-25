//go:build integration

package boot_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/boot"
)

// anIssuer is enough of an Authentik for boot to discover one.
//
// Only the discovery document: nothing here walks the flow, and the flow
// itself is proved in internal/web against a fake that signs real ID tokens
// with a real key. What this exists to prove is that the screen boots behind
// a session rather than behind a header.
func anIssuer(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                server.URL,
			"authorization_endpoint":                server.URL + "/authorize",
			"token_endpoint":                        server.URL + "/token",
			"jwks_uri":                              server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	return server.URL
}

// withAWayIn is the environment of a Squirrel that has one.
func withAWayIn(t *testing.T, extra map[string]string) map[string]string {
	t.Helper()
	env := map[string]string{
		"WEB_IDENTITY":           "ronald",
		"WEB_REQUIRED_GROUP":     "squirrel-users",
		"WEB_OIDC_ISSUER":        anIssuer(t),
		"WEB_OIDC_CLIENT_ID":     "squirrel",
		"WEB_OIDC_CLIENT_SECRET": "shh",
		"WEB_OIDC_REDIRECT_URL":  "https://squirrel.example/auth/callback",
	}
	for k, v := range extra {
		env[k] = v
	}
	return envFor(t, env)
}

// get is a plain GET, optionally carrying a cookie, and it does not follow
// redirects: what a request with nobody behind it is *told* is the thing being
// tested.
func get(t *testing.T, url string, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	res, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// The screen is behind a session, and a header is not one.
//
// This test used to assert the opposite half of the same fact: that
// X-Authentik-Username was the whole authentication. It is the end-to-end
// proof that the middleware coming off the ingress does not leave the pile
// open, so it has to keep failing if the header ever starts working again.
func TestTheScreenIsBehindASession(t *testing.T) {
	withStore(t)
	s := boots(t, withAWayIn(t, nil))
	url := screenURL(s)

	nobody := get(t, url)
	require.Equal(t, http.StatusSeeOther, nobody.StatusCode)
	require.Equal(t, "/auth", nobody.Header.Get("Location"))

	withTheOldHeader, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	withTheOldHeader.Header.Set("X-Authentik-Username", "ronald")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	res, err := client.Do(withTheOldHeader)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusSeeOther, res.StatusCode,
		"the forward-auth header still signs somebody in")

	// A made-up cookie is nobody too.
	require.Equal(t, http.StatusSeeOther,
		get(t, url, &http.Cookie{Name: "squirrel_session", Value: "made up"}).StatusCode)
}

// And the way in is reachable with nothing at all, or there is no way in.
func TestTheWayInIsReachableWithNoSession(t *testing.T) {
	withStore(t)
	s := boots(t, withAWayIn(t, nil))

	res := get(t, screenURL(s)+"auth")
	require.Equal(t, http.StatusOK, res.StatusCode)
}

// A screen that is asked for and cannot be signed into is not a screen. Boot
// refuses rather than mounting one, because the alternative is a deploy that
// looks healthy and locks you out of your own pile.
func TestBootRefusesAScreenWithNoWayIn(t *testing.T) {
	withStore(t)
	_, err := boot.Boot(context.Background(), envFor(t, map[string]string{
		"WEB_IDENTITY": "ronald",
	}))
	require.Error(t, err, "the screen mounted with no way to sign into it")
}

// And one with a way in but nothing deciding who may use it.
func TestBootRefusesAScreenWithNoRequiredGroup(t *testing.T) {
	withStore(t)
	_, err := boot.Boot(context.Background(), withAWayIn(t, map[string]string{
		"WEB_REQUIRED_GROUP": "",
	}))
	require.Error(t, err, "the pile mounted with nothing deciding who may read it")
}

func TestTheScreenIsNotMountedWithoutAnIdentity(t *testing.T) {
	withStore(t)
	s := boots(t, envFor(t, nil))

	require.Equal(t, http.StatusNotFound, get(t, screenURL(s)).StatusCode,
		"no identity, no route — not an open route that warns")
}
