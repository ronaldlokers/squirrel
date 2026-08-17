//go:build integration

package boot_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// End to end over a real socket: an arrival ping produces a nudge in the room.
func TestBootNudgesOnArrival(t *testing.T) {
	s, store := bootWithStore(t)

	p := ownerOf(t, store)
	seedOverdueChore(t, store, p, "vacuum")

	req, err := http.NewRequest(http.MethodPost, presenceURL(s), strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("X-Squirrel-Token", testPresenceSecret)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()

	require.Eventually(t, func() bool {
		return campfireStubSawText(t, "vacuum")
	}, 15*time.Second, 200*time.Millisecond, "the arrival produced a nudge")
}

// A presence ping is not a thought. It must leave no items row — phase 3 spent
// a fix removing button taps from the capture list and this would be the same
// mistake wearing a different hat.
func TestAnArrivalIsNotACapture(t *testing.T) {
	s, store := bootWithStore(t)
	p := ownerOf(t, store)

	var before int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from items where person_id = $1`, p).Scan(&before))

	req, err := http.NewRequest(http.MethodPost, presenceURL(s), strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("X-Squirrel-Token", testPresenceSecret)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()

	var after int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from items where person_id = $1`, p).Scan(&after))
	require.Equal(t, before, after, "an arrival is not a capture")
}

// The route is not mounted without a secret, the same way Send is nil without
// a bot key.
func TestBootWithoutAPresenceSecretHasNoRoute(t *testing.T) {
	s := bootWithoutPresence(t)

	res, err := http.Post(presenceURL(s), "application/json", strings.NewReader(""))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}
