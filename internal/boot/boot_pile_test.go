//go:build integration

package boot_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// get is a plain GET with an optional forward-auth identity, so a test can ask
// the screen the same two questions Traefik's middleware decides between.
func get(t *testing.T, url, identity string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	if identity != "" {
		req.Header.Set("X-Authentik-Username", identity)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestTheScreenIsBehindTheIdentityHeader(t *testing.T) {
	withStore(t)
	s := boots(t, envFor(t, map[string]string{"WEB_IDENTITY": "ronald"}))
	url := pileURL(s)

	require.Equal(t, http.StatusForbidden, get(t, url, "").StatusCode)
	require.Equal(t, http.StatusForbidden, get(t, url, "someone").StatusCode)

	// 503 until connectAndDrain has reached Postgres and learned the owner —
	// the route is live from Listen, its memory is not. Eventually, not a
	// single call, for the same reason every other boot test polls.
	require.Eventually(t, func() bool {
		return get(t, url, "ronald").StatusCode == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond)
}

func TestTheScreenIsNotMountedWithoutAnIdentity(t *testing.T) {
	withStore(t)
	s := boots(t, envFor(t, nil))

	require.Equal(t, http.StatusNotFound, get(t, pileURL(s), "ronald").StatusCode,
		"no identity, no route — not an open route that warns")
}
