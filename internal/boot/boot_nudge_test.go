//go:build integration

package boot_test

import (
	"context"
	"log/slog"
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
//
// Checking the count once, immediately after the 204, proves nothing: the
// drain is async (bootWithStore's DRAIN_INTERVAL_MS is 10ms — see envFor), so
// a genuine capture counted the same way also reads before=0 after=0, since
// its row has not landed yet. require.Never polls for about a second instead,
// which is long enough for several drain ticks to have run if an arrival were
// ever mistakenly spooled.
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

	require.Never(t, func() bool {
		var after int
		if err := store.Pool().QueryRow(context.Background(),
			`select count(*) from items where person_id = $1`, p).Scan(&after); err != nil {
			return false
		}
		return after != before
	}, time.Second, 20*time.Millisecond, "an arrival must never produce an items row")
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

// MountPresence's own "refusing to mount" log only runs when it is actually
// called — skipping the call entirely, as boot.go does when PRESENCE_SECRET
// is empty, left that log line silent. A mis-wired secretKeyRef then produced
// a bot with no arrival trigger, no log anywhere saying so, and behaviour
// that looks entirely normal, since the evening message still runs on
// schedule. boot.go:259's "no sender configured; ..." warning is the existing
// precedent for saying this out loud.
func TestBootWarnsWhenNoPresenceSecretIsConfigured(t *testing.T) {
	withStore(t)

	var logs safeLog
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	boots(t, envFor(t, map[string]string{"CAMPFIRE_CONVERSATION_ID": "9"}))

	require.Contains(t, logs.String(), "no presence secret configured")
}
