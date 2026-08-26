package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Things you cannot act on. A page until 25 August 2026 and a message since.

func aside(state squirrel.ItemState, id int64, text, because string) squirrel.HeldItem {
	return squirrel.HeldItem{ID: id, Text: text, State: state, Because: because}
}

// What the turn drew, as the record kept it.
func drewFor(t *testing.T, f *fakeStore, where string) string {
	t.Helper()
	f.appended = nil
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where="+where))
	require.Len(t, f.appended, 2)
	return string(f.appended[1].Shown) + " " + f.appended[1].Words
}

// Each card says which of the three it is and what would move it. The page
// carried that in headings; a turn cannot, because a heading with one row
// under it looks like a mistake.
func TestWhatYouSetAsideSaysWhichKindEachIs(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet"),
		aside(squirrel.ItemBlocked, 2, "fix the boiler", "the part"),
		aside(squirrel.ItemSomeday, 3, "learn to solder", ""),
	}}
	drew := drewFor(t, f, "held")

	require.Contains(t, drew, "WAITING ON")
	require.Contains(t, drew, "BLOCKED ON")
	require.Contains(t, drew, "SOMEDAY")
	require.Contains(t, drew, "ring the vet")
	require.Contains(t, drew, "the part")
}

// Someday is not waiting on anything, so its card says which kind it is and
// nothing else — rather than trailing a dash into nothing.
func TestSomethingWaitingOnNobodySaysOnlyWhatItIs(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemSomeday, 3, "learn to solder", ""),
	}}

	require.NotContains(t, drewFor(t, f, "held"), "SOMEDAY —")
}

// The two that something outside you will end, before the one only you can.
func TestWhatYouSetAsideKeepsTheCoresOrder(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemSomeday, 3, "learn to solder", ""),
		aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet"),
	}}
	drew := drewFor(t, f, "held")

	require.Less(t, strings.Index(drew, "ring the vet"), strings.Index(drew, "learn to solder"),
		"someday sat among things that are actually moving")
}

func TestNothingSetAsideReadsAsNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=held"))

	require.Contains(t, f.appended[1].Words, "Nothing set aside")
}

// No count anywhere, in either direction. A number beside stalled work is a
// reproach, and the point of setting it aside was to stop being asked about it.
func TestWhatYouSetAsideNeverEmitsACount(t *testing.T) {
	held := []squirrel.HeldItem{}
	for i := int64(1); i <= 7; i++ {
		held = append(held, aside(squirrel.ItemSomeday, i, "a thing", ""))
	}
	drew := drewFor(t, &fakeStore{aside: held}, "held")

	for _, count := range []string{"SOMEDAY (7)", "7 things", "(7)", "7 set aside"} {
		require.NotContains(t, drew, count)
	}
}

// That there is more, never how much more.
func TestWhatYouSetAsideSaysThereIsMoreWithoutSayingHowMuch(t *testing.T) {
	held := []squirrel.HeldItem{}
	for i := int64(1); i <= 25; i++ {
		held = append(held, aside(squirrel.ItemSomeday, i, "a thing", ""))
	}
	drew := drewFor(t, &fakeStore{aside: held}, "held")

	require.Contains(t, drew, "there are more")
	require.NotContains(t, drew, "13 more")
	require.NotContains(t, drew, "of 25")
	require.Equal(t, listLimit, strings.Count(drew, `"pick it back up"`),
		"the cap was not applied")
}

// The way back is the transition everything else here reverses through, and it
// answers in the conversation like every other press.
func TestPickingItBackUpReturnsIt(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet"),
	}}

	w := mounted(t, f).call(t, "POST", "/held/act", strings.NewReader("act=back&id=1&from=thread"))

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Equal(t, []int64{1}, f.unheld)
}

func TestSettingAsideFromTheCardTakesItOutOfThePile(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "ring the vet", squirrel.ItemOpen)}}

	w := mounted(t, f).call(t, "POST", "/held/act", strings.NewReader("aside=waiting&id=1"))

	require.Equal(t, 303, w.Code)
	// Back to the conversation, where Buddy says so — the way out used to
	// travel in this redirect, which is what made setting one aside the only
	// disposition with no undo. It travels with what was said now.
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Len(t, f.appended, 3)
	require.Contains(t, f.appended[1].Words, "Set aside")

	require.Len(t, f.aside, 1)
	require.Equal(t, squirrel.ItemWaiting, f.aside[0].State)
}

// A value that was never offered is read the way a stranger's typing is read.
func TestAnUnknownWayToSetSomethingAsideDoesNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "ring the vet", squirrel.ItemOpen)}}
	m := mounted(t, f)

	for _, bad := range []string{"dropped", "done", "parked", ""} {
		m.call(t, "POST", "/held/act", strings.NewReader("aside="+bad+"&id=1&from=%2F"))
	}
	require.Empty(t, f.aside)
}

// Not a fifth door. The rail is four and its equality is the whole statement
// it makes, and a door for what you set aside would put it back in front of
// you — which is the one thing setting it aside was for.
func TestSettingThingsAsideDidNotBecomeADoor(t *testing.T) {
	f := &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood},
		aside:   []squirrel.HeldItem{aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet")},
	}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	// What this was protecting is unchanged and is the second line: the things
	// you set aside are not *on* the conversation. Where you reach them from
	// did change — it was "not a fifth door on the rail" until 26 August 2026,
	// and there is no rail now, so it is a line in the menu like everywhere
	// else.
	require.Contains(t, body, "what you set aside", "there is no way to it at all")
	require.NotContains(t, body, "ring the vet",
		"what you set aside is on the conversation rather than behind a door")
}

// Reached from the tasks, which is where you look when you wonder what
// happened to something.
func TestTheTasksReachIt(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{task(1, "ring the vet", squirrel.ItemOpen)}}
	body := opened(t, f, "tasks")

	require.Contains(t, body, "what you set aside")
	require.Contains(t, body, `value="held"`)
}

// And the page itself is gone.
func TestThereIsNoPageForWhatYouSetAside(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.NotContains(t, m.routes, "GET /held")
}
