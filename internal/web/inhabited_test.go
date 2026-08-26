package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// On Buddy, the lid said where you were by taking something away.
//
// The menu marks the place you are standing by keeping it in the list, filled
// violet and not a link. Buddy removed its icon instead — so it was the one
// screen that answered "where am I" with an absence, and removing it shifted
// the other two icons left, which moved the one piece of chrome that is meant
// to be identical on all thirteen screens.
// TestBuddyIsMarkedInTheLidRatherThanRemovedFromIt was retired on 25 August
// 2026 with the acorn it was about. Buddy is not a place you can be standing
// any more, so there is nothing in the lid to mark: `ask Buddy` is a chip on
// the live edge, and TestTheWayToBuddyIsOnTheLiveEdge is where it is pinned.

// Two different things were called by the same name on one screen.
//
// The mood history was "what you said before" and the pile's own door, one
// viewport below it, is "what you said". Notes and readings, named alike.
// Principle 7 wants this page reached by a name, and the name has to say which
// of the two it means.
func TestTheMoodHistoryIsNamedForFeelingRatherThanForSaying(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
	}
	m := mounted(t, f)

	post(t, m, "/mood", url.Values{"mood": {"calm"}})
	f.turns, f.appended = f.appended, nil

	thread := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, thread, "how you felt before")
	require.NotContains(t, thread, "what you said before",
		"the pile's door is the one that is about what you said")

	// The link and the page it opens agree, or the rename made it worse.
	require.Contains(t, m.call(t, "GET", "/moods", nil).Body.String(), "how you felt before")
}

// Control labels never vary, whatever the day.
//
// This is the boundary the whole experiment lives inside. Muscle memory is
// what Principle 6's "the same every time" protects: a sentence you read is
// worth varying, and a button you have learned to press without reading is
// not. A control that renames itself is a control you have to read again.
func TestNoControlRenamesItself(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
	}
	m := mounted(t, f)

	// The four answers and the slot's button: the words a hand learns.
	//
	// It was five until 26 August 2026, "make a chore" being the fifth. That
	// one is not on the card any more — the three questions moved behind
	// `something else?`, because seven equally-shaped buttons is six too many
	// on a screen whose premise is that deciding is expensive. What a hand
	// learns is now four verbs and one chip, and the chip is pinned below.
	deck := opened(t, f, "pile")
	for _, label := range []string{"DONE", "KEEP", "DROP", "A TASK", "something else?"} {
		require.Contains(t, deck, label, "the pile stopped saying %q", label)
	}
	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), ">Tell it<")
}
