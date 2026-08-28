package coach

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// What a room costs, measured rather than argued.
//
// The spec asked for this before rooms reached production, as a check on the
// €10 ceiling rather than a gate on the feature. Both directions are real: a
// narrower toolset is fewer schemas in every request, and a per-room window is
// more windows carrying less each.
//
// Measured 28 August 2026, serialising the tools exactly as they go on the
// wire plus the room's own preamble line. Bytes are exact; ~tokens is bytes/4,
// which is an approximation and is only used to put the number in the same
// unit as the ceiling.
//
//	room                 tools  bytes  ~tokens  vs Buddy
//	Buddy                   15   4756     1189       —
//	the chores               9   3390      847   -28.7%
//	the tasks               11   3371      842   -29.1%
//	the pile                 8   3096      774   -34.9%
//	the agenda               5   2234      558   -53.0%
//	the things you kept      4   1463      365   -69.2%
//	what you set aside       4   1461      365   -69.3%
//
// At the routine tier's 20 cents per million input tokens, the largest saving
// — 824 tokens in a shelf — is about €0.00016 a request. Real, and small: this
// is not a feature that pays for itself, and it was never meant to be. What
// the number establishes is the direction, which the two tests below hold.
func TestARoomNeverCostsMoreThanBuddysOwn(t *testing.T) {
	buddy, err := json.Marshal(toolsFor("buddy", true))
	require.NoError(t, err)
	base := len(buddy) + len(inTheRoom("buddy"))

	for _, room := range RoomKeys() {
		t.Run(room, func(t *testing.T) {
			specs, err := json.Marshal(toolsFor(room, true))
			require.NoError(t, err)
			// The preamble line is part of what a room costs. A narrowing that
			// saved on schemas and spent it back on prose would net to nothing
			// and nobody would see it.
			got := len(specs) + len(inTheRoom(room))
			require.Less(t, got, base,
				"%s sends more than Buddy's own room, which is the room that is not narrowed", room)
		})
	}
}

// And the window carries no more than it carried before rooms.
//
// This is the direction that could have gone the other way. Keying the window
// by (person, room) means more windows — but each request carries one of them,
// and a room's own is a subset of what the single window held. Per request it
// can only shrink.
//
// What it cannot measure is the behaviour: losing the thread when you change
// rooms could make you repeat yourself, which is more turns rather than bigger
// ones. That needs use, not a test.
func TestARoomsWindowIsNoBiggerThanTheOneItReplaced(t *testing.T) {
	at := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	c := NewConversations()

	// The same traffic a single window would have held: WindowSize exchanges
	// in each of three rooms, all for one person.
	for i := 0; i < WindowSize; i++ {
		for _, room := range []string{"buddy", "chores", "pile"} {
			c.Add(1, room, "said", "replied", at)
		}
	}

	for _, room := range []string{"buddy", "chores", "pile"} {
		require.LessOrEqual(t, len(c.Recent(1, room, at)), WindowSize,
			"%s carries more than one window's worth", room)
	}
}
