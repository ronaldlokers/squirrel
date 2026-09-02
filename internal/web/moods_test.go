package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func reading(m squirrel.Mood, at time.Time) squirrel.Checkin {
	return squirrel.Checkin{Mood: m, SaidAt: at}
}

// How you have been, asked for by name. A page until 31 August 2026 — the last
// screen in this product that was not a conversation — and a turn since, so
// asking for it does not take you out of the room you are standing in.
func shownMoods(t *testing.T, f *fakeStore) string {
	t.Helper()
	return routed(t, f).callFragment(t, "/me/moods",
		url.Values{"room": {"everything"}}.Encode()).Body.String()
}

func TestTheMoodsPageShowsWhatYouSaidAndWhen(t *testing.T) {
	now := time.Now()
	f := &fakeStore{readings: []squirrel.Checkin{
		reading(squirrel.MoodGood, now),
		reading(squirrel.MoodWiped, now.AddDate(0, 0, -1)),
	}}
	body := shownMoods(t, f)

	require.Contains(t, body, "this week")
	require.Contains(t, body, "mgood")
	require.Contains(t, body, "mwiped")
	// Six rows of seven, always, whatever you said.
	require.Equal(t, 6, strings.Count(body, `class="weekrow"`))
}

// midday is today, in the middle of it. The page says "today" and "yesterday"
// against the real clock, so the date has to be the real one — but the hour
// does not, and leaving it real is how a test starts depending on when it runs.
func midday() time.Time {
	y, m, d := time.Now().Date()
	return time.Date(y, m, d, 12, 0, 0, 0, time.Local)
}

// Two answers on one day is a day you checked in twice, not two facts about
// you, so they share a day — and on a grid a day is one square, showing what
// the day came to.
func TestTwoReadingsInADayShareTheDayOnScreen(t *testing.T) {
	// Midday, and not time.Now(): two readings two hours apart are only on the
	// same day if the first one is not within two hours of midnight. This
	// failed once, at 00:22, and would have failed every night between
	// midnight and two.
	now := midday()
	f := &fakeStore{readings: []squirrel.Checkin{
		reading(squirrel.MoodGood, now),
		reading(squirrel.MoodLow, now.Add(-2*time.Hour)),
	}}
	body := shownMoods(t, f)

	// One day, one cell: the last thing you said on it. The earlier answer is
	// still in the table and this page is not where it is reported.
	require.Equal(t, 1, strings.Count(body, `class="mgood" role="img"`))
	require.NotContains(t, body, `class="mlow" role="img"`)
}

// No average, no streak, no count. The interpretation is yours.
func TestTheMoodsPageSaysNothingAboutWhatItMeans(t *testing.T) {
	now := time.Now()
	readings := []squirrel.Checkin{}
	for i := 0; i < 5; i++ {
		readings = append(readings, reading(squirrel.MoodWiped, now.AddDate(0, 0, -i)))
	}
	body := shownMoods(t, &fakeStore{readings: readings})

	for _, judgement := range []string{
		"average", "streak", "in a row", "trend", "mostly", "5 days", "(5)",
	} {
		require.NotContains(t, strings.ToLower(body), strings.ToLower(judgement))
	}
}

func TestNoReadingsReadsAsNothing(t *testing.T) {
	require.Contains(t, shownMoods(t, &fakeStore{}), "not said how you are lately")
}

// Home shows today's answer and the way to the series, and the series is not
// on the screen until you ask.
//
// The way to it travels with Buddy's acknowledgement, because the answer is
// about to be scrollback and scrollback carries no controls. It is a press
// rather than a link since 31 August 2026, and the settings panel carries a
// second one — asking for it from either draws it where you are standing.
func TestTheThreadOffersIt(t *testing.T) {
	now := time.Now()
	f := &fakeStore{
		checkin:  &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now},
		readings: []squirrel.Checkin{reading(squirrel.MoodWiped, now.AddDate(0, 0, -1))},
	}
	m := mounted(t, f)
	post(t, m, "/mood", url.Values{"mood": {"good"}})
	f.turns, f.appended = f.appended, nil

	body := m.call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, body, `action="/me/moods"`)
	require.NotContains(t, body, "mood-wiped.png", "no series")
	require.NotContains(t, body, `class="weekrow"`, "the series is on the screen unasked")
}

