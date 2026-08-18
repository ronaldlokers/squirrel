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

func truncateAll(t *testing.T, store *squirrel.Store) {
	t.Helper()
	_, err := store.Pool().Exec(context.Background(),
		`truncate table prompt_lines, prompts, events, items, chores, identities, people
		 restart identity cascade`)
	require.NoError(t, err)
}
