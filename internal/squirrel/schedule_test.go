//go:build integration

package squirrel_test

import (
	"context"
	"errors"
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

// A fixed "yesterday midnight" window either double-counts (every normal
// day's window overlaps the previous day's between local midnight and the
// send) or, after a missed day, drops content in the gap between where the
// old window ended and the new one begins. Anchoring to the last digest that
// actually sent closes that gap.
func TestSchedulerCarriesCapturesAcrossAMissedDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	// Monday's digest sends normally.
	monday := time.Date(2026, 8, 17, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, monday))
	require.Len(t, *got, 1)

	// Captured Monday afternoon, well after Monday's digest already sent.
	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport:  "campfire",
		PersonID:   squirrel.Ptr(p),
		RawText:    "fixed the leaky tap",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Date(2026, 8, 17, 15, 0, 0, 0, amsterdam(t)),
	})
	require.NoError(t, err)

	// Tuesday is slept through entirely — no Once call lands in its window
	// at all, simulating a missed day.

	// Wednesday's digest is the next one that actually sends. A fixed
	// "yesterday midnight" window would start this one at Tuesday 00:00,
	// missing Monday afternoon's capture entirely.
	wednesday := time.Date(2026, 8, 19, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, wednesday))
	require.Len(t, *got, 2)
	require.Contains(t, (*got)[1].text, "leaky tap")
}

// The in-memory guard is invisible from Sender output alone — a guarded and
// an unguarded Once both return nil on a repeat call, since the unique index
// absorbs the duplicate either way. Closing the pool after the first send
// makes the difference observable: a guarded Once short-circuits before
// touching the store and returns nil; an unguarded one reaches the first
// query and gets a closed-pool error.
func TestSchedulerGuardShortCircuitsAfterASend(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)

	at := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, at))
	require.Len(t, *got, 1)

	store.Pool().Close()

	require.NoError(t, s.Once(ctx, at.Add(time.Minute)),
		"a guard that had armed would return nil without ever reaching the closed pool")
}

// A guard armed unconditionally would silently suppress every future tick
// after any single failure — the bot going quiet is exactly the failure this
// phase exists to prevent. A failed tick must not arm it: the next tick, on
// the same local date, must still try.
func TestSchedulerGuardDoesNotArmOnAFailedTick(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)
	at := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, s.Once(cancelled, at), "a cancelled context must surface as an error from the first query")
	require.Empty(t, *got)

	require.NoError(t, s.Once(ctx, at.Add(time.Minute)),
		"the failed tick must not have armed the guard")
	require.Len(t, *got, 1)
}

// The guard is keyed on the local calendar date, not "has a digest ever been
// sent" — it must not carry over once the date rolls to the next one.
func TestSchedulerGuardDoesNotSurviveDateRollover(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	_, err := store.InsertItem(ctx, squirrel.Item{
		Transport:  "campfire",
		PersonID:   squirrel.Ptr(p),
		RawText:    "day one thought",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Date(2026, 8, 15, 7, 0, 0, 0, amsterdam(t)),
	})
	require.NoError(t, err)

	s := scheduler(t, store, p, send)

	dayOne := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, dayOne))
	require.Len(t, *got, 1)

	// Captured after day one's digest sent, so it falls inside day two's
	// anchored window.
	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport:  "campfire",
		PersonID:   squirrel.Ptr(p),
		RawText:    "day two thought",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Date(2026, 8, 15, 20, 0, 0, 0, amsterdam(t)),
	})
	require.NoError(t, err)

	dayTwo := time.Date(2026, 8, 16, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, dayTwo))
	require.Len(t, *got, 2, "a new local date must not be suppressed by yesterday's guard")
	require.Contains(t, (*got)[1].text, "day two thought")
}

// ErrDigestAlreadySent means a row is genuinely committed for the day —
// by this process on an earlier tick, or by another one entirely — so it
// must still count as success, and must still arm the guard.
func TestSchedulerGuardArmsOnErrDigestAlreadySent(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	at := time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))
	midnight := time.Date(2026, 8, 15, 0, 0, 0, 0, amsterdam(t))

	// Simulate another process having already recorded today's digest.
	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), &midnight, nil)
	require.NoError(t, err)

	s := scheduler(t, store, p, send)
	require.NoError(t, s.Once(ctx, at), "ErrDigestAlreadySent must be treated as success")
	require.Empty(t, *got, "no send when the digest was already recorded by someone else")

	store.Pool().Close()

	require.NoError(t, s.Once(ctx, at.Add(time.Minute)),
		"the guard must have armed on the ErrDigestAlreadySent path too")
}

