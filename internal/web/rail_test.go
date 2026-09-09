package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func theRail(t *testing.T, page string) string {
	t.Helper()
	from := strings.Index(page, `class="dayrail"`)
	require.GreaterOrEqual(t, from, 0, "the phone has no rail")
	to := strings.Index(page[from:], "</section>")
	require.GreaterOrEqual(t, to, 0, "the rail does not close")
	return page[from : from+to]
}

// The clock is frozen in all three: the rail orders an appointment's hour
// against the hour a part of the day begins, so a test run at five in the
// morning and one run at noon put the same two rows in a different order.
func atNine(t *testing.T) {
	t.Helper()
	was := now
	now = func() time.Time { return time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = was })
}

func TestTheRailIsTodayAndWhatIsNotTodayIsFurtherAhead(t *testing.T) {
	atNine(t)
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "the dentist", Starts: now().Add(2 * time.Hour)},
		{ID: 5, Label: "the school run", Starts: now().Add(30 * time.Hour)},
	}}

	rail := theRail(t, mounted(t, f).call(t, "GET", "/", nil).Body.String())
	seam := strings.Index(rail, "further ahead")

	require.GreaterOrEqual(t, seam, 0, "nothing separates today from what is not today")
	require.Less(t, strings.Index(rail, "the dentist"), seam,
		"a fixed point today is below the seam")
	require.Greater(t, strings.Index(rail, "the school run"), seam,
		"a fixed point that is not today is hanging on today's rail")
}

func TestTheHeadOfTheRailIsTheOneThatWantsYouNow(t *testing.T) {
	atNine(t)
	f := &fakeStore{
		chores: []squirrel.Chore{
			{ID: 1, Name: "bins out", Active: true, EverDone: true, Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 7},
		},
		usually:  map[int64]squirrel.Usually{1: {Part: squirrel.Morning}},
		upcoming: []squirrel.Moment{{ID: 4, Label: "the dentist", Starts: now().Add(2 * time.Hour)}},
	}

	rail := theRail(t, mounted(t, f).call(t, "GET", "/", nil).Body.String())
	head := rail[:strings.Index(rail, `class="hangs"`)]

	require.Contains(t, head, `class="atnow"`, "the rail has no head")
	require.Contains(t, head, "bins out", "the head is not the thing the morning wants")
	require.Contains(t, head, `class="stamps"`, "the head cannot be answered")
	require.NotContains(t, head, "the dentist", "two things are at the head")
}

func TestEverythingElseHangsOffTheRailInTheRacksOrder(t *testing.T) {
	atNine(t)
	f := &fakeStore{
		items:    []squirrel.Item{task(2, "book the MOT", squirrel.ItemOpen)},
		chores:   []squirrel.Chore{{ID: 1, Name: "wash the windows", Active: true, EveryDays: 28}},
		upcoming: []squirrel.Moment{{ID: 4, Label: "the dentist", Starts: now().Add(2 * time.Hour)}},
	}

	rail := theRail(t, mounted(t, f).call(t, "GET", "/", nil).Body.String())
	seam := strings.Index(rail, "whenever you like")

	require.GreaterOrEqual(t, seam, 0, "nothing off the rail is named")
	require.Greater(t, strings.Index(rail, "wash the windows"), seam,
		"a chore with no hour is hanging at one")
	require.Greater(t, strings.Index(rail, "book the MOT"), seam,
		"a thing you do once is hanging at an hour")
	require.Greater(t, strings.Index(rail, "everything you wrote down"), seam,
		"the way to the notes is not off the rail")
}
