//go:build integration

package squirrel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestAppendTurnComesBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	got, err := store.AppendTurn(ctx, p, "buddy", squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "at 14:30 dentist",
	})
	require.NoError(t, err)
	require.NotZero(t, got.ID)

	turns, more, err := store.RecentTurns(ctx, p, "buddy", 10)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, turns, 1)
	require.Equal(t, "at 14:30 dentist", turns[0].Words)
	require.Equal(t, squirrel.SpeakerYou, turns[0].Who)
}

// The order the thread reads in. A newest-first slice would render the
// conversation backwards, and a test that only counted rows would not notice.
func TestRecentTurnsAreOldestFirst(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for _, w := range []string{"first", "second", "third"} {
		_, err := store.AppendTurn(ctx, p, "buddy", squirrel.Turn{Who: squirrel.SpeakerYou, Words: w})
		require.NoError(t, err)
	}

	turns, _, err := store.RecentTurns(ctx, p, "buddy", 10)
	require.NoError(t, err)
	require.Len(t, turns, 3)
	require.Equal(t, []string{"first", "second", "third"},
		[]string{turns[0].Words, turns[1].Words, turns[2].Words})
}

// The cap keeps the newest, not the oldest — a limit applied before the sort
// would show the beginning of the conversation forever.
func TestRecentTurnsKeepsTheNewestAndSaysThereIsMore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for i := 0; i < 5; i++ {
		_, err := store.AppendTurn(ctx, p, "buddy",
			squirrel.Turn{Who: squirrel.SpeakerYou, Words: fmt.Sprintf("turn %d", i)})
		require.NoError(t, err)
	}

	turns, more, err := store.RecentTurns(ctx, p, "buddy", 2)
	require.NoError(t, err)
	require.True(t, more)
	require.Len(t, turns, 2)
	require.Equal(t, []string{"turn 3", "turn 4"}, []string{turns[0].Words, turns[1].Words})
}

func TestTurnsBeforePagesBackwards(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	var ids []int64
	for i := 0; i < 5; i++ {
		got, err := store.AppendTurn(ctx, p, "buddy",
			squirrel.Turn{Who: squirrel.SpeakerYou, Words: fmt.Sprintf("turn %d", i)})
		require.NoError(t, err)
		ids = append(ids, got.ID)
	}

	turns, more, err := store.TurnsBefore(ctx, p, "buddy", ids[3], 2)
	require.NoError(t, err)
	require.True(t, more)
	require.Len(t, turns, 2)
	require.Equal(t, []string{"turn 1", "turn 2"}, []string{turns[0].Words, turns[1].Words})
}

// Shown is stored as written and is never re-read from another table. This is
// the whole of "history is never rewritten" at the storage layer.
func TestShownIsKeptVerbatim(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.AppendTurn(ctx, p, "buddy", squirrel.Turn{
		Who: squirrel.SpeakerBuddy, Words: "Two are due.",
		Shown: []byte(`{"cards":[{"title":"water the plants"}]}`),
	})
	require.NoError(t, err)

	turns, _, err := store.RecentTurns(ctx, p, "buddy", 10)
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.JSONEq(t, `{"cards":[{"title":"water the plants"}]}`, string(turns[0].Shown))
}

// A turn that drew nothing reads back as nothing, rather than as an empty
// document. "There was no card" and "there was a card with no fields" are
// different facts about the turn.
func TestATurnThatDrewNothingHasNoShown(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.AppendTurn(ctx, p, "buddy", squirrel.Turn{Who: squirrel.SpeakerYou, Words: "milk"})
	require.NoError(t, err)

	turns, _, err := store.RecentTurns(ctx, p, "buddy", 10)
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Nil(t, turns[0].Shown)
}

// Another person's conversation is not yours.
func TestRecentTurnsAreScopedToThePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	other, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)
	require.NotEqual(t, p, other)

	_, err = store.AppendTurn(ctx, other, "buddy", squirrel.Turn{Who: squirrel.SpeakerYou, Words: "not yours"})
	require.NoError(t, err)

	turns, _, err := store.RecentTurns(ctx, p, "buddy", 10)
	require.NoError(t, err)
	require.Empty(t, turns)
}

// A room is a scope, not a label. Without the where-clause every room reads
// the whole record, which looks from the screen like one conversation that
// will not partition.
func TestRecentTurnsAreScopedToTheRoom(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.AppendTurn(ctx, p, "pile", squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "in the pile",
	})
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, p, "chores", squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "in the chores",
	})
	require.NoError(t, err)

	turns, _, err := store.RecentTurns(ctx, p, "pile", 10)
	require.NoError(t, err)
	require.Len(t, turns, 1, "the chores' turn came back from the pile")
	require.Equal(t, "in the pile", turns[0].Words)
	require.Equal(t, "pile", turns[0].Room)
}

// The walk back has to step over another room's turns rather than through
// them: the cursor is a time and an id, and both are shared across rooms.
func TestTurnsBeforeAreScopedToTheRoom(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	first, err := store.AppendTurn(ctx, p, "pile", squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "one",
	})
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, p, "chores", squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "elsewhere",
	})
	require.NoError(t, err)
	second, err := store.AppendTurn(ctx, p, "pile", squirrel.Turn{
		Who: squirrel.SpeakerYou, Words: "two",
	})
	require.NoError(t, err)

	turns, _, err := store.TurnsBefore(ctx, p, "pile", second.ID, 10)
	require.NoError(t, err)
	require.Len(t, turns, 1, "the walk back left the room")
	require.Equal(t, first.ID, turns[0].ID)
}
