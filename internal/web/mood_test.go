package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestHomeAsksHowRightNowIs(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "how do you feel?")
	// All five, in the one order both surfaces use.
	for _, m := range squirrel.Moods {
		require.Contains(t, body, `value="`+string(m)+`"`, string(m))
		require.Contains(t, body, "/static/mood-"+string(m)+".png")
	}
}

// The load-bearing test for this feature. The history exists — it is what lets
// a nudge be gentler — and no surface may ever render more than the latest
// reading. A fortnight of your own bad days on a screen is the counter this
// product refuses, wearing a face.
func TestHomeShowsOneReadingAndNeverASeries(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)

	for _, mood := range []string{"wiped", "low", "good"} {
		require.Equal(t, 303, post(t, m, "/mood", url.Values{"mood": {mood}}).Code)
	}

	body := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, "noted")
	require.Contains(t, body, "mood-good.png", "the latest one")
	require.NotContains(t, body, "mood-wiped.png")
	require.NotContains(t, body, "mood-low.png")
}

// It says the word and stops. No sympathy from a program, and nothing about
// the person.
func TestTheCheckinSaysNothingAboutYou(t *testing.T) {
	f := &fakeStore{checkin: &squirrel.Checkin{Mood: squirrel.MoodLow, SaidAt: time.Now()}}
	body := strings.ToLower(mounted(t, f).call(t, "GET", "/", nil).Body.String())

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
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "how do you feel?")
	require.NotContains(t, body, "noted")
}

// Changing your mind is not a special case: it is the same answer given twice.
func TestSayingSomethingElseAsksAgain(t *testing.T) {
	f := &fakeStore{checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}}
	m := mounted(t, f)

	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), "noted")
	require.Contains(t, m.call(t, "GET", "/?ask=1", nil).Body.String(), "how do you feel?")
}

// Not one of the five is no answer rather than a wrong one — this arrives from
// a form, so it is read the way a stranger's typing is read.
func TestAMoodThatIsNotOneOfTheFiveIsIgnored(t *testing.T) {
	f := &fakeStore{}
	w := post(t, mounted(t, f), "/mood", url.Values{"mood": {"terrific"}})

	require.Equal(t, 303, w.Code)
	require.Nil(t, f.checkin)
}

// Home reads the pile for nothing, and a check-in it cannot read is not a
// reason to fail a page that otherwise needs no database at all.
func TestHomeStillStandsWhenTheCheckinCannotBeRead(t *testing.T) {
	w := mounted(t, &fakeStore{err: errTest}).call(t, "GET", "/", nil)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "how do you feel?")
	require.Contains(t, w.Body.String(), "the pile")
}
