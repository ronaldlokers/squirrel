package squirrel_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// safeLog is a concurrency-safe io.Writer: OnArrive's callback and the
// handler itself can both log concurrently with the test reading the buffer
// back out (see waitForArrival — the callback runs off a goroutine).
type safeLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *safeLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *safeLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// captureLogs redirects slog's default handler to a safeLog for the
// duration of one test, restoring the previous default on cleanup — the
// same pattern boot_nudge_test.go uses for asserting on log output.
func captureLogs(t *testing.T) *safeLog {
	t.Helper()
	var logs safeLog
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &logs
}

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

// The owner's only evidence that a ping ever arrived was three SQL queries
// against production Postgres — kubectl logs showed nothing at all, on
// accept or on reject. This is the accepted half of the fix.
func TestPresenceLogsAcceptedPing(t *testing.T) {
	logs := captureLogs(t)
	arrived := make(chan struct{}, 1)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", OnArrive: func() { arrived <- struct{}{} },
	})

	res := postPresence(t, base, "shh")
	defer res.Body.Close()
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	waitForArrival(t, arrived)

	require.Contains(t, logs.String(), "presence: ping accepted")
	require.NotContains(t, logs.String(), "debounced",
		"an accepted ping must not read as a debounced one")
}

// A phone flapping between wifi and cellular produces several pings inside
// the debounce window. The owner needs to see that this is what happened —
// several accepted-looking requests that were actually one — not conclude
// the later ones were lost. Accepted and debounced must show up as two
// different lines, not the same line with a flag buried in it.
func TestPresenceLogsDebouncedPing(t *testing.T) {
	logs := captureLogs(t)
	arrived := make(chan struct{}, 2)
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", Debounce: 10 * time.Minute,
		OnArrive: func() { arrived <- struct{}{} },
		Now:      func() time.Time { return now },
	})

	first := postPresence(t, base, "shh")
	first.Body.Close()
	waitForArrival(t, arrived)

	second := postPresence(t, base, "shh")
	second.Body.Close()
	requireNoArrival(t, arrived)

	out := logs.String()
	require.Contains(t, out, "presence: ping accepted")
	require.Contains(t, out, "presence: ping debounced")
}

// The token check is the only authentication this route has, so a rejection
// is worth a WARN — but the value that failed the check must never reach the
// log store. Echoing a near-miss credential back into logs is the same
// mistake v0.2.0 made leaking the bot key through a *url.Error (see
// campfire.go's stripURL and campfire_send_test.go's
// TestSendFailureDoesNotLeakTheBotKey); this is that precedent applied here.
func TestPresenceRejectedTokenLogsWarnWithoutTheToken(t *testing.T) {
	logs := captureLogs(t)
	arrived := make(chan struct{}, 1)
	base := presenceServer(t, squirrel.PresenceOptions{
		Secret: "shh", OnArrive: func() { arrived <- struct{}{} },
	})

	const attemptedToken = "wrong-token-must-never-reach-the-log-store"
	res := postPresence(t, base, attemptedToken)
	defer res.Body.Close()
	require.Equal(t, http.StatusForbidden, res.StatusCode)
	requireNoArrival(t, arrived)

	out := logs.String()
	require.Contains(t, out, "presence: rejected")
	require.NotContains(t, out, attemptedToken)
}
