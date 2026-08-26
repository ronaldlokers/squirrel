package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheIntervalQuestionAsksForADay(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{{ID: 3, Name: "the bins", Every: 7 * 24 * time.Hour}}}
	m := routed(t, f)

	m.call(t, "POST", "/chores/often", strings.NewReader("id=3"))

	shown := string(f.appended[len(f.appended)-1].Shown)
	require.Contains(t, shown, `"name":"day"`, "the question does not ask which day")
	require.Contains(t, shown, "thu")
	require.Contains(t, shown, "any day")
}

// A day named against weeks takes effect.
func TestNamingADayGivesTheChoreThatDay(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{{ID: 3, Name: "the bins", Every: 7 * 24 * time.Hour}}}
	m := routed(t, f)

	m.call(t, "POST", "/chores/act", strings.NewReader(
		url.Values{"id": {"3"}, "count": {"2"}, "unit": {"weeks"}, "day": {"thu"}}.Encode()))

	require.Len(t, f.rhythms, 1)
	require.Equal(t, time.Thursday, f.rhythms[0].Day)
	require.Equal(t, 2, f.rhythms[0].Weeks)
}

func TestTheRhythmIsSaidInWordsAPersonWouldUse(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{{ID: 3, Name: "the bins", Every: 7 * 24 * time.Hour}}}
	m := routed(t, f)

	m.call(t, "POST", "/chores/act", strings.NewReader(
		url.Values{"id": {"3"}, "count": {"2"}, "unit": {"weeks"}, "day": {"thu"}}.Encode()))

	var said []string
	for _, turn := range f.appended {
		said = append(said, turn.Words)
	}
	joined := strings.Join(said, " | ")
	require.Contains(t, joined, "every other thursday")
	require.NotContains(t, joined, "2 weeks on",
		"it said the rhythm in the machine's vocabulary")
}

// A day means nothing against days or months, and is refused rather than
// quietly changing the interval.
func TestADayIsRefusedWhereItWouldMeanNothing(t *testing.T) {
	for _, answer := range []url.Values{
		// The first two isolate the unit check. A count of 3 or 6 is refused
		// by the count check anyway, so a table without these would pass with
		// the unit check deleted — which is exactly what happened.
		{"id": {"3"}, "count": {"2"}, "unit": {"days"}, "day": {"thu"}},
		{"id": {"3"}, "count": {"1"}, "unit": {"months"}, "day": {"thu"}},
		{"id": {"3"}, "count": {"3"}, "unit": {"days"}, "day": {"thu"}},
		{"id": {"3"}, "count": {"6"}, "unit": {"months"}, "day": {"thu"}},
		{"id": {"3"}, "count": {"4"}, "unit": {"weeks"}, "day": {"thu"}},
		{"id": {"3"}, "count": {"2"}, "unit": {"weeks"}, "day": {"any day"}},
	} {
		f := &fakeStore{chores: []squirrel.Chore{{ID: 3, Name: "the bins", Every: 7 * 24 * time.Hour}}}
		m := routed(t, f)

		m.call(t, "POST", "/chores/act", strings.NewReader(answer.Encode()))

		require.Empty(t, f.rhythms, "a day was accepted against %s", answer.Encode())
	}
}

func TestTakingTheDayOffPutsItBackOnAnInterval(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 3, Name: "the bins", Every: 14 * 24 * time.Hour, Weekday: time.Thursday, Weeks: 2},
	}}
	m := routed(t, f)

	m.call(t, "POST", "/chores/act", strings.NewReader(
		url.Values{"id": {"3"}, "count": {"3"}, "unit": {"days"}, "day": {"any day"}}.Encode()))

	require.Len(t, f.rhythms, 1)
	require.Equal(t, 0, f.rhythms[0].Weeks, "the chore kept a day nobody asked for")
}

// The question opens on the day the chore already has.
func TestTheQuestionRemembersTheDay(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 3, Name: "the bins", Every: 14 * 24 * time.Hour, Weekday: time.Thursday, Weeks: 2},
	}}
	m := routed(t, f)

	m.call(t, "POST", "/chores/often", strings.NewReader("id=3"))

	shown := string(f.appended[len(f.appended)-1].Shown)
	require.Contains(t, shown, `"chosen":"thu"`,
		"the question forgot the day it was given last time")
}
