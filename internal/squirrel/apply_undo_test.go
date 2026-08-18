//go:build integration

package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The screen can put a note back; chat could not. The mechanism was already
// there — every transition reverses — so this was a missing word rather than a
// missing capability.
func TestUndoPutsTheLastTriagedNoteBack(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk")
	id := lineItemID(t, store, p, 1)
	triage(t, store, p, "done 1")
	require.Equal(t, "done", stateOf(t, store, id))

	reply := triage(t, store, p, "!undo")

	require.Contains(t, reply, "buy milk")
	require.Equal(t, "open", stateOf(t, store, id))
}

// It undoes the most recent one, not the one you happen to be looking at.
func TestUndoTakesTheMostRecentTriage(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk", "the boiler")
	first := lineItemID(t, store, p, 1)
	second := lineItemID(t, store, p, 2)

	triage(t, store, p, "drop 2")
	triage(t, store, p, "done 1")
	triage(t, store, p, "!undo")

	require.Equal(t, "open", stateOf(t, store, first), "the one triaged last comes back")
	require.Equal(t, "dropped", stateOf(t, store, second), "the one before it stays where it was")
}

// Said again, it walks back another step. Each undo is itself a transition, so
// there is nothing special about doing two.
func TestUndoAgainWalksBackAgain(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk", "the boiler")
	first := lineItemID(t, store, p, 1)
	second := lineItemID(t, store, p, 2)

	triage(t, store, p, "drop 2")
	triage(t, store, p, "done 1")
	triage(t, store, p, "!undo")
	triage(t, store, p, "!undo")

	require.Equal(t, "open", stateOf(t, store, first))
	require.Equal(t, "open", stateOf(t, store, second))
}

func TestUndoWithNothingToUndo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk")

	reply := triage(t, store, p, "!undo")

	require.Contains(t, reply, "nothing")
	require.Equal(t, "open", stateOf(t, store, lineItemID(t, store, p, 1)))
}

// A command is not a note, on this path as on every other: undoing must not
// resurrect the "!notes" someone typed.
func TestUndoIgnoresCommands(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk")
	id := lineItemID(t, store, p, 1)
	triage(t, store, p, "done 1")
	// "!undo" itself lands in items like every other inbound message.
	triage(t, store, p, "!undo")

	require.Equal(t, "open", stateOf(t, store, id))
}

func TestHelpMentionsUndo(t *testing.T) {
	require.Contains(t, squirrel.HelpMessage().Text, "!undo")
}
