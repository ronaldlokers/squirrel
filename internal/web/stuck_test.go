package web

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// The offer carries a way to say you cannot start it, quietly.
func TestTheOfferOffersAWayToSayYouCannotStart(t *testing.T) {
	body := mounted(t, answered(aTask)).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "i can't start")
	require.Contains(t, body, `value="big"`)
	require.Contains(t, body, `value="not today"`)
}

// Asked once, answered once. The answer is a line and one control, and the
// thing you could not start is still on the card — taking it away would make
// "I can't start" a way of losing it.
func TestTooBigAnswersWithSomethingSmallerAndKeepsTheThing(t *testing.T) {
	m := mounted(t, answered(aTask))
	w := post(t, m, "/now/stuck", url.Values{
		"kind": {"task"}, "id": {"7"}, "why": {"big"},
	})
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?stuck=big", w.Header().Get("Location"))

	body := m.call(t, "GET", "/?stuck=big", nil).Body.String()
	require.Contains(t, body, "smallest piece")
	require.Contains(t, body, "5 MIN")
	require.Contains(t, body, "ring the vet about the booster")
}

// The one branch that captures: what you would have to find out is a thought.
func TestNotKnowingHowOffersTheSlot(t *testing.T) {
	body := mounted(t, answered(aTask)).call(t, "GET", "/?stuck=how", nil).Body.String()

	require.Contains(t, body, "first thing you would have to find out")
	require.Contains(t, body, `action="/capture"`)
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

	body := m.call(t, "GET", "/?stuck=purple", nil).Body.String()
	require.NotContains(t, body, `class="unstuck"`)
	require.Contains(t, body, "i can't start", "the way to ask is still there")
}
