package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Two views, one pile. They may disagree about what is on screen; they may not
// disagree about what a note is.
//
// A card on the screen names the state it was drawn from, and a press carries
// that claim back. Between the draw and the press a `!drop` typed in the room
// can move the row — and without the claim the screen's write lands on top of
// it with nothing said to anybody.
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
	// And it said so — see TestAClashIsSaidInTheConversation.
	require.Equal(t, "/r/everything", res.Header().Get("Location"))
	require.Contains(t, f.appended[0].Words, "moved while you were looking at it",
		"and it went back saying nothing about it")
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

// A press that claims nothing has nothing to be stale about: the room's own
// commands carry no `was`, and they keep the write they had.
func TestAPressThatClaimsNothingWritesUnconditionally(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "the boiler", squirrel.ItemOpen)}}
	f.items[0].State = squirrel.ItemDropped

	res := mounted(t, f).call(t, "POST", "/pile/act",
		strings.NewReader(url.Values{"id": {"3"}, "act": {"done"}}.Encode()))

	require.NotContains(t, res.Header().Get("Location"), "clash")
	require.Equal(t, squirrel.ItemDone, f.states[3])
}

// It moved under you, and Buddy says so. Saying nothing would be the two views
// disagreeing and neither of them mentioning it.
func TestAClashIsSaidInTheConversation(t *testing.T) {
	// The note is already kept, so a press that claims it was open finds it
	// somewhere else — which is what the room having moved it looks like.
	f := &fakeStore{items: []squirrel.Item{note(9, "the boiler", squirrel.ItemKept)}}
	routed(t, f).call(t, "POST", "/pile/act",
		strings.NewReader("id=9&act=drop&was=open&from=thread"))

	require.NotEmpty(t, f.appended)
	require.Contains(t, f.appended[0].Words, "moved while you were looking at it")
}
