//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func amsterdam(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	return loc
}

func scheduler(t *testing.T, store *squirrel.Store, p int64, send squirrel.Sender) *squirrel.Scheduler {
	t.Helper()
	return squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, Send: send, PersonID: p, ConversationID: "9",
		At: 8 * time.Hour, Location: amsterdam(t),
	})
}

func TestSchedulerSendsAfterTheHour(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)

	before := time.Date(2026, 8, 15, 7, 59, 0, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, before))
	require.Empty(t, *got, "not yet 08:00")

	after := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, after))
	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "vacuum")
}

// The failure the original kickoff calls the worst possible one.
func TestSchedulerSendsOncePerDayAcrossRestarts(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	at := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, at))
	// A fresh Scheduler is a fresh process.
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, at.Add(time.Minute)))

	require.Len(t, *got, 1)
}

// A stale digest is worse than a missing one.
func TestSchedulerSkipsADayItSleptThrough(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	wednesdayEarly := time.Date(2026, 8, 19, 3, 0, 0, 0, amsterdam(t))
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, wednesdayEarly))
	require.Empty(t, *got, "Tuesday's digest is not sent at 03:00 on Wednesday")
}

func TestSchedulerSendsNothingWhenThereIsNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	at := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, at))
	require.Empty(t, *got)
}

// midnight.Add(At) moves an absolute instant, not a wall clock: on the day
// clocks spring forward, midnight is 00:00 CET and adding 8h lands on 09:00
// CEST — an hour late. The threshold must instead be built as "08:00 on this
// calendar date" directly.
func TestSchedulerFiresAtWallClockOnSpringForward(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 2, 1, 9, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)

	before := time.Date(2026, 3, 29, 7, 30, 0, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, before))
	require.Empty(t, *got, "not yet 08:00 wall clock on the day clocks spring forward")

	after := time.Date(2026, 3, 29, 8, 30, 0, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, after))
	require.Len(t, *got, 1, "08:30 wall clock is past the 08:00 threshold")
}

// midnight.Add(At) on the day clocks fall back lands on 07:00 CET — an hour
// early, which would send the digest before its configured time.
func TestSchedulerFiresAtWallClockOnFallBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 9, 1, 9, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)

	before := time.Date(2026, 10, 25, 7, 30, 0, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, before))
	require.Empty(t, *got, "07:30 wall clock is still before the 08:00 threshold on the day clocks fall back")

	after := time.Date(2026, 10, 25, 8, 30, 0, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, after))
	require.Len(t, *got, 1, "08:30 wall clock is past the 08:00 threshold")
}

func TestSchedulerRunStopsWithTheContext(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	send, _ := recorder()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); scheduler(t, store, p, send).Run(ctx) }()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}
