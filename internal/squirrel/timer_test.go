//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestATimerRunsAndIsSaidOnce(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	reply := triage(t, store, p, "!timer 10 the kitchen")
	require.Contains(t, reply, "10 minutes on the kitchen")

	t1, found, err := store.CurrentTimer(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the kitchen", t1.Label)

	// Not up yet: nothing to claim, and nothing said.
	_, ready, err := store.ClaimFinishedTimer(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, ready)

	// Up. Claimed exactly once, and nothing is left behind — a finished timer
	// is not a thing this product keeps.
	done, ready, err := store.ClaimFinishedTimer(ctx, p, time.Now().Add(11*time.Minute))
	require.NoError(t, err)
	require.True(t, ready)
	require.Equal(t, "the kitchen", done.Label)

	_, again, err := store.ClaimFinishedTimer(ctx, p, time.Now().Add(11*time.Minute))
	require.NoError(t, err)
	require.False(t, again, "two overlapping ticks cannot both announce it")

	_, still, err := store.CurrentTimer(ctx, p)
	require.NoError(t, err)
	require.False(t, still, "nothing kept")
}

// The answer to "actually, twenty minutes" is to say twenty minutes.
func TestStartingASecondTimerReplacesTheFirst(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "!timer 10 the kitchen")
	triage(t, store, p, "!timer 20 the shed")

	t1, found, err := store.CurrentTimer(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the shed", t1.Label)
	require.InDelta(t, 20*time.Minute, t1.Ends.Sub(t1.Started), float64(time.Second))
}

// Stopping halfway is a normal ending, and stopping nothing is the state you
// asked for rather than an error.
func TestStoppingATimerAndStoppingNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "!timer 10 the kitchen")
	require.Contains(t, triage(t, store, p, "!stop"), "Stopped")

	_, found, err := store.CurrentTimer(ctx, p)
	require.NoError(t, err)
	require.False(t, found)

	require.Contains(t, triage(t, store, p, "!stop"), "Stopped")
}

// What it says at the end asks nothing. Whether the thing got done is not the
// timer's business, and "did you finish?" would make a body double into a
// supervisor.
func TestWhatATimerSaysAtTheEnd(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	triage(t, store, p, "!timer 5 the kitchen")
	done, ready, err := store.ClaimFinishedTimer(context.Background(), p, time.Now().Add(6*time.Minute))
	require.NoError(t, err)
	require.True(t, ready)

	text := squirrel.TimerUpMessage(done).Text
	require.Contains(t, text, "the kitchen")
	require.NotContains(t, text, "?")
	require.NotContains(t, text, "did you")
}
