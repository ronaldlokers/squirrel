//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Seven rooms became five, and nothing that was said is lost.
//
// The pile, the things you kept and what you set aside are one room now, so
// three conversations become one; Buddy's room was named for the speaker and is
// named for what it holds. Both are one migration, and this is the proof that
// it moves rows rather than merely running.
func TestTheOldRoomsConversationsAreInTheNewOnes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// Written the way the screen wrote them before 31 August 2026. The column
	// takes any text, so the old keys still go in — which is what makes this a
	// test of the migration rather than of the constraint.
	for _, said := range []struct{ room, words string }{
		{"pile", "the boiler code"},
		{"kept", "the wifi password"},
		{"held", "chase the landlord"},
		{"buddy", "how do you feel?"},
		{"chores", "bins out"},
	} {
		_, err := store.AppendTurn(ctx, p, said.room, squirrel.Turn{
			Who: squirrel.SpeakerYou, Words: said.words,
		})
		require.NoError(t, err)
	}

	// Rewound so the migration runs for the first time over rows that are
	// already there, which is the only ordering it will ever meet: it runs at
	// deploy, against a record written by the version before it. Migrate is
	// otherwise a no-op the second time and this would prove nothing.
	_, err := store.Pool().Exec(ctx,
		`delete from schema_migrations where version like '%0035%'`)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(ctx))

	notes, _, err := store.RecentTurns(ctx, p, "notes", 20)
	require.NoError(t, err)
	require.Len(t, notes, 3, "the three that became the notes are not all there")
	require.Equal(t, []string{"the boiler code", "the wifi password", "chase the landlord"},
		[]string{notes[0].Words, notes[1].Words, notes[2].Words},
		"the notes lost the order the three were said in")

	everything, _, err := store.RecentTurns(ctx, p, "everything", 20)
	require.NoError(t, err)
	require.Len(t, everything, 1)
	require.Equal(t, "how do you feel?", everything[0].Words)

	chores, _, err := store.RecentTurns(ctx, p, "chores", 20)
	require.NoError(t, err)
	require.Len(t, chores, 1, "a room that did not change lost a turn")

	// And nothing is left behind in a room the screen can no longer open.
	for _, gone := range []string{"pile", "kept", "held", "buddy"} {
		left, _, err := store.RecentTurns(ctx, p, gone, 20)
		require.NoError(t, err)
		require.Empty(t, left, "%s still holds a conversation nothing can reach", gone)
	}
}
