//go:build integration

package boot_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/boot"
)

// safeLog is a concurrency-safe io.Writer: the scheduler, the drain and the
// test itself can all reach slog's default handler at once.
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

// TestBootJoinsTheSchedulerBeforeClosingTheStore guards the shutdown race
// where Stop closes the store while the digest scheduler is still using it.
// Scheduler.Run calls Once synchronously the instant its goroutine starts, so
// a Stop that does not join that goroutine can close the pool out from under
// an in-flight query — the same failure mode Stop's own doc comment says the
// drain join exists to prevent, just for the scheduler instead. EVENING_AT is
// set to a time already past today, so Once's time-of-day guard never
// short-circuits before the store is touched, and each iteration posts a
// real capture so the digest has something to query and send rather than
// bailing out on an empty render.
//
// This is a genuine goroutine-scheduling race, not something a single
// Boot/Stop pair reproduces on demand, so this boots, captures, and stops
// several times over. An "ERROR digest ...context canceled" line is expected
// noise on some iterations either way: Stop cancels the context before it
// joins anything (see Stop's own doc comment), so a scheduler tick caught
// mid-flight at that moment fails with a context-cancellation-flavoured
// error — "reading chores: context canceled", "committing prompt: context
// canceled", and so on — whether or not the join below it is actually
// there. A bare require.NotContains(logs, "digest") could not tell that
// expected noise apart from the bug it exists to catch, and failed on it
// roughly a quarter of the time.
//
// What an unjoined Stop can produce that a correctly-joined one cannot is a
// query that is still actually running against a pool Close has already torn
// down, rather than one that merely observed its context being cancelled —
// the pgx/puddle error text for that specific case is "closed pool", so that
// is what is checked for instead. It is a narrow window even in the buggy
// case (puddle's own Close blocks until resources already checked out are
// returned, so most of the exposure is on the acquire side), which is
// exactly why this loops rather than trusting one iteration either way.
func TestBootJoinsTheSchedulerBeforeClosingTheStore(t *testing.T) {
	store := withStore(t)

	var logs safeLog
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	env := envFor(t, map[string]string{"EVENING_AT": "00:00"})

	// Checked after every iteration rather than once at the end: without the
	// join, one raced iteration tends to leave that iteration's store in a bad
	// state (an unclosed or half-closed pool), and every later iteration then
	// fails for a second, unrelated reason. Failing fast on the first bad
	// iteration keeps the failure attributable to the actual race.
	const iterations = 20
	for i := 0; i < iterations; i++ {
		s, err := boot.Boot(context.Background(), env)
		require.NoError(t, err, "iteration %d", i)

		body := strings.Replace(payload, `"id": 42`, fmt.Sprintf(`"id": %d`, 1000+i), 1)
		res, err := http.Post(
			fmt.Sprintf("http://127.0.0.1:%d/transports/campfire", s.Port()),
			"application/json", strings.NewReader(body))
		require.NoError(t, err, "iteration %d", i)
		res.Body.Close()

		require.Eventually(t, func() bool {
			var n int
			if err := store.Pool().QueryRow(context.Background(),
				`select count(*) from items where raw_text = 'buy milk'`).Scan(&n); err != nil {
				return false
			}
			return n >= 1
		}, 5*time.Second, 10*time.Millisecond, "iteration %d", i)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		stopErr := s.Stop(ctx)
		cancel()

		require.NotContains(t, logs.String(), "closed pool",
			"iteration %d: the scheduler used the store after Stop closed it — "+
				"Stop raced it instead of joining it before closing the store", i)
		require.NoError(t, stopErr, "iteration %d", i)
	}
}

// TestStopDoesNotBlockForThePresenceDelay guards the opposite failure from
// what this test used to assert. Before presence.go threaded a context
// through PresenceOptions (see presence.go's own Ctx doc comment and
// TestPresenceDelayWakesOnContextCancellation in the squirrel package), the
// spawned goroutine only ever woke on its own `time.Sleep(o.Delay)`, which
// selected on nothing — so Stop, which does join that goroutine via wg (the
// guarantee this test used to prove and still relies on), had no way to make
// it return early. A rollout landing within PRESENCE_DELAY of an arrival
// blocked Stop for up to two minutes in production, well past main's 15s
// shutdown budget and a default 30s grace period, so the pod was SIGKILLed
// rather than stopped cleanly.
//
// The fix is Stop's own s.cancel() reaching the same loopCtx boot.go now
// threads into PresenceOptions.Ctx, waking the goroutine immediately instead
// of making it sleep out the delay. This is directly measurable the same way
// the old version of this test measured the opposite: PRESENCE_DELAY is set
// well above any plausible Stop overhead, and Stop, called immediately after
// the arrival is acknowledged, must return in well under that delay rather
// than anywhere near it.
func TestStopDoesNotBlockForThePresenceDelay(t *testing.T) {
	withStore(t)

	const delay = 2 * time.Second
	s := boots(t, envFor(t, map[string]string{
		"PRESENCE_SECRET": testPresenceSecret,
		"PRESENCE_DELAY":  delay.String(),
	}))

	req, err := http.NewRequest(http.MethodPost, presenceURL(s), strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("X-Squirrel-Token", testPresenceSecret)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()

	// Immediately, not after any wait: the whole point is that Stop itself
	// must not be the thing waiting out the arrival's delay.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	start := time.Now()
	stopErr := s.Stop(ctx)
	elapsed := time.Since(start)
	cancel()

	require.NoError(t, stopErr)
	require.Less(t, elapsed, delay/2,
		"Stop waited out the arrival's full delay instead of waking it via context cancellation")
}
