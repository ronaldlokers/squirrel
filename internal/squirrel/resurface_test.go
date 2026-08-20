//go:build integration

package squirrel_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A kept note may come back — only riding along with a message that was going
// out anyway, and never as its own stream.

func TestAKeptNoteRidesAlongWithTheEvening(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{}, []string{"a thought"}, nil,
		"the boiler serial is 44Q")

	require.Contains(t, m.Text, "the boiler serial is 44Q")
	require.Contains(t, m.Text, "You kept this")
}

// Never its own message. With nothing else to say, the evening is silent and a
// kept note is not a reason to break that.
func TestAKeptNoteIsNeverTheWholeMessage(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{}, nil, nil, "the boiler serial is 44Q")
	require.Empty(t, m.Text, "the shelf spoke on its own")
}

// Never more than one. A list of kept notes is the shelf becoming an inbox.
func TestOnlyOneKeptNoteEverRidesAlong(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{}, []string{"a thought"}, nil,
		"the boiler serial is 44Q")
	require.Equal(t, 1, strings.Count(m.Text, "You kept this"))
}

// Last, and unprompted: it is not a task, it is not due, and nothing is being
// asked of you.
func TestTheKeptNoteComesAfterEverythingElse(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{}, []string{"a thought"}, nil,
		"the boiler serial is 44Q")

	require.Less(t, strings.Index(m.Text, "a thought"), strings.Index(m.Text, "You kept this"))
}

// It carries no buttons of its own. The evening's actions belong to the chore
// it raised, and a kept note is not something to answer.
func TestTheKeptNoteAddsNoButtons(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{}, []string{"a thought"}, nil,
		"the boiler serial is 44Q")
	require.Empty(t, m.Actions)
}

// Random rather than oldest-first: a queue would give the shelf a front, and a
// front is a place to be behind.
func TestTheShelfHandsBackOneOfWhatIsOnIt(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "the boiler serial is 44Q", "the meter reading is 48213")
	for i := 1; i <= 2; i++ {
		id := lineItemID(t, store, p, i)
		require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemKept, time.Now()))
	}

	text, found, err := store.AKeptItem(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Contains(t, []string{"the boiler serial is 44Q", "the meter reading is 48213"}, text)
}

func TestAnEmptyShelfHandsBackNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	_, found, err := store.AKeptItem(context.Background(), p)
	require.NoError(t, err)
	require.False(t, found)
}

// Only kept notes. A thing in the pile has not been kept, and a thing that was
// dropped was dropped.
func TestOnlyTheShelfResurfaces(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "still untriaged")
	id := lineItemID(t, store, p, 1)
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDropped, time.Now()))

	_, found, err := store.AKeptItem(ctx, p)
	require.NoError(t, err)
	require.False(t, found)
}
