package web

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// The offer carries the four ways of not being able to start it.
//
// On the card rather than behind a disclosure, because the card is about to be
// scrollback and scrollback carries no controls: a ladder you can only reach by
// pressing something that has already gone is a ladder nobody reaches.
func TestTheOfferOffersAWayToSayYouCannotStart(t *testing.T) {
	body := mounted(t, answered(aTask)).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `action="/now/stuck"`)
	require.Contains(t, body, `value="big"`)
	require.Contains(t, body, `value="not today"`)
}

// Asked once, answered once. The answer is a line and one control, and the
// thing you could not start is still on the card — taking it away would make
// "I can't start" a way of losing it.
func TestTooBigAnswersWithSomethingSmallerAndKeepsTheThing(t *testing.T) {
	f := answered(aTask)
	m := mounted(t, f)
	w := post(t, m, "/now/stuck", url.Values{
		"kind": {"task"}, "id": {"7"}, "why": {"big"},
		"label": {"ring the vet about the booster"},
	})
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))

	// Two turns: what you said, and the way through. The thing you could not
	// start is still above it in the conversation — taking it away would make
	// "I can't start" a way of losing it.
	require.Len(t, f.appended, 2)
	require.Equal(t, "too big", f.appended[0].Words)

	f.turns, f.appended = f.appended, nil
	body := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, "smallest piece")
	require.Contains(t, body, "5 MIN")
}

// The one branch that captures: what you would have to find out is a thought,
// and thoughts go in the pile. There is nothing to offer any more — the dock is
// always there — so Buddy asks and the answer goes where everything else does.
func TestNotKnowingHowAsksAndTheDockTakesTheAnswer(t *testing.T) {
	f := answered(aTask)
	m := mounted(t, f)
	post(t, m, "/now/stuck", url.Values{"kind": {"task"}, "id": {"7"}, "why": {"how"}})

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "find out first")

	f.turns, f.appended = f.appended, nil
	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), `action="/capture"`)
}

// Not today is the same no that "not now" writes, arrived at from a different
// direction — so it writes the same thing and says nothing further.
func TestNotTodayRefusesTheOffer(t *testing.T) {
	s := answered(aTask)
	w := post(t, mounted(t, s), "/now/stuck", url.Values{
		"kind": {"task"}, "id": {"7"}, "why": {"not today"},
	})

	require.Equal(t, "/", w.Header().Get("Location"))
	require.Equal(t, []int64{7}, s.refused)
}

// It arrives through the address bar, so it is read the way a stranger's
// typing is read.
func TestAnAnswerThatWasNeverOfferedDoesNothing(t *testing.T) {
	s := answered(aTask)
	m := mounted(t, s)

	w := post(t, m, "/now/stuck", url.Values{"kind": {"task"}, "id": {"7"}, "why": {"purple"}})
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Empty(t, s.refused)
	require.Empty(t, s.appended, "nothing was said about a word nobody offered")

	body := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, `action="/now/stuck"`, "the way to ask is still there")
}
