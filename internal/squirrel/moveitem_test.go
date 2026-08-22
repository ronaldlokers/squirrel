//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The conditional write, against the real column rather than a fake of it.
//
// The screen holds a tapped action for about a second and a half so the undo
// has a card to sit on, which means its write leaves after the decision — and
// for that window the row is untouched. This is the guard that stops a stale
// decision landing on top of one made in the room, and its whole subtlety is
// in one SQL predicate, so it is worth asking Postgres rather than a map.
func TestMoveItemState(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	fresh := func(t *testing.T) int64 {
		t.Helper()
		id, err := store.InsertItemReturningID(ctx, squirrel.Item{
			PersonID: &p, RawText: "the boiler makes a noise", ReceivedAt: time.Now(),
			Transport: squirrel.ScreenTransport, Payload: []byte(squirrel.ScreenCapture),
		})
		require.NoError(t, err)
		return id
	}
	stateOf := func(t *testing.T, id int64) squirrel.ItemState {
		t.Helper()
		it, found, err := store.ItemByID(ctx, p, id)
		require.NoError(t, err)
		require.True(t, found)
		return it.State
	}

	t.Run("moves a note that is where the caller last saw it", func(t *testing.T) {
		id := fresh(t)

		moved, err := store.MoveItemState(ctx, id, squirrel.ItemOpen, squirrel.ItemDone, time.Now())
		require.NoError(t, err)
		require.True(t, moved)
		require.Equal(t, squirrel.ItemDone, stateOf(t, id))
	})

	t.Run("refuses a note that has gone somewhere else", func(t *testing.T) {
		id := fresh(t)
		require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDropped, time.Now()))

		// The screen still believes it is open, because it was when the card
		// was drawn.
		moved, err := store.MoveItemState(ctx, id, squirrel.ItemOpen, squirrel.ItemDone, time.Now())
		require.NoError(t, err)
		require.False(t, moved)
		require.Equal(t, squirrel.ItemDropped, stateOf(t, id),
			"a stale decision overwrote the one that had already landed")
	})

	t.Run("a note already at the target is a success, not a collision", func(t *testing.T) {
		id := fresh(t)
		require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, time.Now()))

		// A second identical press, or a redelivered webhook. Saying a thing
		// twice is saying it; any variant of this that reports "already done"
		// turns a retry into a failure.
		moved, err := store.MoveItemState(ctx, id, squirrel.ItemOpen, squirrel.ItemDone, time.Now())
		require.NoError(t, err)
		require.True(t, moved)
		require.Equal(t, squirrel.ItemDone, stateOf(t, id))
	})

	t.Run("a note that does not exist is no move and no error", func(t *testing.T) {
		moved, err := store.MoveItemState(ctx, 999999, squirrel.ItemOpen, squirrel.ItemDone, time.Now())
		require.NoError(t, err)
		require.False(t, moved)
	})
}
