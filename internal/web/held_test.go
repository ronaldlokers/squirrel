package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Things you cannot act on, on the screen.

func aside(state squirrel.ItemState, id int64, text, because string) squirrel.HeldItem {
	return squirrel.HeldItem{ID: id, Text: text, State: state, Because: because}
}

// Grouped, because they are three different questions.
func TestTheHeldPageGroupsTheThree(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet"),
		aside(squirrel.ItemBlocked, 2, "fix the boiler", "the part"),
		aside(squirrel.ItemSomeday, 3, "learn to solder", ""),
	}}
	body := mounted(t, f).call(t, "GET", "/held", nil).Body.String()

	require.Contains(t, body, "WAITING ON")
	require.Contains(t, body, "BLOCKED ON")
	require.Contains(t, body, "SOMEDAY")
	require.Contains(t, body, "ring the vet")
	require.Contains(t, body, "the part")
}

// A group with nothing in it does not render, so the page is never a set of
// empty headings telling you what you have not got.
func TestAnEmptyGroupIsAbsentRatherThanEmpty(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemSomeday, 3, "learn to solder", ""),
	}}
	body := mounted(t, f).call(t, "GET", "/held", nil).Body.String()

	require.Contains(t, body, "SOMEDAY")
	require.NotContains(t, body, "WAITING ON")
	require.NotContains(t, body, "BLOCKED ON")
}

func TestNothingSetAsideReadsAsNothing(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/held", nil).Body.String()
	require.Contains(t, body, "nothing set aside")
}

// No count anywhere, on any heading, in either direction.
func TestTheHeldPageNeverEmitsACount(t *testing.T) {
	held := []squirrel.HeldItem{}
	for i := int64(1); i <= 7; i++ {
		held = append(held, aside(squirrel.ItemSomeday, i, "a thing", ""))
	}
	body := mounted(t, &fakeStore{aside: held}).call(t, "GET", "/held", nil).Body.String()

	for _, count := range []string{"SOMEDAY (7)", "7 things", "(7)", "7 set aside"} {
		require.NotContains(t, body, count)
	}
}

// That there is more, never how much more.
func TestTheHeldPageSaysThereIsMoreWithoutSayingHowMuch(t *testing.T) {
	held := []squirrel.HeldItem{}
	for i := int64(1); i <= 25; i++ {
		held = append(held, aside(squirrel.ItemSomeday, i, "a thing", ""))
	}
	body := mounted(t, &fakeStore{aside: held}).call(t, "GET", "/held", nil).Body.String()

	// The cap is what makes the line truthful; the line is what makes the cap
	// bearable. Asserted on the sentence and on the number of rows actually
	// drawn, rather than by fishing for digits in a page full of markup — the
	// first version of this test looked for "25" and matched the stylesheet.
	require.Contains(t, body, "there is more.")
	require.NotContains(t, body, "5 more")
	require.NotContains(t, body, "of 25")
	require.Equal(t, heldLimit, strings.Count(body, `name="act" value="back"`),
		"the cap was not applied")
}

// The way back is the transition everything else here reverses through.
func TestPickingItBackUpReturnsIt(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{
		aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet"),
	}}
	m := mounted(t, f)

	w := m.call(t, "POST", "/held/act", strings.NewReader("act=back&id=1&from=%2Fheld"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/held", w.Header().Get("Location"))
	require.Equal(t, []int64{1}, f.unheld)
}

// The card offers the three, in a disclosure rather than three more buttons in
// a row that is already four answers and a picker.
func TestTheCardOffersTheThreeInADisclosure(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "ring the vet", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "i can't act on this")
	require.Contains(t, body, `name="aside" value="waiting"`)
	require.Contains(t, body, `name="aside" value="blocked"`)
	require.Contains(t, body, `name="aside" value="someday"`)
}

func TestSettingAsideFromTheCardTakesItOutOfThePile(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "ring the vet", squirrel.ItemOpen)}}
	m := mounted(t, f)

	w := m.call(t, "POST", "/held/act", strings.NewReader("aside=waiting&id=1&from=%2Fpile"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/pile", w.Header().Get("Location"))

	require.Len(t, f.aside, 1)
	require.Equal(t, squirrel.ItemWaiting, f.aside[0].State)
}

// A value that was never offered is read the way a stranger's typing is read.
func TestAnUnknownWayToSetSomethingAsideDoesNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "ring the vet", squirrel.ItemOpen)}}
	m := mounted(t, f)

	for _, bad := range []string{"dropped", "done", "parked", ""} {
		m.call(t, "POST", "/held/act", strings.NewReader("aside="+bad+"&id=1&from=%2Fpile"))
	}
	require.Empty(t, f.aside)
}

// Its own page, not a fourth door. Home carries three doors and an offer, and
// a door for what you set aside would put it back in front of you.
func TestSettingThingsAsideDidNotBecomeAFourthDoor(t *testing.T) {
	f := &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood},
		aside:   []squirrel.HeldItem{aside(squirrel.ItemWaiting, 1, "ring the vet", "the vet")},
	}
	home := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, home, `href="/held"`)
	require.NotContains(t, home, "ring the vet")
}

// Reached from the bottom of the tasks, in the line that already reaches the
// archive — which is where you look when you wonder what happened to something.
func TestTheTasksScreenReachesIt(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{task(1, "ring the vet", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/tasks", nil).Body.String()

	require.Contains(t, body, `href="/held"`)
	require.Contains(t, body, "what you cannot act on")
}
