//go:build integration

package squirrel_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// safeLog and captureLogs live in presence_test.go, not here: that file
// carries no build tag, so it compiles into both the plain and the
// integration build — a second copy here would collide with it under
// -tags=integration, since this file's own tag does not exclude presence_test.go.

// testDatabaseURL fails rather than skips, so an unset variable in CI is a
// failure and not a green run over nothing.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")
	return url
}

func withStore(t *testing.T) *squirrel.Store {
	t.Helper()
	ctx := context.Background()

	store, err := squirrel.OpenStore(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(store.Close)

	require.NoError(t, store.Migrate(ctx))
	truncateAll(t, store)
	return store
}

// truncateAll empties the database between tests. It names seven tables and
// relies on `cascade` for the rest — every other table has a foreign key to
// people, items or chores, so emptying those empties them all.
//
// That is a property of the schema rather than of this list, which is why
// TestTruncatingLeavesNothingBehind checks it: a table added without such a key
// would survive this and leak one test's rows into the next, and the symptom
// would be a test that passes alone and fails in the suite.
func truncateAll(t *testing.T, store *squirrel.Store) {
	t.Helper()
	_, err := store.Pool().Exec(context.Background(),
		`truncate table prompt_lines, prompts, events, items, chores, identities, people
		 restart identity cascade`)
	require.NoError(t, err)
}

// Every table is reached by truncateAll, which is a property of the schema
// rather than of the list it names: `cascade` reaches a table only through a
// foreign key, so a table added without one survives and leaks one test's rows
// into the next. The symptom is a test that passes alone and fails in a suite,
// which is among the worst failures to chase.
//
// Checked against the constraints rather than by inserting a row into every
// table, because a check that populates what it can reach proves nothing about
// the table it could not.
func TestTruncateAllReachesEveryTable(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	named := map[string]bool{
		"prompt_lines": true, "prompts": true, "events": true,
		"items": true, "chores": true, "identities": true, "people": true,
	}

	links := map[string][]string{}
	rows, err := store.Pool().Query(ctx, `
		select c.conrelid::regclass::text, c.confrelid::regclass::text
		from pg_constraint c
		join pg_class t on t.oid = c.conrelid
		join pg_namespace n on n.oid = t.relnamespace
		where c.contype = 'f' and n.nspname = 'public'`)
	require.NoError(t, err)
	for rows.Next() {
		var child, parent string
		require.NoError(t, rows.Scan(&child, &parent))
		links[child] = append(links[child], parent)
	}
	require.NoError(t, rows.Err())

	var all []string
	tables, err := store.Pool().Query(ctx, `
		select tablename from pg_tables
		where schemaname = 'public' and tablename <> 'schema_migrations'`)
	require.NoError(t, err)
	for tables.Next() {
		var name string
		require.NoError(t, tables.Scan(&name))
		all = append(all, name)
	}
	require.NoError(t, tables.Err())
	require.NotEmpty(t, all, "the schema has no tables, so this measured nothing")

	// Truncating a table cascades to whatever references it, and to whatever
	// references those, so a table is reached if any chain of foreign keys
	// leads from it to one of the seven.
	var reaches func(string, map[string]bool) bool
	reaches = func(table string, seen map[string]bool) bool {
		if named[table] {
			return true
		}
		if seen[table] {
			return false
		}
		seen[table] = true
		for _, parent := range links[table] {
			if reaches(parent, seen) {
				return true
			}
		}
		return false
	}

	for _, table := range all {
		require.True(t, reaches(table, map[string]bool{}),
			"%s has no foreign key leading to a table truncateAll names, so its rows outlive a test",
			table)
	}
	for name := range named {
		require.Contains(t, all, name, "truncateAll names %s and there is no such table", name)
	}
}
