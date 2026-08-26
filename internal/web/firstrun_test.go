package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Somebody who has never seen this sees the loop played through once before
// being asked to trust it with anything.
func TestAFirstRunPlaysTheLoopThrough(t *testing.T) {
	body := mounted(t, answered(nil)).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `class="worked"`)
	require.Contains(t, body, "post the parcel back")
	require.Contains(t, body, "sort the recycling")
	require.Contains(t, body, "now yours.")
}

// It is drawn, never remembered. The first thing this product does must not be
// to write a false memory into a record whose whole value is that you made all
// of it.
func TestTheWorkedExampleIsNeverWritten(t *testing.T) {
	f := answered(nil)
	mounted(t, f).call(t, "GET", "/", nil)

	for _, turn := range f.appended {
		require.NotContains(t, turn.Words, "parcel")
		require.NotContains(t, string(turn.Shown), "parcel")
	}
}

// Inert by construction. The verbs are words, not controls: the one thing
// worse than not knowing what DONE does is finding out on somebody else's note.
func TestTheWorkedExampleHasNothingToPress(t *testing.T) {
	body := mounted(t, answered(nil)).call(t, "GET", "/", nil).Body.String()

	worked := body[strings.Index(body, `class="worked"`):strings.Index(body, `class="workedends"`)]
	require.NotContains(t, worked, "<form")
	require.NotContains(t, worked, "<button")
	require.NotContains(t, worked, "<a ")
}

// Anybody who has said one thing here never sees it again.
func TestSayingAnythingEndsTheWorkedExample(t *testing.T) {
	f := answered(nil)
	f.turns = []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler"}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "post the parcel back")
}

// And nor does somebody whose things arrived through Campfire without a word
// being said on this screen. A pile with things in it produces something to be
// handed, and a product mid-sentence about your own things does not need the
// loop explained.
func TestAFullPileEndsTheWorkedExample(t *testing.T) {
	f := answered(nil)
	f.waiting = squirrel.Waiting{Pile: 3}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "post the parcel back")
}

// Nor does anybody being handed something. An offer is Squirrel mid-sentence
// about your own things, and a worked example above it would be explaining a
// product that is already working.
func TestSomethingToBeHandedEndsTheWorkedExample(t *testing.T) {
	f := answered(nil)
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, Text: "ring the vet"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "post the parcel back")
}
