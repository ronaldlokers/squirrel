//go:build integration

package squirrel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func drainFixture(t *testing.T) (*squirrel.Store, *squirrel.Spool, string) {
	t.Helper()
	store := withStore(t)
	dir := t.TempDir()
	sp, err := squirrel.OpenSpool(dir)
	require.NoError(t, err)
	return store, sp, dir
}

func personOf(t *testing.T, store *squirrel.Store) *int64 {
	t.Helper()
	var id *int64
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select person_id from items limit 1`).Scan(&id))
	return id
}

func TestDrainInsertsAndDeletes(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	got := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: store, Interval: time.Second,
	}).Once(context.Background())

	require.Equal(t, squirrel.DrainResult{Inserted: 1}, got)
	require.Equal(t, 1, countItems(t, store))

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestDrainResolvesASeededIdentity(t *testing.T) {
	store, sp, _ := drainFixture(t)
	id, err := store.SeedOwner(context.Background(), "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)

	_, err = sp.Write(capture(nil))
	require.NoError(t, err)
	squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second}).
		Once(context.Background())

	require.Equal(t, id, *personOf(t, store))
}

// A capture is never held hostage to knowing whose it was.
func TestDrainStoresUnknownIdentityWithNilPerson(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(func(c *squirrel.Capture) { c.SenderID = squirrel.Ptr("999") }))
	require.NoError(t, err)

	var reported string
	got := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: store, Interval: time.Second,
		OnUnknownIdentity: func(transport, senderID string) { reported = transport + "/" + senderID },
	}).Once(context.Background())

	require.Equal(t, 1, got.Inserted)
	require.Nil(t, personOf(t, store))
	require.Equal(t, "campfire/999", reported)

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestDrainDedupesARedeliveredMessage(t *testing.T) {
	store, sp, _ := drainFixture(t)
	drain := squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second})

	_, err := sp.Write(capture(nil))
	require.NoError(t, err)
	drain.Once(context.Background())

	_, err = sp.Write(capture(nil))
	require.NoError(t, err)
	drain.Once(context.Background())

	require.Equal(t, 1, countItems(t, store))
}

func countEvents(t *testing.T, store *squirrel.Store) int {
	t.Helper()
	var n int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from events`).Scan(&n))
	return n
}

// InsertItem's ON CONFLICT dedupes the row on redelivery, but before this
// fix the drain could not tell that no-op apart from a fresh insert, so it
// ran the Applier again anyway — a second completion event and a second
// reply for the one "done". The row landing at most once must mean the
// intent runs at most once too.
func TestDrainAppliesARedeliveredMessageOnlyOnce(t *testing.T) {
	store, sp, _ := drainFixture(t)
	ctx := context.Background()

	p, err := store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	promptID, err := store.RecordPrompt(ctx, p, "7", "digest", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-1", time.Now()))

	send, got := recorder()
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: store, Interval: time.Second,
		Applier: squirrel.NewApplier(store, send, squirrel.Chat{}, nil),
	})

	done := capture(func(c *squirrel.Capture) { c.Text = "done 1" })

	_, err = sp.Write(done)
	require.NoError(t, err)
	drain.Once(ctx)

	// Redelivered: the same external id arrives a second time.
	_, err = sp.Write(done)
	require.NoError(t, err)
	drain.Once(ctx)

	require.Equal(t, 1, countEvents(t, store), "one completion, not two")
	require.Len(t, *got, 1, "one reply, not two")
}

func TestDrainKeepsTransportsApart(t *testing.T) {
	store, sp, _ := drainFixture(t)

	_, err := sp.Write(capture(nil))
	require.NoError(t, err)
	_, err = sp.Write(capture(func(c *squirrel.Capture) { c.Transport = "matrix" }))
	require.NoError(t, err)

	squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second}).
		Once(context.Background())
	require.Equal(t, 2, countItems(t, store))
}

func TestDrainDefersWhileTheDatabaseIsUnreachable(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	unreachable, err := squirrel.OpenStore(context.Background(),
		"postgres://nobody:nobody@127.0.0.1:1/squirrel")
	require.NoError(t, err)
	defer unreachable.Close()

	got := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: unreachable, Interval: time.Second,
	}).Once(context.Background())

	require.Equal(t, 0, got.Inserted)
	require.Equal(t, 1, got.Deferred)

	names, err := sp.List()
	require.NoError(t, err)
	require.Len(t, names, 1)

	// And it lands once the database comes back.
	got = squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second}).
		Once(context.Background())
	require.Equal(t, 1, got.Inserted)
}

func TestDrainQuarantinesAnUnreadableFile(t *testing.T) {
	store, sp, dir := drainFixture(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "000000000000001-campfire-9.json"), []byte("{ not json"), 0o600))

	drain := squirrel.NewDrain(squirrel.DrainOptions{Spool: sp, Store: store, Interval: time.Second})

	got := drain.Once(context.Background())
	require.Equal(t, 1, got.Quarantined)
	require.Equal(t, 0, got.Inserted)

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)

	// And it is not retried.
	require.Equal(t, 0, drain.Once(context.Background()).Quarantined)
}

// flakyStore wraps a real Store and fails ResolvePerson — the first call
// Once makes on any capture, via d.one — the first failTimes times it is
// called, then delegates to the real Store from then on. Used to force
// deferred passes without needing an actually-unreachable Postgres for the
// whole test, so a later pass on the same Drain instance can succeed against
// the real database. Each call is timestamped, so the test can measure the
// real gaps between ticks instead of reaching into the drain's private
// backoff state.
type flakyStore struct {
	real      *squirrel.Store
	failTimes int

	mu    sync.Mutex
	calls []time.Time
}

