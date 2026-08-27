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
	// A fixed morning, not now().Add(3h).
	//
	// Three hours from now is tomorrow after nine in the evening, so this test
	// asserted "dentist today" against a card correctly reading "dentist
	// tomorrow" — and failed every night between 21:00 and midnight, on any
	// branch, for a reason having nothing to do with anybody's change. The
	// sibling test below already pins tomorrow explicitly; this one was
	// relying on the clock it was run at.
	was := now
	now = func() time.Time { return time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = was })

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

// It is one thing, not a summary of all four. The menu already says all four.
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

func TestAStoreThatCannotCountOpensWithNothing(t *testing.T) {
	f := &fakeStore{checkin: fresh(), err: errTest}
	m := routed(t, f)
	w := m.call(t, "GET", "/", nil)

	require.NotContains(t, strings.ToLower(w.Body.String()), "cannot count")
}

// The day boundary is the person's, not the process's.
//
// A row comes out of the database in UTC. An appointment at half past midnight
// tomorrow, read against a local clock at half past eleven tonight, is
// "tomorrow" where the person is and "today" in UTC — and reading "dentist
// today" the night before is the one way this line could make somebody leave
// the house.
func TestTheOpeningLineUsesThePersonsDay(t *testing.T) {
	ams, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	tonight := time.Date(2026, 8, 25, 23, 30, 0, 0, ams)
	appointment := time.Date(2026, 8, 26, 0, 30, 0, 0, ams)

	// As the store hands it back before conversion: the right instant, in UTC.
	said := openingLine(ams, squirrel.Waiting{Agenda: 1},
		[]squirrel.Moment{{Label: "dentist", Starts: appointment.UTC()}}, tonight)

	require.Contains(t, said, "dentist tomorrow")
	require.Contains(t, said, "00:30", "it printed the time in the wrong clock")
}

// The offer is the product's whole argument — one thing, chosen for you — and
// tonight's opening line turned it off on every day it spoke, which is most
// days. An opening says what is true and asks nothing; it is not something on
// the table.
func TestTheOpeningDoesNotSwallowTheOffer(t *testing.T) {
	f := &fakeStore{
		checkin:  fresh(),
		waiting:  squirrel.Waiting{Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
		offer:    &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet"},
	}
	body := thread(t, f)

	require.Contains(t, body, "dentist", "it did not open at all")
	require.Contains(t, body, "ring the vet", "the opening line swallowed the offer")
}

// And the offer still refuses to talk over anything that is genuinely on the
// table, which is what endsOpen is for.
func TestTheOfferStillWillNotTalkOverACard(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet"},
		turns: []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "This one.",
			Shown: []byte(`{"cards":[{"title":"the boiler","acts":[{"label":"DONE","action":"/pile/act"}]}]}`)}},
	}

	require.NotContains(t, thread(t, f), "ring the vet")
}

// A question you have not answered is not asked again.
//
// It is still on the screen; asking again does not make it easier to answer,
// it makes a column of the same question — which is what a phone showed, three
// deep, with opening lines in between them.
func TestAnUnansweredCheckinIsNotAskedAgain(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	m.call(t, "GET", "/", nil)
	require.Len(t, f.appended, 1, "it did not ask at all")
	require.Contains(t, f.appended[0].Words, "how do you feel")
	f.turns, f.appended = append(f.turns, f.appended...), nil

	m.call(t, "GET", "/", nil)
	require.Empty(t, f.appended, "it asked how you feel a second time")
}

// Not even with other things said after it. Buddy says more after asking —
// the opening line, what a door drew — so by the time you come back the
// question is rarely the newest thing.
func TestAnUnansweredCheckinIsNotAskedAgainFromFurtherUp(t *testing.T) {
	f := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "how do you feel?", Shown: []byte(`{"faces":true}`)},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "the pile"},
		{ID: 3, Who: squirrel.SpeakerBuddy, Words: "This one.", Shown: []byte(`{"cards":[{"title":"the boiler"}]}`)},
	}}
	thread(t, f)

	require.Empty(t, f.appended, "it asked again from under a conversation")
}

// Answering is what makes it askable again, and a fresh reading is what
// checkinTurn already refuses on — so the two halves compose without a second
// piece of state.
func TestAnsweringMakesItAskableAgain(t *testing.T) {
	f := &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()},
		turns: []squirrel.Turn{
			{ID: 1, Who: squirrel.SpeakerBuddy, Words: "how do you feel?", Shown: []byte(`{"faces":true}`)},
			{ID: 2, Who: squirrel.SpeakerYou, Words: "good"},
		},
	}
	thread(t, f)

	require.Empty(t, f.appended, "a fresh reading was asked about anyway")
}

// The opening line does not land on top of the check-in.
//
// The check-in is a question with its answers drawn on it, exactly like the
// picker — and it was left out of endsAsking when that was written, so the two
// alternated down the screen.
func TestTheOpeningDoesNotTalkOverTheCheckin(t *testing.T) {
	f := &fakeStore{
		waiting:  squirrel.Waiting{Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
	}
	body := thread(t, f)

	require.Contains(t, body, "how do you feel")
	require.NotContains(t, body, "dentist",
		"it handed you something while the question was still open")
}
