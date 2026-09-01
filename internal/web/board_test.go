package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func aBoardStore() *fakeStore {
	return &fakeStore{
		items: []squirrel.Item{
			{ID: 1, RawText: "boiler service code is 4471", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
			{ID: 2, RawText: "kaas", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
			{ID: 3, RawText: "vet about the booster", State: squirrel.ItemOpen, Kind: squirrel.ItemTask, ReceivedAt: time.Now()},
		},
		chores: []squirrel.Chore{
			{ID: 7, Name: "bins out", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 7},
		},
	}
}

func TestTheBoardDrawsEveryBayFromTheStore(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, want := range []string{
		"the notes", "the chores", "the tasks", "the agenda",
		"boiler service code is 4471", "kaas", "bins out", "vet about the booster",
	} {
		require.Contains(t, body, want)
	}
}

func TestTheBoardIsNotAConversation(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, gone := range []string{"frombuddy", "fromyou", "whensaid", "class=\"dock\"", "bubble"} {
		require.NotContains(t, body, gone)
	}
}

func TestTheShelvesAreReachedFromTheNotesRackAndCountNothing(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, "what you set aside")
	require.Contains(t, body, "the things you kept")
	require.NotContains(t, body, "ledge\"><span class=\"tab\">what you set aside <span")
}

func TestAStripCarriesTheAnswersItsBayAllows(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, want := range []string{">done<", ">keep<", ">drop<", ">did it<", ">later<"} {
		require.Contains(t, body, want)
	}
}

func TestAnsweringAStripMovesItAndComesBackToTheBoard(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/act", strings.NewReader("what=note&id=1&answer=keep"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/board", w.Header().Get("Location"))
	require.Equal(t, squirrel.ItemKept, f.states[1], "the note is kept")
}

func TestTheTrayHoldsWhatLeftTheBoardTodayAndOffersTheWayBack(t *testing.T) {
	f := aBoardStore()
	f.triaged = []squirrel.Item{
		{ID: 9, RawText: "washing machine one", State: squirrel.ItemDone, Kind: squirrel.ItemNote},
		{ID: 8, RawText: "the thing about the bike lights", State: squirrel.ItemDropped, Kind: squirrel.ItemNote},
	}
	m := mounted(t, f)

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, "washing machine one")
	require.Contains(t, body, "the thing about the bike lights")
	require.Contains(t, body, "put it back")
	require.NotContains(t, body, "tray\"><span class=\"sign\">today's tray</span> <span class=\"n\">")
}

// A row that is not yours is not yours to answer. SetItemState takes an item id
// and no person, so the guard has to be the read before it: the board looks the
// strip up as yours, and writes nothing when it is not.
func TestAStripThatIsNotYoursIsNotYoursToAnswer(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items, squirrel.Item{
		ID: 99, RawText: "somebody else's note", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
	})
	f.notMine = map[int64]bool{99: true}
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/act", strings.NewReader("what=note&id=99&answer=drop"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/board", w.Header().Get("Location"))
	require.NotContains(t, f.states, int64(99), "a strip that is not yours was answered")
}

func TestPuttingBackWhatIsNotYoursWritesNothing(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items, squirrel.Item{
		ID: 99, RawText: "somebody else's note", State: squirrel.ItemDropped, Kind: squirrel.ItemNote,
	})
	f.notMine = map[int64]bool{99: true}
	m := mounted(t, f)

	m.call(t, "POST", "/board/undo", strings.NewReader("id=99"))

	require.NotContains(t, f.states, int64(99), "a strip that is not yours was put back")
}

// A thought typed into the notes rack goes to the spool, not to the database.
// Capture is sacred: the words reach fsynced disk before anybody answers, and
// the drain resolves whose they are — the same path a capture from the
// conversation takes.
func TestWritingOnABlankStripSpoolsTheWords(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	w := m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=meter+reading+48213"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/board", w.Header().Get("Location"))
	require.Len(t, sp.written, 1, "the words did not reach the spool")
	require.Equal(t, "meter reading 48213", sp.written[0].Text)
	require.Len(t, f.items, 3, "a note typed on the board went straight to the database")
}

func TestWritingInTheTasksRackDecidesSomething(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=tasks&words=book+the+boiler+service"))

	require.Empty(t, sp.written, "a task is a decision rather than a capture")
	var found bool
	for _, it := range f.items {
		if it.RawText == "book the boiler service" && it.Kind == squirrel.ItemTask {
			found = true
		}
	}
	require.True(t, found, "the words are not a task in the store")
}

func TestABlankStripWithNothingOnItKeepsNothing(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=+"))

	require.Empty(t, sp.written)
	require.Len(t, f.items, 3, "an empty strip kept something")
}

func TestTheBlankStripIsAFieldYouCanType(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, `action="/board/new"`)
	require.Contains(t, body, `name="words"`)
	require.Contains(t, body, `value="notes"`)
}
