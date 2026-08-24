package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// notEmpty has something on each screen this file visits, because an empty one
// renders the empty state and that is a different composition.
func notEmpty() *fakeStore {
	return &fakeStore{
		items: []squirrel.Item{
			note(1, "the boiler", squirrel.ItemOpen),
			note(2, "kaas", squirrel.ItemKept),
			task(3, "ring the vet", squirrel.ItemOpen),
		},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 8}},
		aside: []squirrel.HeldItem{{
			ID: 4, Text: "chase the vet", State: squirrel.ItemWaiting, Kind: squirrel.ItemTask,
		}},
		upcoming: []squirrel.Moment{{
			ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour),
			Travel: 15 * time.Minute, Ready: 10 * time.Minute,
		}},
	}
}

// Buddy says what it is for, on the screen it is for.
//
// It was an unlabelled acorn between two other unlabelled icons in the lid, so
// nothing anywhere said a conversation was available or what it would be about.
// A line under each screen's title says both.
func TestBuddyNamesTheScreenItIsOn(t *testing.T) {
	// A store with something on every screen: an empty one renders the empty
	// state instead, which is a different composition and deliberately has no
	// line — see TestAnEmptyScreenIsLeftAlone.
	f := notEmpty()
	for _, c := range []struct{ path, says string }{
		{"/pile", "ask Buddy about the pile"},
		{"/kept", "ask Buddy about the pile"},
		{"/tasks", "ask Buddy about the tasks"},
		{"/held", "ask Buddy about the tasks"},
		{"/chores", "ask Buddy about the chores"},
		{"/at", "ask Buddy about the agenda"},
	} {
		body := mounted(t, f).call(t, "GET", c.path, nil).Body.String()
		require.Contains(t, body, c.says, c.path)
		require.Contains(t, body, `href="/buddy`, c.path)
	}
}

// Home has no title, so the line goes above the doors — last before you pick a
// room, which is when not knowing where to start actually happens.
func TestHomeAsksAboveTheDoors(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "ask Buddy")
	require.Less(t, strings.Index(body, "ask Buddy"), strings.Index(body, `class="doors"`),
		"it belongs above the doors, not after them")
}

// A link to the page you are on is furniture, and the stopping screen is not a
// place to be offered a conversation about carrying on.
func TestBuddyIsAbsentWhereItWouldBeWrong(t *testing.T) {
	for _, path := range []string{"/buddy", "/enough"} {
		body := mounted(t, &fakeStore{}).call(t, "GET", path, nil).Body.String()
		require.NotContains(t, body, "ask Buddy", path)
	}
}

// The lid's map says which room you are in, and the agenda is a room.
func TestTheAgendaKnowsWhereItIs(t *testing.T) {
	require.Equal(t, "the agenda", placeName("at"))
}

// An empty screen is a whole statement and is left to make it.
//
// The empty states have a treatment of their own — the mascot, the Headline
// role, and a line offering the two things worth doing instead. A third offer
// on a screen whose job is saying "nothing here, and that is fine" argues with
// the screen.
func TestAnEmptyScreenIsLeftAlone(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "nothing in the pile", "this is the empty state")
	require.NotContains(t, body, "ask Buddy")
}
