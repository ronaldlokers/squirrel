package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The deck holds a tapped action for about a second and a half so the undo has
// a card to sit on. For that whole window the row is untouched — so a `!drop`
// typed in the room inside it wrote the truth, and the screen's write then
// landed on top of it with nothing said to anybody.
//
// Two views, one pile. They may disagree about what is on screen; they may not
// disagree about what a note is.
func TestADecisionThePileHasOvertakenDoesNotOverwriteIt(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(3, "the boiler makes a noise", squirrel.ItemOpen),
	}}

	// The room got there first, inside the hold.
	f.items[0].State = squirrel.ItemDropped

	res := mounted(t, f).call(t, "POST", "/pile/act",
		strings.NewReader(url.Values{
			"id":  {"3"},
			"act": {"done"},
			// What the card was showing when the press happened.
			"was": {"open"},
		}.Encode()))

	require.Equal(t, squirrel.ItemDropped, f.items[0].State,
		"the screen's stale decision overwrote the one made in the room")
	require.Contains(t, res.Header().Get("Location"), "clash=3",
		"and it went back saying nothing about it")
}

// And the screen says so, rather than quietly showing a different pile than
// the one you were deciding about.
func TestThePileSaysWhenItHasMovedUnderYou(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "the boiler", squirrel.ItemOpen)}}

	body := mounted(t, f).call(t, "GET", "/pile?clash=3", nil).Body.String()

	require.Contains(t, body, "that note had already moved")
}

// Saying a thing twice is saying it. A second identical press — or a
// redelivered webhook — finds the note already where it was being sent, and
// that is a success, not a collision. Any variant of this that reports
// "already done" turns a retry into a failure.
func TestPressingTheSameThingTwiceIsStillOneDecision(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "the boiler", squirrel.ItemOpen)}}
	body := func() *strings.Reader {
		return strings.NewReader(url.Values{
			"id": {"3"}, "act": {"done"}, "was": {"open"},
		}.Encode())
	}

	first := mounted(t, f).call(t, "POST", "/pile/act", body())
	require.NotContains(t, first.Header().Get("Location"), "clash")

	// The card is gone by now, so the second press carries the same claim the
	// first did — which is exactly what a retry looks like.
	again := mounted(t, f).call(t, "POST", "/pile/act", body())

	require.Equal(t, squirrel.ItemDone, f.items[0].State)
	require.NotContains(t, again.Header().Get("Location"), "clash",
		"a second press of the same button was reported as a collision")
}

// A form written before any of this said nothing about what it was showing,
// and chat has no hold and so no stale window. Both keep the write they had.
func TestAPressThatClaimsNothingWritesUnconditionally(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "the boiler", squirrel.ItemOpen)}}
	f.items[0].State = squirrel.ItemDropped

	res := mounted(t, f).call(t, "POST", "/pile/act",
		strings.NewReader(url.Values{"id": {"3"}, "act": {"done"}}.Encode()))

	require.NotContains(t, res.Header().Get("Location"), "clash")
	require.Equal(t, squirrel.ItemDone, f.states[3])
}

// The card has to say what it is showing, or none of the above can happen.
func TestTheCardSaysWhatItIsShowing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "the boiler", squirrel.ItemOpen)}}

	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `name="was" value="open"`)
}
