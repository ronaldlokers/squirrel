package web

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheRoomsAreTheProductsOwnNames(t *testing.T) {
	want := map[string]string{
		"buddy":  "Buddy",
		"pile":   "the pile",
		"chores": "the chores",
		"at":     "the agenda",
		"tasks":  "the tasks",
		"held":   "what you set aside",
		"kept":   "the things you kept",
	}
	require.Len(t, rooms, len(want), "a room was added or removed without this test")
	for _, r := range rooms {
		name, ok := want[r.Key]
		require.True(t, ok, "unknown room %q", r.Key)
		require.Equal(t, name, r.Name)
		require.NotContains(t, r.Name, "#", "hash-names carry Slack's register")
		require.NotEmpty(t, r.Button, "%s has no button", r.Key)
		require.NotEmpty(t, r.Action, "%s posts nowhere", r.Key)
	}
}

// A room's dock is one control carrying the whole of what typing will do, so
// the same button on two rooms that file differently is the one failure this
// design cannot afford.
func TestNoTwoRoomsShareAButtonUnlessTheyShareADestination(t *testing.T) {
	where := map[string]string{}
	for _, r := range rooms {
		if seen, ok := where[r.Button]; ok {
			require.Equal(t, seen, r.Action,
				"%q means two different things", r.Button)
			continue
		}
		where[r.Button] = r.Action
	}
}

// Buddy's is the one button that names no room, because his room is where you
// talk rather than where a thing lands.
func TestOnlyBuddysButtonNamesNoPlace(t *testing.T) {
	for _, r := range rooms {
		if r.Key == "buddy" {
			require.Equal(t, "Tell it", r.Button)
			continue
		}
		require.True(t, strings.Contains(r.Button, "chore") ||
			strings.Contains(r.Button, "task") ||
			strings.Contains(r.Button, "pile") ||
			strings.Contains(r.Button, "agenda"),
			"%s's button %q does not say where the words go", r.Key, r.Button)
	}
}

func TestTheRoomIsBuddysWhenNobodySaidWhich(t *testing.T) {
	require.Equal(t, "buddy", roomOf(context.Background()))
}

func TestTheRoomOnTheRequestIsTheRoomThatIsRead(t *testing.T) {
	ctx := context.WithValue(context.Background(), roomKey{}, "chores")
	require.Equal(t, "chores", roomOf(ctx))
}
