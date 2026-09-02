//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

// Buddy's room reads the whole record rather than one room's share of it, so
// what was said in a room that is no longer a place is still reachable. The
// rows keep the room they were said in; the reading stops asking.
func TestEverythingBeforeWalksBackAcrossRooms(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	for i, said := range []struct{ room, words string }{
		{"notes", "the boiler code"},
		{"chores", "bins on Tuesday"},
		{"everything", "kaas"},
		{"tasks", "ring the vet"},
	} {
		_, err := store.AppendTurn(ctx, p, said.room, squirrel.Turn{
			Who: squirrel.SpeakerYou, Words: said.words, SaidAt: at.Add(time.Duration(i) * time.Minute),
		})
		require.NoError(t, err)
	}

	all, _, err := store.EverythingSaid(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, all, 4, "the record is not one conversation")

	newest := all[len(all)-1]
	require.Equal(t, "ring the vet", newest.Words)

	older, more, err := store.EverythingBefore(ctx, p, newest.ID, 2)
	require.NoError(t, err)
	require.False(t, more == false && len(older) != 2, "the walk back did not read across rooms")
	require.Equal(t, []string{"bins on Tuesday", "kaas"}, []string{older[0].Words, older[1].Words})
}
