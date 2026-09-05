package web

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func aRackWithoutAgenda() *fakeStore {
	return &fakeStore{
		items: []squirrel.Item{
			note(1, "boiler service code is 4471", squirrel.ItemOpen),
			task(3, "vet about the booster", squirrel.ItemOpen),
		},
		chores: []squirrel.Chore{
			{ID: 7, Name: "bins out", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7},
		},
	}
}

func TestWithNoCoachTheBoardOffersNoWayToAsk(t *testing.T) {
	body := mounted(t, aRackWithoutAgenda()).call(t, "GET", "/board", nil).Body.String()

	require.NotContains(t, body, "ask about this",
		"a board with no coach must draw exactly as it did before this feature")
	require.NotContains(t, body, `action="/board/ask"`)
}

func TestWithACoachEveryLiveStripCanBeAsked(t *testing.T) {
	body := mountedWith(t, aRackWithoutAgenda(), &fakeCoach{}).call(t, "GET", "/board", nil).Body.String()

	require.Equal(t, 3, strings.Count(body, "ask about this"),
		"one press per live strip: two notes worth (a note and a task) and a chore")
}

func TestDrawingTheBoardCallsNoModel(t *testing.T) {
	c := &fakeCoach{reply: "should never be seen"}
	mountedWith(t, aRackWithoutAgenda(), c).call(t, "GET", "/board", nil)

	require.Empty(t, c.asked, "opening the board must never cost a model call")
}

func TestPressingAskQuestionsTheModelAboutThatStripAndNarrowsTheRoom(t *testing.T) {
	c := &fakeCoach{reply: "This is the third note about that boiler."}
	m := mountedWith(t, aRackWithoutAgenda(), c)

	w := post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?bay=notes", w.Header().Get("Location"))
	require.Len(t, c.asked, 1, "the press should have asked exactly once")
	got := c.asked[0]
	require.Equal(t, "strip", got.kind)
	require.Equal(t, "notes", got.room, "asking about a note must not open the tasks or chores toolset")
	require.Equal(t, "What is going on with this?", got.said)
	require.Equal(t, "boiler service code is 4471", got.subject, "the model was not told what strip it was about")
}

func roomFieldNear(t *testing.T, body, marker string) string {
	t.Helper()
	at := strings.Index(body, marker)
	require.GreaterOrEqual(t, at, 0, "marker %q not found", marker)
	end := strings.Index(body[at:], "</article>")
	require.GreaterOrEqual(t, end, 0)
	block := body[at : at+end]
	at2 := strings.Index(block, `name="room" value="`)
	require.GreaterOrEqual(t, at2, 0, "no room field near %q", marker)
	rest := block[at2+len(`name="room" value="`):]
	return rest[:strings.Index(rest, `"`)]
}

func TestEachBaysDrawnAskButtonNarrowsToItsOwnRoom(t *testing.T) {
	body := mountedWith(t, aRackWithoutAgenda(), &fakeCoach{}).call(t, "GET", "/board", nil).Body.String()

	require.Equal(t, "notes", roomFieldNear(t, body, "boiler service code is 4471"))
	require.Equal(t, "tasks", roomFieldNear(t, body, "vet about the booster"))
	require.Equal(t, "chores", roomFieldNear(t, body, "bins out"))
}

func TestPressingAskOnATaskNarrowsToTheTasksRoom(t *testing.T) {
	c := &fakeCoach{reply: "one thing at a time"}
	m := mountedWith(t, aRackWithoutAgenda(), c)

	post(t, m, "/board/ask", url.Values{
		"id": {"3"}, "what": {"task"}, "bay": {"tasks"}, "room": {"tasks"},
		"words": {"vet about the booster"},
	})

	require.Len(t, c.asked, 1)
	require.Equal(t, "tasks", c.asked[0].room)
}

func TestTheAnswerRendersAsALineUnderTheStripItWasAskedOf(t *testing.T) {
	c := &fakeCoach{reply: "This is the third note about that boiler."}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, `<div class="seen">This is the third note about that boiler.`,
		"the answer must render in the same register as marginalia")
	stripAt := strings.Index(body, "boiler service code is 4471")
	seenAt := strings.Index(body, "This is the third note about that boiler.")
	require.Greater(t, seenAt, stripAt, "the line must sit under the strip it answers, not above it")
}

func TestTheAnswerDoesNotOfferAControlThatActsOnTheNote(t *testing.T) {
	c := &fakeCoach{
		reply:   "keep it for a rainy day",
		propose: &Proposal{Do: "drop", Said: "drop it", Text: "drop it", RefID: 1},
		opens:   "notes",
	}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.NotContains(t, body, "KEEP IT",
		"a board ask must not draw a proposal card that names the note back into a write")
	require.NotContains(t, body, `name="do" value="drop"`)
}

func TestAFailedAskLeavesTheBoardExactlyAsItWas(t *testing.T) {
	c := &fakeCoach{err: errTest}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	before := m.call(t, "GET", "/board", nil).Body.String()
	w := post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})
	after := m.call(t, "GET", "/board", nil).Body.String()

	require.Equal(t, 303, w.Code)
	require.Equal(t, before, after, "an unreachable model must not change what the board draws")
	require.Empty(t, f.noticed, "an unreachable model must write nothing about the strip")
}

func TestAskingWithNoCoachDoesNothing(t *testing.T) {
	f := aRackWithoutAgenda()
	w := post(t, mounted(t, f), "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})

	require.Equal(t, 303, w.Code)
	require.Empty(t, f.noticed, "with no coach there is nothing to write")
}

func TestTheStoredAnswerIsRefusedTheSameWayAMarginaliaLineIs(t *testing.T) {
	c := &fakeCoach{reply: "keep the boiler code somewhere safer"}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})
	require.Len(t, f.noticed, 1)
	id := f.noticed[0].ID

	w := post(t, m, "/board/notuseful", url.Values{"id": {strconv.FormatInt(id, 10)}, "bay": {"notes"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, []int64{id}, f.unuseful)
}

func TestAskingAgainAboutTheSameStripReplacesTheAnswerRatherThanStackingIt(t *testing.T) {
	c := &fakeCoach{reply: "first answer"}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})
	c.reply = "second answer"
	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})

	require.Len(t, f.noticed, 1, "a strip carries one line, not a conversation")
	require.Equal(t, "second answer", f.noticed[0].Words)
}
