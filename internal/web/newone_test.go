package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A new one, at every door. Making a thing from nothing went with the screens,
// and a chore or an appointment is not a thought you had — you decide to have
// it, so the dock is not where it comes from.

// Every door offers it, and offers it whether or not there is anything there.
// An empty list is the moment you are most likely to want to add to it.
func TestEveryDoorOffersANewOne(t *testing.T) {
	for _, d := range []struct{ where, chip string }{
		{"chores", "a new chore"},
		{"tasks", "a new task"},
		{"at", "a new appointment"},
		{"pile", "put something down"},
	} {
		t.Run(d.where, func(t *testing.T) {
			full := &fakeStore{
				items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen), task(2, "ring the vet", squirrel.ItemOpen)},
				chores:  []squirrel.Chore{{ID: 1, Name: "bins out", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7}},
				moments: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(time.Hour)}},
			}
			require.Contains(t, drewFor(t, full, d.where), d.chip,
				"%s does not offer a new one", d.where)
			require.Contains(t, drewFor(t, &fakeStore{}, d.where), d.chip,
				"%s offers nothing when it is empty, which is when you want it", d.where)
		})
	}
}

// Never over a question. A turn waiting for an answer is not the moment to
// offer starting something else.
func TestANewOneIsNotOfferedOverAQuestion(t *testing.T) {
	asking := squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "How often?",
		Shown: []byte(`{"pick":{"action":"/chores/act","do":"that's it","rows":[]}}`)}

	require.Equal(t, asking.Shown,
		alsoOffer(asking, turnChip{Label: "a new chore", Action: "/chores/ask"}).Shown)
}

// A chore is two answers: what it is, and how often.
func TestMakingAChoreAsksBothHalves(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	f.appended = nil
	m.call(t, "POST", "/chores/ask", nil)
	require.Contains(t, f.appended[1].Words, "What should come back?")
	require.Contains(t, string(f.appended[1].Shown), `"action":"/chores/name"`)

	f.appended = nil
	m.call(t, "POST", "/chores/name", strings.NewReader("name=water+the+plants"))
	require.Contains(t, string(f.appended[1].Shown), `"action":"/chores/new"`)
	require.Contains(t, string(f.appended[1].Shown), "water the plants")
	require.Empty(t, f.chores, "it made the chore before asking how often")

	f.appended = nil
	m.call(t, "POST", "/chores/new", strings.NewReader("name=water+the+plants&count=2&unit=weeks"))
	require.Len(t, f.chores, 1)
	require.Equal(t, "water the plants", f.chores[0].Name)
	require.Equal(t, 14*24*time.Hour, f.chores[0].Every)
}

// Nothing is written until the second answer, so walking away halfway leaves
// no half-made chore behind.
func TestAbandoningTheChoreQuestionMakesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/name", strings.NewReader("name=water+the+plants"))

	require.Empty(t, f.chores)
}

// And an empty answer makes nothing and says nothing. An empty box submitted
// by accident is not a mistake worth a sentence.
func TestAnEmptyNameMakesNothing(t *testing.T) {
	f := &fakeStore{}
	f.appended = nil
	routed(t, f).call(t, "POST", "/chores/name", strings.NewReader("name=+++"))

	require.Empty(t, f.chores)
	require.Empty(t, f.appended)
}

func TestMakingATaskTakesOneAnswer(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	f.appended = nil
	m.call(t, "POST", "/tasks/ask", nil)
	require.Contains(t, string(f.appended[1].Shown), `"action":"/tasks/new"`)

	f.appended = nil
	m.call(t, "POST", "/tasks/new", strings.NewReader("text=book+the+car+in"))

	require.Len(t, f.items, 1)
	require.Equal(t, squirrel.ItemTask, f.items[0].Kind)
	require.Equal(t, "book the car in", f.appended[0].Words)
	require.Contains(t, f.appended[1].Words, "On the list")
}

// The pile's chip goes through capture, which is the offline durability
// promise. A second way in with weaker guarantees would be a second way in
// nobody could tell apart.
func TestThePilesChipGoesThroughCapture(t *testing.T) {
	f := &fakeStore{}
	f.appended = nil
	routed(t, f).call(t, "POST", "/pile/ask", nil)

	require.Contains(t, string(f.appended[1].Shown), `"action":"/capture"`)
}

// The appointment chip goes straight to the day picker: an appointment is a
// day and a time before it is anything else.
func TestTheAppointmentChipAsksWhichDay(t *testing.T) {
	require.Contains(t, drewFor(t, &fakeStore{}, "at"), `"action":"/at/new"`)
}
