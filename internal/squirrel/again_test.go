//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The cost of an interruption is the reconstruction. The timer already knew
// what you were on and used to throw it away.

func TestWhatYouWereOnSurvivesTheTimerEnding(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.StartTimer(ctx, p, "the kitchen", time.Minute, time.Now().Add(-2*time.Minute))
	require.NoError(t, err)
	_, found, err := store.ClaimFinishedTimer(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	o, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.OfferAgain, o.Kind)
	require.Equal(t, "the kitchen", o.Text)
	require.Contains(t, o.Because, "you were on this")
}

// Stopping is not failing — you may have stopped to answer the door, and that
// is exactly the case this is for.
func TestStoppingEarlyStillLeavesTheWayBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.StartTimer(ctx, p, "the kitchen", 20*time.Minute, time.Now())
	require.NoError(t, err)
	require.NoError(t, store.StopTimer(ctx, p))

	o, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.OfferAgain, o.Kind)
}

// Never a sentence about finishing. "You were on this" is a fact; "you did not
// finish this" is about the person.
func TestTheBreadcrumbNeverMentionsFinishing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.StartTimer(ctx, p, "the kitchen", 20*time.Minute, time.Now())
	require.NoError(t, err)
	require.NoError(t, store.StopTimer(ctx, p))

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NotContains(t, o.Because, "finish")
	require.NotContains(t, o.Because, "abandon")
	require.NotContains(t, squirrel.NowMessage(o).Text, "finish")
}

// After an hour it is a fact about earlier rather than an offer.
func TestTheBreadcrumbLapses(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.StartTimer(ctx, p, "the kitchen", time.Minute, time.Now())
	require.NoError(t, err)
	require.NoError(t, store.StopTimer(ctx, p))
	agedTimer(t, store, p, 2*time.Hour)

	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found)
}

// A running timer is still the answer to "what now", and the breadcrumb of the
// one before it must not shadow it.
func TestARunningTimerBeatsTheBreadcrumb(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.StartTimer(ctx, p, "the kitchen", time.Minute, time.Now())
	require.NoError(t, err)
	require.NoError(t, store.StopTimer(ctx, p))
	_, err = store.StartTimer(ctx, p, "the bathroom", 20*time.Minute, time.Now())
	require.NoError(t, err)

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.Equal(t, squirrel.OfferTimer, o.Kind)
	require.Equal(t, "the bathroom", o.Text)
}

// Picking something back up is yours from an hour ago, not Squirrel's
// initiative, so the capacity gate does not touch it.
func TestTheBreadcrumbSurvivesALowDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "screen", time.Now()))
	_, err := store.StartTimer(ctx, p, "the kitchen", time.Minute, time.Now())
	require.NoError(t, err)
	require.NoError(t, store.StopTimer(ctx, p))

	o, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.OfferAgain, o.Kind)
}

// One row per person, still. There is nowhere for timers started and abandoned
// to accumulate, which is what the table was protecting in the first place.
func TestNothingAccumulatesBehindTheBreadcrumb(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for _, label := range []string{"one", "two", "three"} {
		_, err := store.StartTimer(ctx, p, label, time.Minute, time.Now())
		require.NoError(t, err)
		require.NoError(t, store.StopTimer(ctx, p))
	}

	var rows int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from timers where person_id = $1`, p).Scan(&rows))
	require.Equal(t, 1, rows)
}

// The scheduler says "time" once, and a second tick finds nothing to say.
func TestATimerIsStillOnlyAnnouncedOnce(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.StartTimer(ctx, p, "the kitchen", time.Minute, time.Now().Add(-2*time.Minute))
	require.NoError(t, err)

	_, found, err := store.ClaimFinishedTimer(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	_, found, err = store.ClaimFinishedTimer(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found)
}

// agedTimer pushes an ended timer into the past, so the lapse can be tested
// without waiting an hour.
func agedTimer(t *testing.T, store *squirrel.Store, personID int64, by time.Duration) {
	t.Helper()
	_, err := store.Pool().Exec(context.Background(),
		`update timers set ended_at = now() - $2::interval where person_id = $1`,
		personID, by.String())
	require.NoError(t, err)
}
