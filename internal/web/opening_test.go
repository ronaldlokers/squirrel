package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy says the first thing, when there is a first thing worth saying.

// Nothing worth saying is the common case, and it is silence rather than a
// greeting. A line that arrives whether or not anything is happening is
// wallpaper by the third day.
func TestOnAQuietAfternoonBuddySaysNothingFirst(t *testing.T) {
	f := &fakeStore{checkin: fresh()}
	body := thread(t, f)

	for _, greeting := range []string{"Hello", "Welcome back", "Good morning", "Anything else"} {
		require.NotContains(t, body, greeting)
	}
	require.Empty(t, f.appended, "it said something with nothing to say")
}

// Something with a time you can be late for is the thing that most wants
// saying, because it will happen whether or not you look.
func TestSomethingTodayIsWhatBuddyOpensWith(t *testing.T) {
	at := now().Add(3 * time.Hour)
	f := &fakeStore{
		checkin:  fresh(),
		items:    []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting:  squirrel.Waiting{Pile: 1, Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: at}},
	}
	body := thread(t, f)

	require.Contains(t, body, "dentist today")
	require.Contains(t, body, at.Format("15:04"))
	require.Contains(t, body, "the agenda", "it mentioned it and gave no way to it")
}

// Tomorrow is said as tomorrow. Reading "dentist today" the night before is
// the one way this line could make you leave the house.
func TestSomethingTomorrowSaysTomorrow(t *testing.T) {
	f := &fakeStore{
		checkin:  fresh(),
		waiting:  squirrel.Waiting{Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: squirrel.StartOfDay(now()).Add(34 * time.Hour)}},
	}

	require.Contains(t, thread(t, f), "dentist tomorrow")
}

// And something next week is not mentioned unprompted. It is on the agenda,
// which is where you go to look at it.
func TestSomethingNextWeekIsNotOpenedWith(t *testing.T) {
	f := &fakeStore{
		checkin:  fresh(),
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(9 * 24 * time.Hour)}},
	}

	require.NotContains(t, thread(t, f), "dentist")
}

// The world before your own initiative: a chore came back on its own, and the
// pile is a thing you chose to put off.
func TestWhatCameBackIsSaidBeforeThePile(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting: squirrel.Waiting{Pile: 1, Chores: 1},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Active: true,
			Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 9, EverDone: true}},
	}
	body := thread(t, f)

	require.Contains(t, body, "come back round")
	require.NotContains(t, body, "not decided about")
}

// And the same sentence is not said twice.
//
// This is the defect the offer had for an afternoon: appended on every load,
// so a reload put a second copy in the record. The offer refuses to talk over
// an open turn, which is not enough here — this speaks when nothing is open.
func TestBuddyDoesNotOpenWithTheSameThingTwice(t *testing.T) {
	f := &fakeStore{
		checkin:  fresh(),
		waiting:  squirrel.Waiting{Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
	}
	m := routed(t, f)

	m.call(t, "GET", "/", nil)
	require.Len(t, f.appended, 1, "it did not open at all")
	f.turns, f.appended = append(f.turns, f.appended...), nil

	m.call(t, "GET", "/", nil)
	require.Empty(t, f.appended, "it said the same thing again")
}

// But it does open again once what it would say has changed.
func TestBuddyOpensAgainWhenSomethingElseIsTrue(t *testing.T) {
	f := &fakeStore{
		checkin:  fresh(),
		waiting:  squirrel.Waiting{Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
	}
	m := routed(t, f)

	m.call(t, "GET", "/", nil)
	f.turns, f.appended = append(f.turns, f.appended...), nil

	f.upcoming = []squirrel.Moment{{ID: 2, Label: "the school run", Starts: now().Add(4 * time.Hour)}}
	m.call(t, "GET", "/", nil)

	require.Len(t, f.appended, 1, "it stayed quiet about something new")
	require.Contains(t, f.appended[0].Words, "the school run")
}

// Never over a question. Buddy does not talk over himself, and an opening line
// on top of an unanswered picker is exactly that.
func TestBuddyDoesNotOpenOverAQuestion(t *testing.T) {
	f := &fakeStore{
		checkin:  fresh(),
		waiting:  squirrel.Waiting{Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
		turns: []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "How often?",
			Shown: []byte(`{"pick":{"action":"/chores/act","do":"that's it","rows":[]}}`)}},
	}
	thread(t, f)

	require.Empty(t, f.appended)
}

// It is one thing, not a summary of all four. The rail already says all four.
func TestTheOpeningLineIsOneThing(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting: squirrel.Waiting{Pile: 1, Chores: 1, Agenda: 1},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Active: true,
			Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 9, EverDone: true}},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
	}
	thread(t, f)

	require.Len(t, f.appended, 1)
	said := f.appended[0].Words
	require.Contains(t, said, "dentist")
	require.NotContains(t, said, "come back round")
	require.NotContains(t, said, "pile")
	require.Less(t, len(said), 90, "it is a paragraph, not an opening")
}

// A count that cannot be read is a line not drawn. You came to talk, not to be
// told the database is unwell.
func TestAStoreThatCannotCountOpensWithNothing(t *testing.T) {
	f := &fakeStore{checkin: fresh(), err: errTest}
	m := routed(t, f)
	w := m.call(t, "GET", "/", nil)

	require.NotContains(t, strings.ToLower(w.Body.String()), "cannot count")
}
