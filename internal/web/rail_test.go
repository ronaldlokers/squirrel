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
	require.Contains(t, head, `class="bigstamps"`, "the head cannot be answered")
	require.NotContains(t, head, "the dentist", "two things are at the head")
}

// Off the rail is two rows and a count each: what you decided to do once, and
// what is on the wall. Everything behind them is not today's business, and a
// phone that lists it is a phone you scroll past to find today.
func TestOffTheRailIsTwoRowsAndTheirCounts(t *testing.T) {
	atNine(t)
	f := &fakeStore{
		items: []squirrel.Item{
			task(2, "book the MOT", squirrel.ItemOpen),
			task(3, "ring the vet back", squirrel.ItemOpen),
			note(4, "the boiler code", squirrel.ItemOpen),
		},
		chores:   []squirrel.Chore{{ID: 1, Name: "wash the windows", Active: true, EveryDays: 28}},
		upcoming: []squirrel.Moment{{ID: 5, Label: "the dentist", Starts: now().Add(2 * time.Hour)}},
	}

	rail := theRail(t, mounted(t, f).call(t, "GET", "/", nil).Body.String())
	seam := strings.Index(rail, "whenever you like")

	require.GreaterOrEqual(t, seam, 0, "nothing off the rail is named")
	require.Contains(t, rail[seam:], `<span class="tally">2</span>`,
		"the row does not say how many things you decided to do once")
	require.Contains(t, rail[seam:], "things you decided")
	require.Contains(t, rail[seam:], `<span class="tally">1</span>`,
		"the row does not say how many are on the wall")
	require.Contains(t, rail[seam:], "in the notes")
	require.Contains(t, rail[seam:], `href="/notes"`, "the notes row leads nowhere")

	require.NotContains(t, rail, "wash the windows",
		"a chore that is not asking today is on the phone")
	require.Greater(t, strings.Index(rail, "book the MOT"), seam,
		"a thing you do once is hanging on the rail")
}
