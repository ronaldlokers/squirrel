//go:build integration

package squirrel_test

import (
	"context"
	"expvar"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func expvarInt(t *testing.T, name string) int64 {
	t.Helper()
	v := expvar.Get(name)
	require.NotNil(t, v, "no expvar published under %q", name)
	i, ok := v.(*expvar.Int)
	require.True(t, ok, "%q is not an *expvar.Int", name)
	return i.Value()
}

func TestDrainOnceReportsSpoolDepth(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)
	_, err = sp.Write(capture(func(c *squirrel.Capture) { c.ExternalID = squirrel.Ptr("43") }))
	require.NoError(t, err)

	squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second}).
		Once(context.Background())

	require.Equal(t, int64(2), expvarInt(t, "spool_depth"))
}

func TestDrainOnceCountsDeferredFiles(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	unreachable, err := squirrel.OpenStore(context.Background(),
		"postgres://nobody:nobody@127.0.0.1:1/squirrel")
	require.NoError(t, err)
	defer unreachable.Close()

	before := expvarInt(t, "drain_deferred_total")

	got := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: unreachable, Interval: time.Second,
	}).Once(context.Background())
	require.Equal(t, 1, got.Deferred)

	require.Equal(t, before+1, expvarInt(t, "drain_deferred_total"))

	squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second}).
		Once(context.Background())
}

func TestDrainRunReportsTheCurrentBackoff(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	flaky := &flakyStore{real: store, failTimes: 1}

	var mu sync.Mutex
	var seenAfterFirstWait int64
	waited := 0
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: flaky, Interval: 50 * time.Millisecond, MaxBackoff: time.Minute,
		OnWait: func(d time.Duration) {
			mu.Lock()
			defer mu.Unlock()
			waited++
			if waited == 1 {
				seenAfterFirstWait = expvarInt(t, "drain_backoff_ms")
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		drain.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return len(flaky.snapshot()) >= 2 && countItems(t, store) == 1
	}, 5*time.Second, 5*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, int64(100), seenAfterFirstWait, "the reported backoff was not the doubled value Run chose")
}
