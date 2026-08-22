package web

import (
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
func TestBuddyIsMarkedInTheLidRatherThanRemovedFromIt(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
	}
	m := mounted(t, f)

	elsewhere := m.call(t, "GET", "/pile", nil).Body.String()
	require.Contains(t, elsewhere, `class="lidbtn tobuddy"`)
	require.NotContains(t, elsewhere, "lidbtn tobuddy here")

	onBuddy := m.call(t, "GET", "/buddy", nil).Body.String()
	require.Contains(t, onBuddy, "lidbtn tobuddy here",
		"Buddy still says where you are by removing its own icon")
	require.Contains(t, onBuddy, `aria-current="page"`)
	require.NotContains(t, onBuddy, `href="/buddy?from=`,
		"the place you are standing is still offered as somewhere to go")
}

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

	home := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, home, "how you felt before")
	require.Contains(t, home, "what you said", "the pile's door is the one that says said")

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

	// The deck's five answers and the slot's button: the words a hand learns.
	deck := m.call(t, "GET", "/pile", nil).Body.String()
	for _, label := range []string{"DONE", "KEEP", "DROP", "A TASK", "MAKE A CHORE", "PUT IT BACK"} {
		require.Contains(t, deck, label, "the deck stopped saying %q", label)
	}
	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), ">Tell it<")
}
