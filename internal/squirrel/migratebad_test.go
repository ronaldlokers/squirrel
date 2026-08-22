//go:build integration

package squirrel

import (
	"context"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// storeForMigrations is the internal-package equivalent of withStore: these
// tests reach for `migrateFrom` and the pool, which the external test package
// cannot see. It never writes a row, so it shares the database happily.
func storeForMigrations(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")

	store, err := OpenStore(context.Background(), url)
	require.NoError(t, err)
	t.Cleanup(store.Close)
	require.NoError(t, store.Migrate(context.Background()))
	return store
}

// A migration that will not apply says so, against a real database.
//
// Boot retries a bad migration and an unreachable database identically — on
// purpose, because nothing there may block a capture being accepted — so the
// log is the only thing that tells them apart, and it used to call both
// "database unavailable". One of those means "roll the image back" and the
// other means "wait".
//
// This runs the real migrator with a real Postgres and a migration that cannot
// apply, rather than constructing the error by hand and asserting on what the
// test itself just wrote.
func TestAMigrationThatWillNotApplySaysWhichOne(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)

	err := store.migrateFrom(ctx, fstest.MapFS{
		"migrations/9001_broken.sql": {Data: []byte("this is not sql at all;")},
	})

	require.Error(t, err)
	require.True(t, IsMigrationFailure(err),
		"a bad migration reported as an unreachable database sends you to the wrong runbook: %v", err)
	require.Contains(t, err.Error(), "9001_broken.sql",
		"the log has to name the file, or the runbook's first step cannot be taken")
}

// And it leaves the schema exactly where it was: each migration applies inside
// a transaction that also records it, so there is no half-applied state to
// reason about at two in the morning.
func TestAFailedMigrationRecordsNothing(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)

	_ = store.migrateFrom(ctx, fstest.MapFS{
		"migrations/9002_broken.sql": {Data: []byte("create table zz (); oops")},
	})

	var recorded bool
	require.NoError(t, store.pool.QueryRow(ctx,
		`select exists (select 1 from schema_migrations where version = $1)`,
		"migrations/9002_broken.sql").Scan(&recorded))
	require.False(t, recorded, "a migration that failed was recorded as applied")
}
