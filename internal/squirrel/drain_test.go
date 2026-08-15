//go:build integration

package squirrel_test

import (
	"context"
	"os"
	"path/filepath"
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
