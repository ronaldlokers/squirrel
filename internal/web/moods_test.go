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

func TestTheMoodsPageShowsWhatYouSaidAndWhen(t *testing.T) {
	now := time.Now()
	f := &fakeStore{readings: []squirrel.Checkin{
		reading(squirrel.MoodGood, now),
		reading(squirrel.MoodWiped, now.AddDate(0, 0, -1)),
	}}
	body := mounted(t, f).call(t, "GET", "/moods", nil).Body.String()

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
	body := mounted(t, f).call(t, "GET", "/moods", nil).Body.String()

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
	body := mounted(t, &fakeStore{readings: readings}).call(t, "GET", "/moods", nil).Body.String()

	for _, judgement := range []string{
		"average", "streak", "in a row", "trend", "mostly", "5 days", "(5)",
	} {
		require.NotContains(t, strings.ToLower(body), strings.ToLower(judgement))
	}
}

func TestNoReadingsReadsAsNothing(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/moods", nil).Body.String()
	require.Contains(t, body, "not said how you are lately")
}

// Home shows today's answer and a link. The link is asking; the series lives
// behind it and nowhere else.
// The way to it travels with Buddy's acknowledgement, because the answer is
// about to be scrollback and scrollback carries no controls. Without it /moods
// would be reachable from nowhere in the product, which is the bug the old
// home screen had for an afternoon.
func TestTheThreadLinksToIt(t *testing.T) {
	now := time.Now()
	f := &fakeStore{
		checkin:  &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now},
		readings: []squirrel.Checkin{reading(squirrel.MoodWiped, now.AddDate(0, 0, -1))},
	}
	m := mounted(t, f)
	post(t, m, "/mood", url.Values{"mood": {"good"}})
	f.turns, f.appended = f.appended, nil

	body := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, `href="/moods"`)
	require.NotContains(t, body, "mood-wiped.png", "no series")
}

// It is not in the lid and not a door. You go looking, or you do not see it.
func TestTheMoodsPageIsNotInTheLid(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	for _, path := range []string{"/", "/r/notes"} {
		body := mounted(t, f).call(t, "GET", path, nil).Body.String()
		require.NotContains(t, body, `href="/moods"`, "reachable from %s", path)
	}
}

// The gaps are the honest part, and the reason this is a grid. A day you said
// nothing on is drawn, not left out.
func TestDaysYouSaidNothingAreDrawn(t *testing.T) {
	f := &fakeStore{readings: []squirrel.Checkin{reading(squirrel.MoodGood, midday())}}
	body := mounted(t, f).call(t, "GET", "/moods", nil).Body.String()

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
	body := mounted(t, f).call(t, "GET", "/moods", nil).Body.String()

	ahead := 6 - int((time.Now().Weekday()+6)%7)
	require.Equal(t, ahead, strings.Count(body, `class="ahead"`))
}
