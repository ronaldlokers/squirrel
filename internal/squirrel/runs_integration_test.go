//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestARunIsRememberedAndOffered(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.MarkRun(ctx, p, squirrel.RunPile, now.Add(-20*time.Minute)))

	got, found, err := store.RunFor(ctx, p, now)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.RunPile, got.Place)
	require.InDelta(t, (20 * time.Minute).Seconds(), got.Since.Seconds(), 2)
}

// The rule to hold hardest: coming back to yesterday's half-finished pile is
// not resuming, it is being nagged.
func TestAnOldRunIsNotOffered(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.MarkRun(ctx, p, squirrel.RunPile, now.Add(-squirrel.KeepingPlace-time.Minute)))

	_, found, err := store.RunFor(ctx, p, now)
	require.NoError(t, err)
	require.False(t, found, "a stale run was offered back")
}

// Answering keeps it alive, so the clock measures silence rather than how long
// the run has been going. A long afternoon of triage is not stale.
func TestAnsweringKeepsARunAlive(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.MarkRun(ctx, p, squirrel.RunPile, now.Add(-4*time.Hour)))
	require.NoError(t, store.MarkRun(ctx, p, squirrel.RunPile, now.Add(-5*time.Minute)))

	_, found, err := store.RunFor(ctx, p, now)
	require.NoError(t, err)
	require.True(t, found, "a run being worked was treated as abandoned")
}

// One row per person: you are not doing two.
func TestASecondRunReplacesTheFirst(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.MarkRun(ctx, p, squirrel.RunPile, now))
	require.NoError(t, store.MarkRun(ctx, p, "chores", now))

	got, found, err := store.RunFor(ctx, p, now)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "chores", got.Place)

	var rows int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from runs where person_id = $1`, p).Scan(&rows))
	require.Equal(t, 1, rows, "runs accumulated into a history of afternoons")
}

func TestEndingARunForgetsIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.MarkRun(ctx, p, squirrel.RunPile, now))
	require.NoError(t, store.EndRun(ctx, p))

	_, found, err := store.RunFor(ctx, p, now)
	require.NoError(t, err)
	require.False(t, found)
}

// Ending one that is not there is not an error: finishing and stopping both
// call it, and so does starting fresh.
func TestEndingARunThatIsNotThereIsFine(t *testing.T) {
	store := withStore(t)
	require.NoError(t, store.EndRun(context.Background(), owner(t, store)))
}

// Somebody else's run is never yours.
func TestARunBelongsToOnePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-theirs", "theirs")
	require.NoError(t, err)
	now := time.Now()

	require.NoError(t, store.MarkRun(ctx, theirs, squirrel.RunPile, now))

	_, found, err := store.RunFor(ctx, mine, now)
	require.NoError(t, err)
	require.False(t, found, "somebody else's run was offered to me")
}
