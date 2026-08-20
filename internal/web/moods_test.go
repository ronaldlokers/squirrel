package web

import (
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

	require.Contains(t, body, "today")
	require.Contains(t, body, "yesterday")
	require.Contains(t, body, "mood-good.png")
	require.Contains(t, body, "mood-wiped.png")
}

// Two answers on one day is a day you checked in twice, not two facts about
// you, so they share a day.
func TestTwoReadingsInADayShareTheDayOnScreen(t *testing.T) {
	now := time.Now()
	f := &fakeStore{readings: []squirrel.Checkin{
		reading(squirrel.MoodGood, now),
		reading(squirrel.MoodLow, now.Add(-2*time.Hour)),
	}}
	body := mounted(t, f).call(t, "GET", "/moods", nil).Body.String()

	require.Equal(t, 1, strings.Count(body, `class="when"`))
	require.Equal(t, 2, strings.Count(body, "mood-"))
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
func TestHomeLinksToItAndShowsNoSeries(t *testing.T) {
	now := time.Now()
	f := &fakeStore{
		checkin:  &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now},
		readings: []squirrel.Checkin{reading(squirrel.MoodWiped, now.AddDate(0, 0, -1))},
	}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `href="/moods"`)
	// Today's answer, and not yesterday's.
	require.Contains(t, body, "mood-good.png")
	require.NotContains(t, body, "mood-wiped.png")
}

// It is not in the lid and not a door. You go looking, or you do not see it.
func TestTheMoodsPageIsNotInTheLid(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	for _, path := range []string{"/pile", "/tasks", "/chores"} {
		body := mounted(t, f).call(t, "GET", path, nil).Body.String()
		require.NotContains(t, body, `href="/moods"`, "reachable from %s", path)
	}
}
