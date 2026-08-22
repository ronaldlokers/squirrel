package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func aConversation(t *testing.T) (*fakeStore, *fakeCoach) {
	t.Helper()
	f := &fakeStore{checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}}
	c := &fakeCoach{talk: []Exchange{
		{Said: "everything at once", Replied: "start with the envelope."},
		{Said: "still stuck", Replied: "you have done this three times this week"},
	}}
	return f, c
}

// The sheet offers a way to say the last thing landed badly.
//
// Principle 5 was opened on 20 August so the coach could be useful at the one
// thing a coach is for, and the cost was written down at the time: it can now
// say something that lands badly on a bad day. Every exchange has been kept
// since, for exactly that reason — and until now nothing read one back, so
// there was no way for a bad night to matter afterwards.
func TestTheSheetOffersAWayToSayItLandedBadly(t *testing.T) {
	f, c := aConversation(t)

	body := mountedWith(t, f, c).call(t, "GET", "/buddy", nil).Body.String()

	require.Contains(t, body, "that landed badly")
	require.Contains(t, body, `action="/buddy/badly"`)

	// On the last reply and nowhere else: an exchange is two strings and
	// carries no row id, so this can only mean the one you just read.
	require.Equal(t, 1, strings.Count(body, `action="/buddy/badly"`),
		"every reply in the conversation offered it, and only the last one can mean it")
	require.Less(t, strings.Index(body, "you have done this three times"),
		strings.Index(body, "that landed badly"),
		"the control sits above the reply it is about")
}

// Pressing it records it, and says only that it was heard.
func TestSayingItLandedBadlyIsHeardAndNothingElse(t *testing.T) {
	f, c := aConversation(t)
	m := mountedWith(t, f, c)

	to := m.call(t, "POST", "/buddy/badly", strings.NewReader("")).Header().Get("Location")
	require.Equal(t, "/buddy?heard=1", to)
	require.Len(t, f.landedBadly, 1, "the press was not recorded")

	body := m.call(t, "GET", to, nil).Body.String()
	require.Contains(t, body, "noted")

	// No count, no list, no history. How often a thing lands badly is a fact
	// about the person, and rule 2 forbids one on any surface.
	for _, counting := range []string{"1 time", "2 times", "twice", "so far", "total"} {
		require.NotContains(t, strings.ToLower(body), counting)
	}
}

// A press with nothing behind it says nothing rather than claiming it heard
// something.
func TestAPressWithNothingToMarkClaimsNothing(t *testing.T) {
	f, c := aConversation(t)
	f.noReplyToMark = true

	to := mountedWith(t, f, c).call(t, "POST", "/buddy/badly", strings.NewReader("")).
		Header().Get("Location")

	require.Equal(t, "/buddy", to, "it said it had heard something that was not there")
}

// And a store that cannot record it does not answer with an error page.
//
// Saying "that landed badly" and being handed a failure is the worst possible
// answer to it, and nothing on the screen depends on the write.
func TestAFailureToRecordIsNotAnErrorPage(t *testing.T) {
	f, c := aConversation(t)
	f.err = errTest

	res := mountedWith(t, f, c).call(t, "POST", "/buddy/badly", strings.NewReader(""))

	require.Equal(t, 303, res.Code, "it failed loudly at the worst moment to fail loudly")
	require.Equal(t, "/buddy", res.Header().Get("Location"))
}
