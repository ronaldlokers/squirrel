//go:build integration

package squirrel_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

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

// chatFor adapts a phase 2 Sender — text only, no id back — into a Chat whose
// Send forwards a Message's text through it. It exists so tests written
// before buttons existed, which build an Applier or a Scheduler around a
// plain recorder(), keep exercising the same reply text now that both send
// through Chat rather than Sender. Update and Boost are left nil; a test that
// needs those builds its own Chat directly.
func chatFor(send squirrel.Sender) squirrel.Chat {
	return squirrel.Chat{
		Send: func(ctx context.Context, conversationID string, m squirrel.Message) (string, error) {
			return "", send(ctx, conversationID, m.Text)
		},
	}
}
