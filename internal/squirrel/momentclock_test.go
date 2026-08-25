//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A fixed point comes back in the person's clock, not the driver's.
//
// A timestamptz is an instant and carries no zone, so pgx hands it back in
// UTC. Every one of these times is then printed — "at 14:30", "leave about
// 14:05" — and printing the right instant with the wrong digits on it is a
// missed appointment.
//
// This is issue #148 one layer further out. That fix threaded the location
// into everything that parses a time; the reading side was never audited.
func TestAFixedPointComesBackInThePersonsClock(t *testing.T) {
	store := withStore(t)
	ams, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	store.In(ams)
	ctx := context.Background()
	p := owner(t, store)

	// A wall-clock time somebody would recognise, in their own day.
	at := time.Date(time.Now().Year()+1, 3, 14, 14, 30, 0, 0, ams)
	made, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: at, Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)

	coming, err := store.Upcoming(ctx, p, time.Now(), 5)
	require.NoError(t, err)
	require.NotEmpty(t, coming)
	require.Equal(t, "14:30", coming[0].Starts.Format("15:04"),
		"the agenda prints the appointment in the wrong clock")

	one, found, err := store.MomentByID(ctx, p, made.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "14:30", one.Starts.Format("15:04"),
		"the fixed point prints in the wrong clock")

	// And the sentence built from it, which is what a notification says.
	require.Contains(t, squirrel.LeaveWords(one), "at 14:30",
		"the leave-by sentence names the wrong time")
}

// With no location configured nothing is converted, which is what every test
// that does not care about this relies on.
func TestWithNoLocationAFixedPointIsLeftAlone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	at := time.Now().Add(3 * time.Hour)
	_, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: at, Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)

	coming, err := store.Upcoming(ctx, p, time.Now(), 5)
	require.NoError(t, err)
	require.NotEmpty(t, coming)
	// Truncated, because a timestamptz keeps microseconds and Go keeps
	// nanoseconds — the round trip loses the tail and always has.
	require.True(t, coming[0].Starts.Truncate(time.Millisecond).Equal(at.Truncate(time.Millisecond)),
		"the instant itself moved")
}
