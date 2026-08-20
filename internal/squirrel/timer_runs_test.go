//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// How long a thing takes, measured. What matters as much as what this records
// is what it refuses to: migration 0017 was right that a history of every
// timer is a record of what you do not finish, and this is narrower on purpose.

// ranFor starts a timer and lets it reach its end, which is the only way a run
// is ever recorded.
func ranFor(t *testing.T, store *squirrel.Store, personID int64, label string, minutes int, at time.Time) {
	t.Helper()
	ctx := context.Background()

	_, err := store.StartTimer(ctx, personID, label, time.Duration(minutes)*time.Minute, at)
	require.NoError(t, err)

	_, found, err := store.ClaimFinishedTimer(ctx, personID, at.Add(time.Duration(minutes)*time.Minute))
	require.NoError(t, err)
	require.True(t, found)
}

func TestTypicalIsTheMedianOfFinishedRuns(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i, minutes := range []int{5, 10, 30} {
		ranFor(t, store, p, "put the bins out", minutes, now.Add(time.Duration(i)*time.Hour))
	}

	got, ok, err := store.TypicalMinutes(ctx, p, "put the bins out")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 10, got, "the median, not the mean — 15 would be the mean")
}

// A median of one is not a median. It is the last time you did it, wearing a
// word that implies more.
func TestTooFewRunsSayNothing(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	ranFor(t, store, p, "put the bins out", 10, now)
	ranFor(t, store, p, "put the bins out", 10, now.Add(time.Hour))

	_, ok, err := store.TypicalMinutes(ctx, p, "put the bins out")
	require.NoError(t, err)
	require.False(t, ok)
}

// The narrowing that makes 0022 a different thing from what 0017 refused. A
// timer stopped early is not a measurement, and a table holding both would be
// a record of what you do not finish.
func TestAStoppedTimerIsNotRecorded(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i := 0; i < 5; i++ {
		_, err := store.StartTimer(ctx, p, "put the bins out", 10*time.Minute,
			now.Add(time.Duration(i)*time.Hour))
		require.NoError(t, err)
		require.NoError(t, store.StopTimer(ctx, p))
	}

	_, ok, err := store.TypicalMinutes(ctx, p, "put the bins out")
	require.NoError(t, err)
	require.False(t, ok, "stopping early was recorded as a measurement")
}

// Nor is one replaced by starting another, which is what abandoning looks like
// in this schema.
func TestAReplacedTimerIsNotRecorded(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i := 0; i < 5; i++ {
		_, err := store.StartTimer(ctx, p, "put the bins out", 10*time.Minute,
			now.Add(time.Duration(i)*time.Hour))
		require.NoError(t, err)
	}

	_, ok, err := store.TypicalMinutes(ctx, p, "put the bins out")
	require.NoError(t, err)
	require.False(t, ok)
}

// "the bins" and "The Bins " are the same thing to everyone except a string
// comparison.
func TestTheLabelIsMatchedTheWayAPersonWouldMatchIt(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	ranFor(t, store, p, "The Bins", 10, now)
	ranFor(t, store, p, "the bins ", 10, now.Add(time.Hour))
	ranFor(t, store, p, " the BINS", 10, now.Add(2*time.Hour))

	got, ok, err := store.TypicalMinutes(ctx, p, "the bins")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 10, got)
}

// Two things timed under different names are two things.
func TestOneLabelDoesNotAnswerForAnother(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i := 0; i < 3; i++ {
		ranFor(t, store, p, "put the bins out", 10, now.Add(time.Duration(i)*time.Hour))
	}

	_, ok, err := store.TypicalMinutes(ctx, p, "ring the vet")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestRunsAreNotSharedBetweenPeople(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	mine := owner(t, store)
	now := time.Now()

	for i := 0; i < 3; i++ {
		ranFor(t, store, mine, "put the bins out", 10, now.Add(time.Duration(i)*time.Hour))
	}

	_, ok, err := store.TypicalMinutes(ctx, mine+1000, "put the bins out")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestAnEmptyLabelAnswersNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	_, ok, err := store.TypicalMinutes(context.Background(), p, "   ")
	require.NoError(t, err)
	require.False(t, ok)
}

// The property the whole narrowing exists for, asserted as an absence: there
// is no store function that returns the runs, so nothing can count them and
// nothing can render how often you did or did not finish.
//
// A test cannot prove a function does not exist, so this pins the next best
// thing — the one function that reads this table answers a duration and
// nothing else, and it answers the same duration however many runs there are.
func TestNothingHereAnswersHowOftenYouFinished(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i := 0; i < 3; i++ {
		ranFor(t, store, p, "put the bins out", 10, now.Add(time.Duration(i)*time.Hour))
	}
	three, _, err := store.TypicalMinutes(ctx, p, "put the bins out")
	require.NoError(t, err)

	for i := 3; i < 20; i++ {
		ranFor(t, store, p, "put the bins out", 10, now.Add(time.Duration(i)*time.Hour))
	}
	twenty, _, err := store.TypicalMinutes(ctx, p, "put the bins out")
	require.NoError(t, err)

	require.Equal(t, three, twenty, "the answer carries a count of runs in it")
}
