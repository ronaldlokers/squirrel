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

// A way to say the last thing landed badly, on the reply it is about.
//
// Principle 5 was opened on 20 August so the coach could be useful at the one
// thing a coach is for, and the cost was written down at the time: it can now
// say something that lands badly on a bad day. Every exchange has been kept
// since, for exactly that reason — and until now nothing read one back, so
// there was no way for a bad night to matter afterwards.
//
// On the reply, and only the newest reply carries controls: by the live edge
// rule the offer is gone from anything in scrollback, which is what stops it
// meaning a reply from three days ago.
func TestTheReplyOffersAWayToSayItLandedBadly(t *testing.T) {
	f, c := aConversation(t)
	m := mountedWith(t, f, c)
	drew := asked(t, m, f, "said=still+stuck")

	require.Contains(t, drew, "that went badly")
	require.Contains(t, drew, "/buddy/badly")
	require.Equal(t, 1, strings.Count(drew, "/buddy/badly"),
		"every reply offered it, and only the last one can mean it")
}

// Pressing it records it, and says only that it was heard.
func TestSayingItLandedBadlyIsHeardAndNothingElse(t *testing.T) {
	f, c := aConversation(t)
	m := mountedWith(t, f, c)

	f.appended = nil
	w := m.call(t, "POST", "/buddy/badly", strings.NewReader(""))
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Len(t, f.landedBadly, 1, "the press was not recorded")

	body := f.appended[1].Words
	require.Contains(t, body, "I will be shown that one.",
		"it did not say the words would be shown to Buddy")

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

	f.appended = nil
	mountedWith(t, f, c).call(t, "POST", "/buddy/badly", strings.NewReader(""))

	// The wording varies by the day, so this asks for the shape rather than
	// one phrasing: taken, and no claim that anything was marked.
	require.Contains(t, squirrel.Sayings(squirrel.SayingHeard), f.appended[1].Words)
	require.NotContains(t, f.appended[1].Words, "I will be shown",
		"it said it had heard something that was not there")
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
	require.Equal(t, "/", res.Header().Get("Location"))
}