// It is not in the lid and not a door. You go looking, or you do not see it.
func TestTheMoodsPageIsNotInTheLid(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	for _, path := range []string{"/", "/r/everything"} {
		body := mounted(t, f).call(t, "GET", path, nil).Body.String()
		require.NotContains(t, body, `href="/moods"`, "reachable from %s", path)
	}
}

// The gaps are the honest part, and the reason this is a grid. A day you said
// nothing on is drawn, not left out.
func TestDaysYouSaidNothingAreDrawn(t *testing.T) {
	f := &fakeStore{readings: []squirrel.Checkin{reading(squirrel.MoodGood, midday())}}
	body := shownMoods(t, f)

	require.Contains(t, body, "nothing said")
	// Six weeks less the one day answered, less the days of this week that
	// have not happened yet.
	ahead := 6 - int((time.Now().Weekday()+6)%7)
	require.Equal(t, 41-ahead, strings.Count(body, `class="nought" role="img"`))
}

// A day that has not happened is not a gap. Drawing next Saturday as an empty
// outline says you failed to check in on it.
func TestDaysThatHaveNotHappenedAreNotGaps(t *testing.T) {
	f := &fakeStore{readings: []squirrel.Checkin{reading(squirrel.MoodGood, midday())}}
	body := shownMoods(t, f)

	ahead := 6 - int((time.Now().Weekday()+6)%7)
	require.Equal(t, ahead, strings.Count(body, `class="ahead"`))
}

// The page is gone, and the URL still lands. It may be on a home screen.
func TestTheReadingsPageIsGoneAndItsURLStillLands(t *testing.T) {
	res := mounted(t, &fakeStore{}).call(t, "GET", "/moods", nil)

	require.Equal(t, 301, res.Code)
	require.Equal(t, "/r/everything", res.Header().Get("Location"))
	require.NotContains(t, res.Body.String(), "weekrow", "the page still draws")
}

// Asking for it answers in the room you asked from, which is the whole of why
// it stopped being a page.
func TestTheReadingsAnswerInTheRoomYouAskedFrom(t *testing.T) {
	f := &fakeStore{readings: []squirrel.Checkin{reading(squirrel.MoodGood, time.Now())}}
	res := routed(t, f).callFragment(t, "/me/moods",
		url.Values{"room": {"everything"}}.Encode())

	require.Equal(t, 200, res.Code)
	require.Contains(t, res.Body.String(), "weekrow")
	require.NotContains(t, res.Body.String(), "<!doctype html>", "it answered with a page")
	require.Len(t, f.appended, 2, "asking is something you did and is not in the record")
	for _, said := range f.appended {
		require.Equal(t, "everything", said.Room, "it answered in somebody else's room")
	}

	// And the scriptless press comes back to the same room rather than to the
	// front door — the floor every press on this screen stands on.
	back := routed(t, &fakeStore{}).call(t, "POST", "/me/moods",
		strings.NewReader(url.Values{"room": {"everything"}}.Encode()))
	require.Equal(t, 303, back.Code)
	require.Equal(t, "/r/everything", back.Header().Get("Location"))
}

// And a failure is a sentence in the conversation rather than a screen of
// nothing, which is what every other read on this product does.
func TestTheReadingsSayWhenTheyCannotBeRead(t *testing.T) {
	f := &fakeStore{readingsErr: errTest}
	res := routed(t, f).callFragment(t, "/me/moods",
		url.Values{"room": {"everything"}}.Encode())

	require.Equal(t, 200, res.Code)
	require.Contains(t, res.Body.String(), "cannot reach those just now")
	require.NotContains(t, res.Body.String(), "weekrow")
}
