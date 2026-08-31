package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

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

	// The chip and what it draws agree, or the rename made it worse: the label
	// and the echo in the record are one string. See howYouFeltBefore.
	require.Contains(t, shownMoods(t, f), "how you felt before")
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
	deck := opened(t, f, "notes")
	for _, label := range []string{"DONE", "KEEP", "DROP", "A TASK", "something else?"} {
		require.Contains(t, deck, label, "the pile stopped saying %q", label)
	}
	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), ">Tell it<")
}
