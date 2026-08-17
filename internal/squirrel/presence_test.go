package squirrel_test

import (
	"context"
	"net/http"
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

// waitForArrival blocks until a signal arrives on ch or fails the test. Every
// OnArrive call now runs off a spawned goroutine (so a synchronous nudge
// callback can never hold the response open, and a panic in it can be
// recovered) — that makes its completion async relative to the HTTP
// response, so tests that want to observe it have to wait for it rather than
// assume it already happened.
func waitForArrival(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for OnArrive")
	}
}

// requireNoArrival asserts nothing is waiting on ch right now. It's only
// used where the absence is already guaranteed deterministically by the
// handler's synchronous logic (a rejected token, or a request the mutex-
// protected debounce check decided was within the window) — not as a
// "probably didn't happen yet" race.
func requireNoArrival(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("unexpected OnArrive")
	default:
	}
}

func TestPresenceCallsBackOnArrival(t *testing.T) {
	arrived := make(chan struct{}, 1)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", OnArrive: func() { arrived <- struct{}{} },
	})

	res := postPresence(t, base, "shh")
	defer res.Body.Close()

	require.Equal(t, http.StatusNoContent, res.StatusCode)
	waitForArrival(t, arrived)
}

// The only authentication this endpoint has.
func TestPresenceRefusesAWrongSecret(t *testing.T) {
	arrived := make(chan struct{}, 1)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", OnArrive: func() { arrived <- struct{}{} },
	})

	res := postPresence(t, base, "wrong")
	defer res.Body.Close()

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	requireNoArrival(t, arrived)
}

// An unset secret must not fall back to authenticating everything.
// subtle.ConstantTimeCompare("", "") returns 1, so without this guard a
// missing Secret would accept a request that carries no token header at
// all. MountPresence refuses to register the route in that case, so the
// path stays unrouted — a 404, same as any other unmounted path.
func TestPresenceRefusesToMountWithAnEmptySecret(t *testing.T) {
	base := presenceServer(t, squirrel.PresenceOptions{
		OnArrive: func() {},
	})

	req, err := http.NewRequest(http.MethodPost, base+"/hooks/home", nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusNotFound, res.StatusCode)
}

// Phones flap between wifi and cellular and Home Assistant fires on each
// flap. The within/last decision runs synchronously under a mutex inside the
// handler, before the response is written — so by the time each
// postPresence call returns, whether that request would trigger a callback
// is already settled, even though the callback itself (always run off a
// goroutine, see MountPresence) may still be in flight. That settledness is
// what lets `now` be mutated between calls here without a data race: each
// call's outcome can't change after its response has come back.
func TestPresenceDebouncesRepeatArrivals(t *testing.T) {
	arrived := make(chan struct{}, 4)
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", Debounce: 10 * time.Minute,
		OnArrive: func() { arrived <- struct{}{} },
		Now:      func() time.Time { return now },
	})

	for range 3 {
		res := postPresence(t, base, "shh")
		res.Body.Close()
	}
	waitForArrival(t, arrived)
	requireNoArrival(t, arrived)

	now = now.Add(11 * time.Minute)
	res := postPresence(t, base, "shh")
	res.Body.Close()
	waitForArrival(t, arrived)
}

// TestPresenceDelayWakesOnContextCancellation proves the fix for the shutdown
// hang: before Ctx existed, the goroutine's `time.Sleep(o.Delay)` selected on
// nothing, so a caller joining it (boot.go's wg, via the Go hook) had to wait
// out the full Delay — up to two minutes in production, since PRESENCE_DELAY
// wants "you have a coat on" to mean minutes — well past main's 15s shutdown
// budget and a default 30s grace period, so the pod would be SIGKILLed
// instead of stopping cleanly. Cancelling Ctx must wake an in-flight Delay
// immediately, and must skip OnArrive entirely rather than running it against
// a context a caller is already tearing things down for.
func TestPresenceDelayWakesOnContextCancellation(t *testing.T) {
	arrived := make(chan struct{}, 1)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh",
		Delay:  2 * time.Second,
		OnArrive: func() {
			arrived <- struct{}{}
		},
		Ctx: ctx,
		// A custom Go hook so the test can observe the goroutine actually
		// finishing, the same purpose boot.go's wg serves in production —
		// without this there is no way to distinguish "woke up promptly" from
		// "the default bare `go fn()` happens to still be running".
		Go: func(fn func()) {
			go func() {
				fn()
				close(done)
			}()
		},
	})

	res := postPresence(t, base, "shh")
	res.Body.Close()

	start := time.Now()
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("the presence goroutine did not wake on context cancellation")
	}
	require.Less(t, time.Since(start), time.Second,
		"cancelling Ctx must wake an in-flight Delay rather than waiting it out")
	requireNoArrival(t, arrived)
}
