//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// started puts a timer in the past so a test can be about an hour ago without
// waiting an hour.
func started(t *testing.T, store *squirrel.Store, p int64, ago time.Duration, ran time.Duration, ramp bool) {
	t.Helper()
	ctx := context.Background()
	_, err := store.StartTimer(ctx, p, "the tax return", ran, time.Now().Add(-ago))
	require.NoError(t, err)
	if ramp {
		require.NoError(t, store.ArmRamp(ctx, p, true))
	}
}

// A timer half an hour past its end, opted in, speaks.
func TestATimerLongPastItsEndSpeaks(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	// Started 100 minutes ago, ran for 25: it ended 75 minutes ago.
	started(t, store, p, 100*time.Minute, 25*time.Minute, true)

	got, found, err := store.RampDue(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the tax return", got.Label)
}

// One that has only just run out does not.
func TestATimerJustPastItsEndStaysQuiet(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	started(t, store, p, 30*time.Minute, 25*time.Minute, true)

	_, found, err := store.RampDue(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "it spoke five minutes after the timer ended")
}

// **Opt-in.** A timer nobody ticked the box for never speaks, however long it
// has been. The chat's !timer, the coach's own hand and the nudge all start
// timers, and none of them can have been opted in on.
func TestATimerNobodyOptedInOnNeverSpeaks(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	started(t, store, p, 5*time.Hour, 25*time.Minute, false)

	_, found, err := store.RampDue(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "it interrupted a timer nobody asked it to watch")
}

// It says it once.
func TestItSpeaksOnce(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	started(t, store, p, 100*time.Minute, 25*time.Minute, true)

	_, found, err := store.RampDue(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	require.NoError(t, store.RampSaid(ctx, p, time.Now()))

	_, found, err = store.RampDue(ctx, p, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.False(t, found, "it said it twice about one timer")
}

// A new timer is a new decision, so it may speak again — but only if the box
// was ticked again.
func TestANewTimerMaySpeakAgain(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	started(t, store, p, 100*time.Minute, 25*time.Minute, true)
	require.NoError(t, store.RampSaid(ctx, p, time.Now()))

	started(t, store, p, 100*time.Minute, 25*time.Minute, true)
	_, found, err := store.RampDue(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found, "a fresh timer, opted in again, stayed silent")

	// And a fresh one without the box stays silent.
	started(t, store, p, 100*time.Minute, 25*time.Minute, false)
	_, found, err = store.RampDue(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found)
}

// "leave me alone" means today, not this timer. It survives a new timer, which
// is the whole difference between it and having already spoken.
func TestLeaveMeAloneSurvivesANewTimer(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	started(t, store, p, 100*time.Minute, 25*time.Minute, true)

	require.NoError(t, store.HushRamp(ctx, p, time.Now()))

	started(t, store, p, 100*time.Minute, 25*time.Minute, true)
	_, found, err := store.RampDue(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "leave me alone lasted only as long as that timer")
}

// And it is today rather than a rolling day: somebody who says it at four in
// the afternoon is talking about this afternoon.
func TestLeaveMeAloneEndsWithTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	started(t, store, p, 100*time.Minute, 25*time.Minute, true)
	require.NoError(t, store.HushRamp(ctx, p, time.Now()))

	started(t, store, p, 100*time.Minute, 25*time.Minute, true)
	_, found, err := store.RampDue(ctx, p, time.Now().AddDate(0, 0, 1))
	require.NoError(t, err)
	require.True(t, found, "leave me alone silenced tomorrow as well")
}

// A stopped timer says nothing. Stopping is the answer, not a thing to be
// asked about afterwards.
func TestAStoppedTimerSaysNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	started(t, store, p, 100*time.Minute, 25*time.Minute, true)
	require.NoError(t, store.StopTimer(ctx, p))

	_, found, err := store.RampDue(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "it asked about a timer that was already stopped")
}

// Somebody else's timer is not yours.
func TestARampBelongsToOnePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-theirs", "theirs")
	require.NoError(t, err)
	started(t, store, theirs, 100*time.Minute, 25*time.Minute, true)

	_, found, err := store.RampDue(ctx, mine, time.Now())
	require.NoError(t, err)
	require.False(t, found, "somebody else's timer interrupted me")
}
