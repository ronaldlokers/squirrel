package web

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"

	"github.com/ronaldlokers/squirrel/internal/coach"
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

	require.NotContains(t, body, "ask Buddy",
		"a board with no coach must draw exactly as it did before this feature")
	require.NotContains(t, body, `action="/board/ask"`)
}

func TestWithACoachEveryLiveStripCanBeAsked(t *testing.T) {
	m := mountedWith(t, aRackWithoutAgenda(), &fakeCoach{})

	require.Equal(t, 2, strings.Count(m.call(t, "GET", "/board", nil).Body.String(), "ask Buddy"),
		"the board is one chore and one thing you do once, and both can be asked about")
	require.Equal(t, 1, strings.Count(m.call(t, "GET", "/?bay=notes", nil).Body.String(), "ask Buddy"),
		"the one strip behind the notes cannot be asked about")
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
	require.Equal(t, "/?bay=notes&answered=1", w.Header().Get("Location"))
	require.Len(t, c.asked, 1, "the press should have asked exactly once")
	got := c.asked[0]
	require.Equal(t, "strip", got.kind)
	require.Equal(t, "notes", got.room, "asking about a note must not open the tasks or chores toolset")
	require.Equal(t, "boiler service code is 4471", got.said,
		"the model was not told what strip it was about")
	require.Empty(t, got.subject, "the strip is what was said; naming it again on screen says it twice")
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
	m := mountedWith(t, aRackWithoutAgenda(), &fakeCoach{})

	require.Equal(t, "notes", roomFieldNear(t, m.call(t, "GET", "/?bay=notes", nil).Body.String(),
		"boiler service code is 4471"))
	require.Equal(t, "tasks", roomFieldNear(t, m.call(t, "GET", "/", nil).Body.String(),
		"vet about the booster"))
	// The room is the model's toolset and not a place on the screen, so a
	// chore in the weekly rack still opens the chores one.
	require.Equal(t, "chores", roomFieldNear(t, m.call(t, "GET", "/", nil).Body.String(), "bins out"))
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

	body := m.call(t, "GET", "/?bay=notes", nil).Body.String()

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

func TestPressingAskSaysTheStripSoAPileCanBeSeenAsOne(t *testing.T) {
	pile := "the tax thing, the vet, the bins and ring the school"
	c := &fakeCoach{reply: "start with the school"}
	m := mountedWith(t, aRackWithoutAgenda(), c)

	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {pile},
	})

	require.Len(t, c.asked, 1)
	require.Equal(t, pile, c.asked[0].said,
		"the words the escalation is decided on never reached the model as what was said")
	require.True(t, coach.Overwhelmed(c.asked[0].said),
		"a pile handed to the coach as this turn's words is not recognised as one")
}

func TestPressingAskWithNoWordsCallsNoModel(t *testing.T) {
	c := &fakeCoach{reply: "should never be seen"}
	m := mountedWith(t, aRackWithoutAgenda(), c)

	w := post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"}, "words": {"   "},
	})

	require.Equal(t, 303, w.Code)
	require.Empty(t, c.asked)
}

func TestTheAskPressIsNotDrawnAsADisposition(t *testing.T) {
	body := mountedWith(t, aRackWithoutAgenda(), &fakeCoach{}).call(t, "GET", "/?bay=notes", nil).Body.String()

	require.NotContains(t, body, `formaction="/board/ask"`,
		"the press rides the dispositions' own form, so it is struck like one")
	require.Contains(t, body, `<form class="asking" method="post" action="/board/ask">`)
	require.Contains(t, body, `<button class="quiet">ask Buddy</button>`)
	require.NotContains(t, body, `class="stamp" type="submit" formmethod="post" formaction="/board/ask"`)

	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "ask Buddy") {
			require.NotContains(t, line, "stamp",
				"the ask press is drawn in the dispositions' register")
			require.NotContains(t, line, `class="k"`,
				"the ask press advertises a key letter")
		}
	}
}

func TestRefusingANoticeIsQuietAndLowercase(t *testing.T) {
	css := mounted(t, aBoardStore()).call(t, "GET", "/static/board.css", nil).Body.String()
	at := strings.Index(css, ".strip .seen .off {")
	require.GreaterOrEqual(t, at, 0)
	block := css[at : at+strings.Index(css[at:], "}")]

	require.NotContains(t, block, "text-transform: uppercase",
		"not useful is drawn in caps, and the law says quiet and lowercase")
}

func TestTheAnswerIsSomethingAScreenReaderIsSentTo(t *testing.T) {
	c := &fakeCoach{reply: "This is the third note about that boiler."}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	w := post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})
	require.Equal(t, "/?bay=notes&answered=1", w.Header().Get("Location"))

	body := m.call(t, "GET", w.Header().Get("Location"), nil).Body.String()
	require.Contains(t, body, `<div class="seen" id="justanswered" tabindex="-1">This is the third note about that boiler.`,
		"nothing on the page can be moved to, so the answer arrives in silence")

	quiet := m.call(t, "GET", "/?bay=notes", nil).Body.String()
	require.NotContains(t, quiet, "justanswered",
		"every later draw of the board sends you back to the same answer")
}

func TestTheAnswerOfAStripYouDidNotAskAboutIsNotTheOneSentTo(t *testing.T) {
	c := &fakeCoach{reply: "one thing at a time"}
	f := aRackWithoutAgenda()
	m := mountedWith(t, f, c)

	post(t, m, "/board/ask", url.Values{
		"id": {"1"}, "what": {"note"}, "bay": {"notes"}, "room": {"notes"},
		"words": {"boiler service code is 4471"},
	})
	post(t, m, "/board/ask", url.Values{
		"id": {"3"}, "what": {"task"}, "bay": {"tasks"}, "room": {"tasks"},
		"words": {"vet about the booster"},
	})

	body := m.call(t, "GET", "/?bay=notes&answered=1", nil).Body.String()
	require.Equal(t, 1, strings.Count(body, "justanswered"))
}
