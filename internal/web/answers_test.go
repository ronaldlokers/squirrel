//go:build browser

// The card's four answers, in a real browser.
//
// This file was gate_test.go, and it tested the opposite of what it now tests.
// For one day the four answers sat behind a disclosure that asked "what is
// this?" first — a change made on a review's suggestion, chosen from options,
// used, and then asked back: "the pile should have all its buttons visible
// instead of hidden behind a click."
//
// The gate is gone from the markup and the stylesheet. Its tests stayed, so
// five of them had been failing on this branch ever since — which is the whole
// argument for the file existing in this form rather than being deleted: what
// the owner asked for had no test holding it in place, and the tests that were
// there were holding the thing he rejected.
//
// checkVisibility rather than a bounding rect throughout, for the reason this
// package has now learnt three times: a closed <details> keeps its contents in
// the DOM and Chrome still reports geometry for them, so a rect test says every
// hidden answer is on screen.
package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// visible is whether an element is actually shown, rather than merely present.
const visible = `(sel) => {
	const el = document.querySelector(sel);
	return !!el && el.checkVisibility({
		checkOpacity: true, checkVisibilityCSS: true, contentVisibilityAuto: true,
	});
}`

// All four, on arrival, without a press.
//
// The gate's argument was that what stalls is not which of the four, it is
// whether you are deciding about this thing at all. That may even be true; it
// is not what it felt like to use. A press that reveals what you already know
// is there is a press that costs something every single time.
func TestBrowserTheCardShowsItsAnswersWithoutAPress(t *testing.T) {
	c, _ := open(t, aPile())

	for _, answer := range []string{
		`[data-act="done"]`, `[data-act="keep"]`, `[data-act="drop"]`, `[data-act="task"]`,
	} {
		require.Equal(t, true, c.eval(t, `return (`+visible+`)('`+answer+`')`),
			"%s was not on screen when the card arrived", answer)
	}

	require.Equal(t, false, c.eval(t, `return !!document.querySelector("details.answer")`),
		"the gate is back")
}

// Skipping must never cost an open: it is the one thing here that does nothing
// to a note, and it is what you press when you have no capacity to decide.
func TestBrowserSkippingNeedsNoOpen(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, true, c.eval(t, `return (`+visible+`)("#later")`),
		"skipping went behind something")
}

// The letters act from the card as it arrives. A key is for the times you
// already knew what the note was.
func TestBrowserAKeyActsOnTheCardAsItArrives(t *testing.T) {
	c, _ := open(t, aPile())

	c.key(t, "d")
	c.until(t, "the stamp", `document.querySelector("#card").classList.contains("stamped")`)
	require.Equal(t, "DONE", c.eval(t, `return document.getElementById("stampText").textContent`))
	require.Equal(t, true, c.eval(t, `return (`+visible+`)("#undo")`),
		"the undo did not arrive")
}

// The interval question is the one key that reveals rather than acts.
func TestBrowserTheIntervalKeyRevealsTheChips(t *testing.T) {
	c, _ := open(t, aPile())

	c.key(t, "c")
	c.until(t, "the interval chips", `(`+visible+`)("button[name=every]")`)
}

// Answered, the whole tray goes: what is left on the card is the stamp and the
// way back. The four answers and both repairs leave together, because none of
// them means anything about a note that has gone.
func TestBrowserTheTrayGoesOnceTheNoteIsAnswered(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector('[data-act="keep"]').click()`)
	c.until(t, "the stamp", `document.querySelector("#card").classList.contains("stamped")`)

	require.Equal(t, false, c.eval(t, `return (`+visible+`)('[data-act="done"]')`),
		"the answers were still offered after the note had been answered")
	require.Equal(t, false, c.eval(t, `return (`+visible+`)(".ways")`),
		"the repair was still offered after the note had gone")
}

// Correcting the words is not an answer, so it sits outside the row of them.
func TestBrowserFixingTheWordsNeedsNoOpen(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, true, c.eval(t, `return (`+visible+`)(".ways .reword > summary")`),
		"fixing the words went behind something")
}

// Stopping, from the pile, in one press.
func TestBrowserStoppingIsOnePress(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector('.hint a[href="/enough"]').click()`)
	c.until(t, "the stopping screen", `location.pathname === "/enough"`)
	// Today's wording of it. The line varies by the day now; what this presses
	// is the way there, and what it checks is that the screen says the thing.
	require.Contains(t, c.eval(t, `return document.body.textContent`),
		squirrel.Say(squirrel.SayingEnough, time.Now()))
}
