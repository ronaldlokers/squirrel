//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The whole point: a note promoted to a task leaves the pile. If it did not,
// this would be a label rather than a kind.
func TestATaskLeavesThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet")
	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)

	moved, err := store.SetItemKind(ctx, p, items[0].ID, squirrel.ItemTask)
	require.NoError(t, err)
	require.True(t, moved)

	pile, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, pile, "it is not a thought any more")

	tasks, _, err := store.Tasks(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "ring the vet", tasks[0].RawText)
}

// Deciding was the mistake, and undoing a decision must not require finishing
// it.
func TestATaskCanBecomeANoteAgain(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet")
	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)

	_, err = store.SetItemKind(ctx, p, items[0].ID, squirrel.ItemTask)
	require.NoError(t, err)
	_, err = store.SetItemKind(ctx, p, items[0].ID, squirrel.ItemNote)
	require.NoError(t, err)

	pile, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, pile, 1, "the same row, back where it was")
	require.Equal(t, items[0].ID, pile[0].ID)
}

func TestTheArchiveHoldsOnlyTasksThatAreDone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet", "a thought that got done")
	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Newest first, so items[0] is "a thought that got done".
	require.NoError(t, store.SetItemState(ctx, items[0].ID, squirrel.ItemDone, time.Now()))

	_, err = store.SetItemKind(ctx, p, items[1].ID, squirrel.ItemTask)
	require.NoError(t, err)
	require.NoError(t, store.SetItemState(ctx, items[1].ID, squirrel.ItemDone, time.Now()))

	archived, _, err := store.ArchivedTasks(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, archived, 1)
	require.Equal(t, "ring the vet", archived[0].RawText)

	open, _, err := store.Tasks(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, open, "done is not still to do")
}

// Everything that already existed keeps working: a row with no kind said is a
// note, which every existing row truthfully is.
func TestEveryExistingRowIsANote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")
	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, squirrel.ItemNote, items[0].Kind)
}

// Skipping past a note must not land on a task: the cursor walks the pile, and
// a task is not in it.
func TestSkippingStaysInThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "first", "second", "third")
	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 3)

	_, err = store.SetItemKind(ctx, p, items[1].ID, squirrel.ItemTask)
	require.NoError(t, err)

	after, _, err := store.OpenItemsAfter(ctx, p, items[0].ID, 10)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, items[2].ID, after[0].ID, "past the one that became a task")
}