func (f *flakyStore) ResolvePerson(ctx context.Context, transport string, externalID *string) (*int64, error) {
	f.mu.Lock()
	f.calls = append(f.calls, time.Now())
	n := len(f.calls)
	f.mu.Unlock()

	if n <= f.failTimes {
		return nil, errors.New("simulated transient failure")
	}
	return f.real.ResolvePerson(ctx, transport, externalID)
}

func (f *flakyStore) InsertItem(ctx context.Context, i squirrel.Item) (bool, error) {
	return f.real.InsertItem(ctx, i)
}

func (f *flakyStore) snapshot() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]time.Time, len(f.calls))
	copy(out, f.calls)
	return out
}

// The only drain mechanism that had no coverage in the Go port: Run must back
// off while the database is unreachable, and reset once it recovers, rather
// than either hammering Postgres on every tick or staying slow forever after
// one blip. flakyStore forces exactly two deferred passes before letting real
// ones through, so growth and reset both happen on one Drain instance.
//
// It asserts on the waits Run *chose*, reported through OnWait, rather than on
// the gaps between ticks as measured by a clock. Measuring the gaps timed the
// runner as much as the code: the assertion was loosened once (a half to three
// quarters) and flaked again anyway at 213ms against a 150ms threshold, on a
// docs-only pull request. There is no ratio that fixes that — a shared CI
// runner can stall for longer than the difference being measured, so the only
// honest fix is to stop measuring elapsed time.
//
// What is checked is now exact rather than approximate, which is a gain and not
// a compromise: the doubling is arithmetic and the reset is an assignment, so
// both have right answers.
func TestDrainRunGrowsBackoffOnDeferralAndResetsAfterSuccess(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	flaky := &flakyStore{real: store, failTimes: 2}

	var waitMu sync.Mutex
	waits := []time.Duration{}
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: flaky, Interval: 50 * time.Millisecond, MaxBackoff: time.Minute,
		OnWait: func(d time.Duration) {
			waitMu.Lock()
			waits = append(waits, d)
			waitMu.Unlock()
		},
	})
	waitsSoFar := func() []time.Duration {
		waitMu.Lock()
		defer waitMu.Unlock()
		return append([]time.Duration(nil), waits...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		drain.Run(ctx)
	}()

	// Ticks 1 and 2 both defer (flakyStore fails), doubling the backoff each
	// time; tick 3 succeeds, inserts the row, and resets the backoff. Wait for
	// the row too, not just the call count — the insert that follows a
	// successful resolve is still in flight the instant the call happens.
	require.Eventually(t, func() bool {
		return len(flaky.snapshot()) >= 3 && countItems(t, store) == 1
	}, 5*time.Second, 5*time.Millisecond)

	// Queue a second capture right after the reset so the next (short) tick
	// has something to process, making the reset directly observable rather
	// than inferred.
	_, err = sp.Write(capture(func(c *squirrel.Capture) { c.ExternalID = squirrel.Ptr("43") }))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(flaky.snapshot()) >= 4 && countItems(t, store) == 2
	}, 5*time.Second, 5*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}

	// The first three passes are the whole story, and every one of them is an
	// exact number rather than a range: the two deferred passes double the
	// base, and the clean pass assigns it straight back.
	waited := waitsSoFar()
	require.GreaterOrEqual(t, len(waited), 3)
	require.Equal(t, 100*time.Millisecond, waited[0], "a deferred pass doubles the base")
	require.Equal(t, 200*time.Millisecond, waited[1], "a second deferred pass doubles again")
	require.Equal(t, 50*time.Millisecond, waited[2], "a clean pass resets to the base")

	require.Equal(t, 2, countItems(t, store))
	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestDrainRunStopsWithTheContext(t *testing.T) {
	store, sp, _ := drainFixture(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		squirrel.NewDrain(squirrel.DrainOptions{
			Spool: sp, Store: store, Interval: 10 * time.Millisecond,
		}).Run(ctx)
	}()

	require.Eventually(t, func() bool { return countItems(t, store) == 1 },
		2*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}

// The rule is about what the applier is for: every branch of it ends in
// something said back, so with nowhere to say it there is nothing to run. The
// screen's slot is the case that made it matter — it spools like the room now,
// and "anything you type is a note" has always been the whole of what it does.
// Running Match over it would quietly turn the slot into a command line.
func TestDrainDoesNotApplyACaptureWithNoConversation(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	spool, err := squirrel.OpenSpool(t.TempDir())
	require.NoError(t, err)

	sender := "ronald"
	_, err = store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: squirrel.ScreenTransport, ExternalID: sender}})
	require.NoError(t, err)

	_, err = spool.Write(squirrel.Capture{
		Transport:  squirrel.ScreenTransport,
		SenderID:   &sender,
		Text:       "done 2",
		Payload:    []byte(squirrel.ScreenCapture),
		ReceivedAt: time.Now(),
	})
	require.NoError(t, err)

	chat, sent := chatRecorder("1")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Store: store, Spool: spool, Applier: applier,
	})
	drain.Once(ctx)

	// The thought landed, verbatim, and belongs to you.
	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "done 2", items[0].RawText)

	// And nothing was said back, because there was nowhere to say it.
	require.Empty(t, *sent, "the slot was read as a command")
}