// RecordPrompt commits a digest's row before Send is even attempted — the
// ordering that makes the unique index the idempotency guarantee — so a
// failed send still leaves a row behind. Without a "was actually delivered"
// predicate, LastDigestSentAt would anchor the next digest's capture window
// to that row's sent_at anyway, and every capture made on the day the send
// failed would fall before the (wrongly early) anchor of the next digest
// that actually sent — gone from every digest for good.
func TestSchedulerRecoversCapturesAfterAFailedSend(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	// Captured before Monday's digest is attempted.
	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport:  "campfire",
		PersonID:   squirrel.Ptr(p),
		RawText:    "before the outage",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Date(2026, 8, 17, 7, 0, 0, 0, amsterdam(t)),
	})
	require.NoError(t, err)

	failingSend := func(context.Context, string, string) error {
		return errors.New("campfire unreachable")
	}
	monday := time.Date(2026, 8, 17, 8, 0, 1, 0, amsterdam(t))
	require.Error(t, scheduler(t, store, p, failingSend).Once(ctx, monday),
		"Send fails, so Once must report an error")

	// Captured after Monday's failed send.
	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport:  "campfire",
		PersonID:   squirrel.Ptr(p),
		RawText:    "after the outage",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Date(2026, 8, 17, 15, 0, 0, 0, amsterdam(t)),
	})
	require.NoError(t, err)

	send, got := recorder()
	tuesday := time.Date(2026, 8, 18, 8, 0, 1, 0, amsterdam(t))
	require.NoError(t, scheduler(t, store, p, send).Once(ctx, tuesday))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "before the outage",
		"a capture from the day the send failed must not be lost")
	require.Contains(t, (*got)[0].text, "after the outage")
}

// The tolerance gate compares absolute instants, and DefaultTolerance floors
// at 24h, so a daily chore's tolerance is also exactly 24h. Two ticks 900ms
// and 400ms into their respective seconds — nothing more exotic than
// ordinary scheduler jitter — land half a second under 24 hours apart in
// absolute time, and an untouched 24h tolerance used to make that enough to
// skip the second morning's nudge entirely.
func TestSchedulerToleratesTickJitterOnADailyChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "meds", oneDay, oneDay)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 8, 15, 8, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)

	day1 := time.Date(2026, 8, 17, 8, 0, 0, 900_000_000, amsterdam(t))
	require.NoError(t, s.Once(ctx, day1))
	require.Len(t, *got, 1, "meds is already overdue by day 1")

	day2 := time.Date(2026, 8, 18, 8, 0, 0, 400_000_000, amsterdam(t))
	require.NoError(t, s.Once(ctx, day2))
	require.Len(t, *got, 2, "half a second of jitter must not skip a daily chore's nudge")
	require.Contains(t, (*got)[1].text, "meds")
}

// In Europe/Amsterdam, 08:00 on 2026-03-28 to 08:00 on 2026-03-29 is only 23
// hours: clocks spring forward during that night. A daily chore's tolerance
// is exactly 24h (DefaultTolerance floors there too), so without slack the
// digest on the 29th falls short of its own chore's tolerance by an hour and
// the chore is skipped that one morning every year, in every timezone that
// observes DST.
func TestSchedulerDailyChoreSurvivesSpringForward(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "meds", oneDay, oneDay)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 3, 25, 8, 0, 0, 0, amsterdam(t))))

	s := scheduler(t, store, p, send)

	for i, day := range []int{26, 27, 28, 29} {
		at := time.Date(2026, 3, day, 8, 0, 1, 0, amsterdam(t))
		require.NoError(t, s.Once(ctx, at))
		require.Len(t, *got, i+1, "2026-03-%d morning's nudge", day)
	}
	require.Contains(t, (*got)[3].text, "meds",
		"the spring-forward morning must still get its nudge")
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
