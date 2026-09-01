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
