package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheThreadAsksHowRightNowIs(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "how do you feel?")
	// All five, in the one order both surfaces use.
	for _, m := range squirrel.Moods {
		require.Contains(t, body, `value="`+string(m)+`"`, string(m))
		require.Contains(t, body, "/static/mood-"+string(m)+".png")
	}
}

// Every answer stays, and the drawings do not pile up.
//
// This was the load-bearing test for the opposite rule: no surface may ever
// render more than the latest reading, because a fortnight of your own bad
// days on a screen is the counter this product refuses, wearing a face. The
// owner retired that on 24 August 2026 along with Principle 2 — history is
// never rewritten, and a conversation keeps what was said in it.
//
// What survives of the rule, and it is not nothing: only the live edge draws
// the faces. Scrollback holds the words you said, one line each, and never a
// column of five drawings repeating down the page. /moods is still the only
// place that groups them.
func TestEveryAnswerStaysAndOnlyTheLiveEdgeDrawsFaces(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)

	for _, mood := range []string{"wiped", "low", "good"} {
		require.Equal(t, 303, post(t, m, "/mood", url.Values{"mood": {mood}}).Code)
	}
	f.turns, f.appended = f.appended, nil

	body := m.call(t, "GET", "/r/everything", nil).Body.String()
	for _, said := range []string{"wiped", "low", "good"} {
		require.Contains(t, body, said, "what you said stays")
	}
	require.NotContains(t, body, "mood-wiped.png", "scrollback carries no faces")
	require.NotContains(t, body, "mood-low.png")
}

// It says the word and stops. No sympathy from a program, and nothing about
// the person.
func TestTheCheckinSaysNothingAboutYou(t *testing.T) {
	f := &fakeStore{checkin: &squirrel.Checkin{Mood: squirrel.MoodLow, SaidAt: time.Now()}}
	post(t, mounted(t, f), "/mood", url.Values{"mood": {"low"}})
	f.turns, f.appended = f.appended, nil
	body := strings.ToLower(mounted(t, f).call(t, "GET", "/r/everything", nil).Body.String())

	require.Contains(t, body, "noted")
	for _, said := range []string{
		"sorry", "hope", "you have been", "again today", "yesterday",
		"three days", "streak", "lately", "still",
	} {
		require.NotContains(t, body, said)
	}
}

// Six hours, because the question is "how do you feel?" and this morning is
// not now. A stale reading is not a bad one; it is not an answer to the
// question being asked.
func TestAStaleReadingIsAskedAgain(t *testing.T) {
	f := &fakeStore{checkin: &squirrel.Checkin{
		Mood: squirrel.MoodGood, SaidAt: time.Now().Add(-7 * time.Hour),
	}}
	body := mounted(t, f).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "how do you feel?")
	require.NotContains(t, body, "noted")
}

// Changing your mind is not a special case: it is the same answer given twice.
func TestSayingSomethingElseAsksAgain(t *testing.T) {
	f := &fakeStore{checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}}
	m := mounted(t, f)

	// A fresh reading, so nothing is asked...
	require.NotContains(t, m.call(t, "GET", "/r/everything", nil).Body.String(), "how do you feel?")
	// ...until you say you want to say something else. The way to do that
	// travels with Buddy's acknowledgement, because the answer is about to be
	// scrollback and scrollback carries no controls.
	require.Contains(t, m.call(t, "GET", "/r/everything?ask=1", nil).Body.String(), "how do you feel?")
}

// Not one of the five is no answer rather than a wrong one — this arrives from
// a form, so it is read the way a stranger's typing is read.
func TestAMoodThatIsNotOneOfTheFiveIsIgnored(t *testing.T) {
	f := &fakeStore{}
	w := post(t, mounted(t, f), "/mood", url.Values{"mood": {"terrific"}})

	require.Equal(t, 303, w.Code)
	require.Nil(t, f.checkin)
}

// A record it cannot read is not a reason to take the screen away. The doors
// still work and the dock still writes to the spool, which is the whole of what
// an unreachable database must not stop — and Buddy says so rather than
// rendering an empty conversation, because an empty conversation looks like
// your history is gone.
func TestTheThreadStillStandsWhenNothingCanBeRead(t *testing.T) {
	w := mounted(t, &fakeStore{err: errTest}).call(t, "GET", "/r/everything", nil)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "I cannot reach what we said")
	require.Contains(t, w.Body.String(), "the notes")
	require.Contains(t, w.Body.String(), `action="/capture"`)
}
