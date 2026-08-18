//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Every row that existed before 0008 becomes `open`, which is true: nothing had
// ever been triaged, because there was nothing to triage it into.
func TestExistingItemsStartOpen(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "a thought from before")

	var state string
	var stateAt *time.Time
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select state, state_at from items where id = $1`, id).Scan(&state, &stateAt))

	require.Equal(t, "open", state)
	require.Nil(t, stateAt, "state_at stays null until something actually happens to the note")
}

// The vocabulary is closed in the database, not only in Go. The constants and a
// hand-written UPDATE are two different doors into the same table, and a
// migration that only Go respects is a migration that stops being true the
// first time someone opens psql.
func TestItemStateRejectsAnUnknownValue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "a thought")

	_, err := store.Pool().Exec(ctx,
		`update items set state = 'archived' where id = $1`, id)
	require.Error(t, err, "the check constraint is what keeps the state vocabulary closed")
}

func TestItemStateAcceptsEveryKnownValue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "a thought")

	for _, state := range []string{"open", "done", "dropped", "kept"} {
		_, err := store.Pool().Exec(ctx,
			`update items set state = $2 where id = $1`, id, state)
		require.NoError(t, err, "state %q is part of the vocabulary and must be accepted", state)
	}
}
