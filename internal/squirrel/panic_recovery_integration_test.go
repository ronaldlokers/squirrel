//go:build integration

package squirrel

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// These two guard the actual failure mode described in the review: a panic
// in Match escaping Applier.Apply through Drain.one and out of Drain.Run
// killed the whole process, and because CapturesSince re-runs Match over
// every stored row on every digest attempt, the very next scheduler tick hit
// the same row and panicked too — forever, since the row is durable and
// never reprocessed differently.
//
// Match itself has no reachable panic once the byte-length bug in ParseEvery
// is fixed (see intent_duration_test.go), so matchFn is swapped for a
// stand-in that always panics — exercising the recover as a safety net
// rather than re-testing the fixed bug. This file is package squirrel (not
// squirrel_test) specifically so it can reach the unexported matchFn seam,
// which duplicates a little of testsupport_test.go's store setup rather than
// sharing it: that helper lives in the external squirrel_test package and is
// not visible from here.

func withPanickingMatch(t *testing.T) {
	t.Helper()
	prev := matchFn
	matchFn = func(string) Intent { panic("boom") }
	t.Cleanup(func() { matchFn = prev })
}

func testStoreForPanicRecovery(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()

	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")

	store, err := OpenStore(ctx, url)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	require.NoError(t, store.Migrate(ctx))
	_, err = store.Pool().Exec(ctx,
		`truncate table prompt_lines, prompts, events, items, chores, identities, people
		 restart identity cascade`)
	require.NoError(t, err)
	return store
}

// TestDrainSurvivesAPanicInMatch is the exact reported scenario: a capture
// lands, InsertItem commits it and the spool file is removed, and only then
// does the Applier run Match over it and panic. The row must stay landed and
// Drain.Once must return normally rather than crash the process.
func TestDrainSurvivesAPanicInMatch(t *testing.T) {
	store := testStoreForPanicRecovery(t)
	ctx := context.Background()

	p, err := store.SeedOwner(ctx, "ronald",
		[]IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)

	dir := t.TempDir()
	sp, err := OpenSpool(dir)
	require.NoError(t, err)

	send := func(context.Context, string, string) error { return nil }
	applier := NewApplier(store, send, Chat{}, nil)

	_, err = sp.Write(Capture{
		Transport:      "campfire",
		ExternalID:     Ptr("99"),
		ConversationID: Ptr("7"),
		SenderID:       Ptr("1"),
		Text:           "buy milk",
		ReceivedAt:     time.Now(),
		Payload:        []byte(`{}`),
	})
	require.NoError(t, err)

	withPanickingMatch(t)

	var drainErrs []error
	var result DrainResult
	require.NotPanics(t, func() {
		result = NewDrain(DrainOptions{
			Spool: sp, Store: store, Interval: time.Second, Applier: applier,
			OnError: func(err error) { drainErrs = append(drainErrs, err) },
		}).Once(ctx)
	})

	require.Equal(t, DrainResult{Inserted: 1}, result,
		"the row lands even though applying its intent panicked")
	require.Len(t, drainErrs, 1)
	require.Contains(t, drainErrs[0].Error(), "panicked")

	var n int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from items where person_id = $1`, p).Scan(&n))
	require.Equal(t, 1, n, "the capture is durable regardless of the panic")
}

// TestSchedulerOnceSurvivesAPanicInMatch covers the second half of the same
// failure: CapturesSince runs Match over every stored row on every digest
// attempt, so a row that panics Match must not be able to fatally crash the
// scheduler on every future tick.
func TestSchedulerOnceSurvivesAPanicInMatch(t *testing.T) {
	store := testStoreForPanicRecovery(t)
	ctx := context.Background()

	p, err := store.SeedOwner(ctx, "ronald", nil)
	require.NoError(t, err)

	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	_, err = store.InsertItem(ctx, Item{
		Transport:  "campfire",
		PersonID:   Ptr(p),
		RawText:    "buy milk",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Date(2026, 8, 15, 7, 0, 0, 0, loc),
	})
	require.NoError(t, err)

	withPanickingMatch(t)

	s := NewScheduler(SchedulerOptions{
		Store: store, Send: func(context.Context, string, string) error { return nil },
		PersonID: p, ConversationID: "9", At: 8 * time.Hour, Location: loc,
	})

	at := time.Date(2026, 8, 15, 8, 0, 1, 0, loc)

	var err1 error
	require.NotPanics(t, func() { err1 = s.Once(ctx, at) })
	require.Error(t, err1)
	require.Contains(t, err1.Error(), "panicked")

	// The scheduler must not have wedged itself: a later tick, on the same
	// date, must still try rather than staying fatally broken.
	var err2 error
	require.NotPanics(t, func() { err2 = s.Once(ctx, at.Add(time.Minute)) })
	require.Error(t, err2)
	require.Contains(t, err2.Error(), "panicked")
}
