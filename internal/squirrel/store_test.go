//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func countItems(t *testing.T, store *squirrel.Store) int {
	t.Helper()
	var n int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from items`).Scan(&n))
	return n
}

func item() squirrel.Item {
	return squirrel.Item{
		Transport:  "campfire",
		RawText:    "",
		Payload:    []byte(`{"unparseable":"nonsense"}`),
		ReceivedAt: time.Date(2026, 8, 14, 9, 31, 4, 512_000_000, time.UTC),
	}
}

func TestMigrateCreatesTheThreeTables(t *testing.T) {
	store := withStore(t)

	for _, name := range []string{"people", "identities", "items"} {
		var exists bool
		require.NoError(t, store.Pool().QueryRow(context.Background(),
			`select exists (select 1 from information_schema.tables
			   where table_schema = 'public' and table_name = $1)`, name,
		).Scan(&exists))
		require.True(t, exists, name)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	store := withStore(t)
	require.NoError(t, store.Migrate(context.Background()))
}

func TestInsertItemStoresNullsInEveryFailOpenColumn(t *testing.T) {
	store := withStore(t)
	inserted, err := store.InsertItem(context.Background(), item())
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, 1, countItems(t, store))
}

// The unique index is partial, so several null-external-id rows coexist.
func TestInsertItemAllowsManyNullExternalIDs(t *testing.T) {
	store := withStore(t)
	first, err := store.InsertItem(context.Background(), item())
	require.NoError(t, err)
	require.True(t, first)
	second, err := store.InsertItem(context.Background(), item())
	require.NoError(t, err)
	require.True(t, second)
	require.Equal(t, 2, countItems(t, store))
}

// If the ON CONFLICT predicate did not match the partial index, Postgres
// would reject the statement outright rather than dedupe.
func TestInsertItemDedupesARealExternalID(t *testing.T) {
	store := withStore(t)

	withID := item()
	withID.ExternalID = squirrel.Ptr("42")

	first, err := store.InsertItem(context.Background(), withID)
	require.NoError(t, err)
	require.True(t, first, "the first delivery inserts a row")

	second, err := store.InsertItem(context.Background(), withID)
	require.NoError(t, err)
	require.False(t, second, "the redelivery is absorbed by ON CONFLICT, not inserted")

	require.Equal(t, 1, countItems(t, store))
}

func TestInsertItemKeepsTransportsApart(t *testing.T) {
	store := withStore(t)

	campfire := item()
	campfire.ExternalID = squirrel.Ptr("42")
	matrix := campfire
	matrix.Transport = "matrix"

	first, err := store.InsertItem(context.Background(), campfire)
	require.NoError(t, err)
	require.True(t, first)
	second, err := store.InsertItem(context.Background(), matrix)
	require.NoError(t, err)
	require.True(t, second)
	require.Equal(t, 2, countItems(t, store))
}
