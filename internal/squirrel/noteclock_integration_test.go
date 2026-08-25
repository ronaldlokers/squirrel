//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A note comes back in the person's clock, not the driver's.
//
// The same defect the fixed points had, on the other table. A timestamptz is
// an instant and carries no zone, so pgx hands it back in UTC — and the pods
// run in UTC on purpose since #148, so anything captured late in the evening
// wore the previous day's date on the corner of its card.
//
// These run under TZ=UTC, which is what the clusters do and what no test was
// doing when the first half of this was found.

// lateLastNight is half past midnight where the person is, which is half past
// ten the previous evening in UTC. Everything about this test is that those
// are different days.
func lateLastNight(t *testing.T, loc *time.Location) time.Time {
	t.Helper()
	return time.Date(2026, 3, 14, 0, 30, 0, 0, loc)
}

func TestANoteComesBackInThePersonsClock(t *testing.T) {
	store := withStore(t)
	ams, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	store.In(ams)
	ctx := context.Background()
	p := owner(t, store)

	at := lateLastNight(t, ams)
	require.Equal(t, 13, at.UTC().Day(), "the fixture does not straddle a day boundary")

	id, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: "the boiler makes a noise",
		Payload: []byte(squirrel.ScreenCapture), ReceivedAt: at,
	})
	require.NoError(t, err)

	// By id, which is what the card and the undo path both read.
	one, found, err := store.ItemByID(ctx, p, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 14, one.ReceivedAt.Day(),
		"a note captured after midnight carries the previous day")
	require.Equal(t, "14 March", one.ReceivedAt.Format("2 January"))

	// And in a list, which is what every door draws from.
	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "14 March", items[0].ReceivedAt.Format("2 January"))
}

// With no location configured nothing is converted, which is what every test
// that does not care about this relies on.
func TestWithNoLocationANoteIsLeftAlone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	at := time.Now()
	_, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: "the boiler",
		Payload: []byte(squirrel.ScreenCapture), ReceivedAt: at,
	})
	require.NoError(t, err)

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	// Truncated, because a timestamptz keeps microseconds and Go keeps
	// nanoseconds — the round trip loses the tail and always has.
	require.True(t,
		items[0].ReceivedAt.Truncate(time.Millisecond).Equal(at.Truncate(time.Millisecond)),
		"the instant itself moved")
}
