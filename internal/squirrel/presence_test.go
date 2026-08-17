package squirrel_test

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// presenceServer mounts the arrival route on a live listener and returns its
// base URL. `writable` and `listen` are the package's existing test helpers
// (internal/squirrel/http_test.go) for standing up a *Server and serving real
// requests over a socket — there is no exported way to hand a handler a
// request in-process, so this reuses what is already there rather than
// growing one.
func presenceServer(t *testing.T, o squirrel.PresenceOptions) string {
	t.Helper()
	s := squirrel.NewServer(writable(true))
	squirrel.MountPresence(s, "/hooks/home", o)
	return listen(t, s)
}

func postPresence(t *testing.T, base, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+"/hooks/home", nil)
	require.NoError(t, err)
	req.Header.Set("X-Squirrel-Token", token)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func TestPresenceCallsBackOnArrival(t *testing.T) {
	var arrived atomic.Int32
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", OnArrive: func() { arrived.Add(1) },
	})

	res := postPresence(t, base, "shh")
	defer res.Body.Close()

	require.Equal(t, http.StatusNoContent, res.StatusCode)
	require.EqualValues(t, 1, arrived.Load())
}

// The only authentication this endpoint has.
func TestPresenceRefusesAWrongSecret(t *testing.T) {
	var arrived atomic.Int32
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", OnArrive: func() { arrived.Add(1) },
	})

	res := postPresence(t, base, "wrong")
	defer res.Body.Close()

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	require.Zero(t, arrived.Load())
}

// Phones flap between wifi and cellular and Home Assistant fires on each
// flap. Each postPresence call blocks until the response returns, and the
// handler only writes its (bodyless, unflushed) response after OnArrive has
// run — so by the time postPresence returns, the callback for that request
// has already been observed or skipped. That ordering is what lets `now` be
// mutated between calls here without a data race.
func TestPresenceDebouncesRepeatArrivals(t *testing.T) {
	var arrived atomic.Int32
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", Debounce: 10 * time.Minute,
		OnArrive: func() { arrived.Add(1) },
		Now:      func() time.Time { return now },
	})

	for range 3 {
		res := postPresence(t, base, "shh")
		res.Body.Close()
	}
	require.EqualValues(t, 1, arrived.Load())

	now = now.Add(11 * time.Minute)
	res := postPresence(t, base, "shh")
	res.Body.Close()
	require.EqualValues(t, 2, arrived.Load(), "past the window it counts again")
}
