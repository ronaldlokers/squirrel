package web

import (
	"testing"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

// Buddy's room draws what was said, not what was said in it: four rooms stopped
// being places and their rows kept the room they were written in, so a read
// scoped to one room is a record with holes in it.
func TestBuddysRoomReadsTheWholeRecord(t *testing.T) {
	f := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Room: "chores", Who: squirrel.SpeakerYou, Words: "bins on Tuesday"},
		{ID: 2, Room: "everything", Who: squirrel.SpeakerYou, Words: "kaas"},
	}}
	m := mounted(t, f)

	body := m.call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "bins on Tuesday", "what was said in another room is unreachable")
	require.Contains(t, body, "kaas")
	require.Equal(t, "everything said", f.roomRead, "the read is still scoped to a room")
}
