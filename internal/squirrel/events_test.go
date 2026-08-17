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
