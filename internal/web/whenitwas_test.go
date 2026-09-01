package web

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func at(t *testing.T, when string) time.Time {
	t.Helper()
	said, err := time.ParseInLocation("2006-01-02 15:04", when, time.UTC)
	require.NoError(t, err)
	return said
}

// A run is one utterance, so one time is the whole truth about it. A time on
// every line is a log of your day, which is the shape the run-resumption
// sentence refuses a clock time to avoid.
func TestTheTimeIsOncePerRunAndNotOnEveryLine(t *testing.T) {
	turns := []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler code", SaidAt: at(t, "2026-08-31 14:12")},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "and the wifi one", SaidAt: at(t, "2026-08-31 14:13")},
		{ID: 3, Who: squirrel.SpeakerBuddy, Words: "Down they go.", SaidAt: at(t, "2026-08-31 14:14")},
	}
	vs := turnViews(context.Background(), turns)

	require.Equal(t, "14:12", vs[0].When, "the run does not say when it started")
	require.Empty(t, vs[1].When, "a time on every line makes a log")
	require.Equal(t, "14:14", vs[2].When, "the other speaker's run says nothing")
}

// The time is read in your zone, never the container's — issue #148.
func TestTheTimeIsReadWhereYouAre(t *testing.T) {
	amsterdam, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	turns := []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler code", SaidAt: at(t, "2026-08-31 14:12")},
	}

	utc := turnViews(context.Background(), turns)
	here := turnViews(context.WithValue(context.Background(), zoneKey{}, amsterdam), turns)

	require.Equal(t, "14:12", utc[0].When)
	require.Equal(t, "16:12", here[0].When, "the time is drawn in the container's zone")
}

func TestTheDayIsSaidOnceWhereItTurnsOver(t *testing.T) {
	now = func() time.Time { return at(t, "2026-08-31 20:00") }
	t.Cleanup(func() { now = time.Now })

	turns := []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "monday", SaidAt: at(t, "2026-08-24 09:00")},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "still monday", SaidAt: at(t, "2026-08-24 17:00")},
		{ID: 3, Who: squirrel.SpeakerYou, Words: "the day before", SaidAt: at(t, "2026-08-30 11:00")},
		{ID: 4, Who: squirrel.SpeakerYou, Words: "this morning", SaidAt: at(t, "2026-08-31 08:30")},
	}
	vs := turnViews(context.Background(), turns)

	require.Equal(t, "Monday 24 August", vs[0].Day)
	require.Empty(t, vs[1].Day, "the day is said twice inside one day")
	require.Equal(t, "yesterday", vs[2].Day)
	require.Equal(t, "today", vs[3].Day)
}

// A fragment is appended under turns that are already on screen, so the
// divider the page path wants would repeat a day that is already said.
func TestAFragmentDoesNotRepeatTheDay(t *testing.T) {
	f := &fakeStore{}
	body := routed(t, f).callFragment(t, "/capture", "room=notes&text=the+boiler+code").Body.String()

	require.NotContains(t, body, `class="whenday"`,
		"a press draws the day again under the one already there")
	require.Contains(t, body, `class="whensaid"`, "a press draws no time at all")
}

// The clock is pinned, and it has to be: the room appends its own turn at now,
// so a fixture two hours old is the same day at noon and the day before at one
// in the morning. This test failed that way before it was pinned.
func TestTheConversationSaysWhenItWas(t *testing.T) {
	now = func() time.Time { return at(t, "2026-08-31 14:00") }
	t.Cleanup(func() { now = time.Now })

	f := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler code", SaidAt: at(t, "2026-08-31 12:00")},
	}}
	body := opened(t, f, "notes")

	require.Contains(t, body, `<time class="whensaid">12:00</time>`, "the conversation does not say when")
	require.Equal(t, 1, strings.Count(body, `class="whenday"`),
		"the day is said more than once for one day")
	require.Contains(t, body, `<p class="whenday">today</p>`)
}

// The check-in is drawn and never written, so asking every hour does not fill a
// record whose job is to hold what you said. Your answer is kept; the question
// is not.
func TestTheCheckinIsDrawnAndTheAnswerIsKept(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	body := m.call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, body, "how do you feel")
	require.Empty(t, f.appended, "the question was written into the record")

	m.callFragment(t, "/mood", "room=everything&mood=calm")

	require.NotEmpty(t, f.appended, "the answer was not kept")
	require.Equal(t, squirrel.MoodCalm, f.recorded, "the reading was not kept")
	for _, said := range f.appended {
		require.NotContains(t, said.Words, "how do you feel",
			"answering wrote the question into the record after all")
	}
}

// Sixteen a day is what an hourly question costs a record that keeps it. Drawn,
// it costs nothing however many times you arrive.
func TestArrivingAgainDoesNotStackTheQuestion(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	for i := 0; i < 5; i++ {
		body := m.call(t, "GET", "/r/everything", nil).Body.String()
		require.Equal(t, 1, strings.Count(body, "how do you feel"),
			"the question is on the screen more than once")
	}
	require.Empty(t, f.appended)
}
