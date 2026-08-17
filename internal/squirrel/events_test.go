//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompletionResetsTheClock(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	overdue := time.Now().Add(twoWeeks + time.Hour)
	due, err := store.DueChores(ctx, p, overdue)
	require.NoError(t, err)
	require.Len(t, due, 1)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", overdue))

	due, err = store.DueChores(ctx, p, overdue.Add(time.Hour))
	require.NoError(t, err)
	require.Empty(t, due, "completing it makes it not due")
}

// The claim the whole derived-baseline design rests on: a completion from a
// sensor resets the clock with no chore-specific code involved.
func TestSensorEventResetsTheClockToo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	overdue := time.Now().Add(twoWeeks + time.Hour)

	_, err = store.Pool().Exec(ctx,
		`insert into events (chore_id, person_id, source, occurred_at, payload)
		 values ($1, $2, 'sensor', $3, '{"device":"roborock"}')`, c.ID, p, overdue)
	require.NoError(t, err)

	due, err := store.DueChores(ctx, p, overdue.Add(time.Hour))
	require.NoError(t, err)
	require.Empty(t, due)
}

func TestSinceDaysCountsFromTheLastCompletion(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	done := time.Now().Add(-19 * 24 * time.Hour)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", done))

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.Equal(t, 19, due[0].SinceDays)
	require.Equal(t, 14, due[0].EveryDays)
}

func TestRetractionRestoresTheClock(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 15*24*time.Hour)

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "overdue before the completion")

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))
	due, err = store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due, "the completion reset the clock")

	found, err := store.RetractCompletion(ctx, c.ID, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	due, err = store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "retraction put it back")
}

func TestRetractionLeavesTheEventInTheLog(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))
	_, err = store.RetractCompletion(ctx, c.ID, p, time.Now())
	require.NoError(t, err)

	var total, live int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*), count(*) filter (where retracted_at is null)
		   from events where chore_id = $1`, c.ID).Scan(&total, &live))
	require.Equal(t, 1, total, "the event is still there")
	require.Equal(t, 0, live, "but it no longer counts")
}

func TestRetractingNothingIsNotAnError(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	found, err := store.RetractCompletion(ctx, c.ID, p, time.Now())
	require.NoError(t, err, "an un-tap with nothing to undo is a no-op, not a failure")
	require.False(t, found)
}

// A tap carries a position, and the position resolves through a person-scoped
// prompt — so this is defence in depth rather than the only guard. It goes in
// because "unreachable" now depends on a decision rather than on structure.
func TestCompletionRefusesAnotherPersonsChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	a := owner(t, store)
	b, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)

	theirs, err := store.UpsertChore(ctx, b, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	require.Error(t, store.RecordCompletion(ctx, theirs.ID, a, "ack", time.Now()))

	var n int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1`, theirs.ID).Scan(&n))
	require.Zero(t, n)
}
