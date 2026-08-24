//go:build browser

// The chores screen's own picker, and the keys it was not given.
package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func twoChores() *fakeStore {
	f := aPile()
	f.chores = []squirrel.Chore{
		{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 6, Active: true, EverDone: true},
		{ID: 2, Name: "water the plants", Every: 4 * 24 * time.Hour, EveryDays: 4, SinceDays: 9, Active: true, EverDone: true},
	}
	return f
}

// Looking something up had a keyboard path and keeping a thought did not.
//
// `/` has focused the lid's search since it was written. Nothing focused the
// slot, so on the deliberate-desktop scene the first thing home asked of a
// keyboard user was to reach for the mouse — for the one act this product
// calls sacred.
func TestBrowserAKeyReachesTheSlotOnHome(t *testing.T) {
	f := aPile()
	srv := screen(t, f)
	c := browserAt(t, srv, "/")

	c.key(t, "t")

	require.Equal(t, "TEXTAREA", c.eval(t, `return document.activeElement.tagName`))
	require.Equal(t, true, c.eval(t, `return !!document.activeElement.closest(".slot")`),
		"t on home did not reach the slot")
}

// Retired on 24 August 2026, with the control they covered.
//
// The interval question was a `details.often` disclosure inside the chore card,
// and these pinned its behaviour: that opening it replaced the row rather than
// opening a modal, that 1-4 answered it, and that the arrows belonged to it
// while it was open. It is a turn of its own now — see askHowOften — with two
// radio rows and one submit, and none of those three facts has anything left to
// be true of.
//
// What went with it and is not replaced: the digit keys that answered the
// question without reaching for the mouse. Recorded in docs/roadmap.md rather
// than quietly dropped.

// Both were about the deck: a stamp on a card that is gone, and `t` meaning
// two things on a screen that had both a slot and a deck. There is one dock and
// no deck, so the collision cannot happen — TestBrowserAKeyInTheDockIsJustA
// Letter is what pins the remaining half.
