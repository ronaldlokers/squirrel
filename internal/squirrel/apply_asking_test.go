//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The sentence carries two facts and both survive the round trip.
func TestDefiningEveryOtherTuesdayKeepsBoth(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	triage(t, store, p, "every other tuesday: bins out")

	chores, err := store.ActiveChores(context.Background(), p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, 14, chores[0].EveryDays, "the rhythm")
	require.Equal(t, "tuesdays", chores[0].Ask.Days.Words(), "and the preference")
}

// Changing the rhythm is not a request to forget the preference. Saying
// "every 2 weeks bins out" after "every other tuesday: bins out" would
// otherwise silently drop the tuesday.
func TestChangingTheRhythmKeepsThePreference(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "every other tuesday: bins out")

	// The path the chores screen takes when you press HOW OFTEN: an upsert
	// with no preference in it.
	_, err := store.UpsertChore(ctx, p, "bins out", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, 7, chores[0].EveryDays)
	require.Equal(t, "tuesdays", chores[0].Ask.Days.Words())
}

// The load-bearing one: a preference must never make a chore late. It is due
// exactly when it was; only the asking waits.
func TestAPreferenceDoesNotChangeWhenAChoreIsDue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "every other tuesday: bins out")

	// A Sunday, which is not the preferred day.
	sunday := time.Date(2026, 8, 23, 12, 0, 0, 0, time.Local)
	require.Equal(t, time.Sunday, sunday.Weekday())

	// Twenty days before the Sunday being asked about, not twenty days before
	// whenever this happens to run. See backdateTo.
	backdateTo(t, store, "bins out", sunday.AddDate(0, 0, -20))

	due, err := store.DueChores(ctx, p, sunday)
	require.NoError(t, err)
	require.Len(t, due, 1, "due is due, whatever day it is")
	require.False(t, due[0].Ask.Open(sunday), "but not worth raising today")
}

// And the chores screen still shows it, because seeing a chore is not the same
// as being interrupted by one.
func TestTheChoresListIgnoresTheAskingWindow(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	triage(t, store, p, "every weekday: the tablets")

	chores, err := store.ActiveChores(context.Background(), p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, "weekdays", chores[0].Ask.Days.Words())
}
