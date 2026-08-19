//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// snoozeTap stands a chore up and nudges about it. The tolerance is an hour
// rather than the usual week: the assertions below need an instant that is
// past the prompt's own tolerance window but still inside a snooze that lasts
// until tomorrow, and a week-long window leaves no such instant.
func snoozeTap(t *testing.T) (*squirrel.Store, int64, squirrel.Chore) {
	t.Helper()
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "bins out", twoWeeks, time.Hour)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	id, err := store.RecordPrompt(ctx, p, "9", "nudge", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	return store, p, c
}

func snoozedUntil(t *testing.T, store *squirrel.Store) time.Time {
	t.Helper()
	var until *time.Time
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select snoozed_until from chores where name = 'bins out'`).Scan(&until))
	require.NotNil(t, until, "the chore was never snoozed")
	return *until
}

// The nudge arrives at the moment you are least able to decide anything, and
// until now the only answer it could hear was yes. Typing !snooze at exactly
// that moment is the cost this button removes.
//
// Tomorrow rather than a duration, because the label is the duration: a button
// cannot ask how long, and the honest answer is the one written on it.
func TestNotTodayOnTheNudgeIsQuietUntilTomorrow(t *testing.T) {
	store, p, _ := snoozeTap(t)
	ctx := context.Background()
	send, _ := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "snooze:1", true), squirrel.Ptr(p)))

	until := snoozedUntil(t, store)
	require.True(t, until.After(time.Now()), "quiet from now")
	require.Less(t, time.Until(until), 48*time.Hour, "tomorrow, not a week")
	// Midnight, not "in 24 hours" — the latter would move the moment it asks a
	// little later every time it was pressed.
	require.Equal(t, 0, until.Hour())
	require.Equal(t, 0, until.Minute())
}

// The gate itself, asked at two instants either side of the quiet, so the test
// does not depend on what time of day it runs at.
func TestTheQuietStopsTheAskingAndThenEnds(t *testing.T) {
	store, p, _ := snoozeTap(t)
	ctx := context.Background()
	send, _ := recorder()

	past := time.Now().Add(2 * time.Hour)
	due, err := store.DueChores(ctx, p, past)
	require.NoError(t, err)
	require.Len(t, due, 1, "due before the tap")

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "snooze:1", true), squirrel.Ptr(p)))

	// A fixed quiet, so the two instants below are unambiguous whatever hour
	// this test runs at.
	_, err = store.Pool().Exec(ctx, `update chores set snoozed_until = now() + interval '3 hours'`)
	require.NoError(t, err)

	due, err = store.DueChores(ctx, p, time.Now().Add(2*time.Hour))
	require.NoError(t, err)
	require.Empty(t, due, "inside the quiet, it stops asking")

	due, err = store.DueChores(ctx, p, time.Now().Add(4*time.Hour))
	require.NoError(t, err)
	require.Len(t, due, 1, "past the quiet, it asks again")
}

// Nothing here is a decision you cannot reverse: untapping is the same write
// with a time in the past, exactly as an unselected done retracts a completion.
func TestUntappingNotTodayAsksAgain(t *testing.T) {
	store, p, _ := snoozeTap(t)
	ctx := context.Background()
	send, _ := recorder()
	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)

	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "snooze:1", true), squirrel.Ptr(p)))
	require.True(t, snoozedUntil(t, store).After(time.Now()))

	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "snooze:1", false), squirrel.Ptr(p)))
	require.True(t, snoozedUntil(t, store).Before(time.Now()), "asking again")

	due, err := store.DueChores(ctx, p, time.Now().Add(2*time.Hour))
	require.NoError(t, err)
	require.Len(t, due, 1)
}

// Putting it off is not doing it. The baseline is untouched, so when the quiet
// ends the chore is exactly as due as it was.
func TestNotTodayIsNotDone(t *testing.T) {
	store, p, _ := snoozeTap(t)
	ctx := context.Background()
	send, _ := recorder()
	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)

	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "snooze:1", true), squirrel.Ptr(p)))

	// Wind the quiet back to the past: the same thing tomorrow does.
	_, err := store.Pool().Exec(ctx, `update chores set snoozed_until = now() - interval '1 minute'`)
	require.NoError(t, err)

	due, err := store.DueChores(ctx, p, time.Now().Add(2*time.Hour))
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.False(t, due[0].EverDone, "putting it off is not doing it")
	require.GreaterOrEqual(t, due[0].SinceDays, 20, "the clock kept running while it was quiet")
}
