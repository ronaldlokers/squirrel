//go:build browser

// The card's one question, in a real browser.
//
// The answers are behind a disclosure now, which is exactly the kind of change
// that quietly breaks the accelerators: a letter key looks for a button that
// is no longer on the screen. Go can see the markup and not the cascade, and
// the whole point of the gate is what is visible.
package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// visible is whether an element is actually shown, rather than merely present.
//
// checkVisibility rather than a bounding rect, and the difference is the whole
// reason this helper exists: a closed <details> keeps its contents in the DOM
// and Chrome still reports geometry for them, because the subtree is laid out
// and then skipped rather than removed. A rect test says every hidden answer
// is on screen.
const visible = `(sel) => {
	const el = document.querySelector(sel);
	return !!el && el.checkVisibility({
		checkOpacity: true, checkVisibilityCSS: true, contentVisibilityAuto: true,
	});
}`

func TestBrowserTheCardAsksOneQuestionFirst(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, true, c.eval(t, `return (`+visible+`)(".answer > summary")`),
		"the card did not ask its question")
	for _, answer := range []string{`[data-act="done"]`, `[data-act="keep"]`, `[data-act="drop"]`, `[data-act="task"]`} {
		require.Equal(t, false, c.eval(t, `return (`+visible+`)('`+answer+`')`),
			"%s was on screen before the question was answered", answer)
	}

	c.eval(t, `document.querySelector(".answer > summary").click()`)
	c.until(t, "the answers", `(`+visible+`)('[data-act="done"]')`)
	for _, answer := range []string{`[data-act="keep"]`, `[data-act="drop"]`, `[data-act="task"]`} {
		require.Equal(t, true, c.eval(t, `return (`+visible+`)('`+answer+`')`),
			"%s did not come with the others", answer)
	}
}

// Skipping must never cost an open: it is the one thing here that does nothing
// to a note, and it is what you press when you have no capacity to decide.
func TestBrowserSkippingNeedsNoOpen(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, true, c.eval(t, `return (`+visible+`)("#later")`),
		"skipping went behind the question")
	require.Equal(t, false, c.eval(t, `return document.querySelector("details.answer").open`))
}

// The letters still act from a shut card. A key is for the times you already
// knew what the note was, so making it wait for the question would take the
// accelerator away from the person it exists for.
func TestBrowserAKeyStillActsThroughTheShutGate(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, false, c.eval(t, `return document.querySelector("details.answer").open`))
	c.key(t, "d")
	c.until(t, "the stamp", `document.querySelector("#card").classList.contains("stamped")`)
	require.Equal(t, "DONE", c.eval(t, `return document.getElementById("stampText").textContent`))
	require.Equal(t, true, c.eval(t, `return (`+visible+`)("#undo")`),
		"the undo did not arrive")
}

// The interval question is the one key that reveals rather than acts, so it
// has to open the gate on the way — otherwise C opened a disclosure nested
// inside a shut one and nothing appeared.
func TestBrowserTheIntervalKeyOpensTheGate(t *testing.T) {
	c, _ := open(t, aPile())

	c.key(t, "c")
	c.until(t, "the interval chips", `(`+visible+`)("button[name=every]")`)
	require.Equal(t, true, c.eval(t, `return document.querySelector("details.answer").open`))
}

// Answered, the question goes with everything else in the tray: what is left
// on the card is the stamp and the way back.
func TestBrowserTheQuestionGoesOnceItIsAnswered(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".answer > summary").click()`)
	c.until(t, "the answers", `(`+visible+`)('[data-act="done"]')`)
	c.eval(t, `document.querySelector('[data-act="keep"]').click()`)
	c.until(t, "the stamp", `document.querySelector("#card").classList.contains("stamped")`)

	require.Equal(t, false, c.eval(t, `return (`+visible+`)(".answer > summary")`),
		"the question was still being asked after it was answered")
	require.Equal(t, false, c.eval(t, `return (`+visible+`)(".ways")`),
		"the repair was still offered after the note had gone")
}

// Correcting the words is not an answer, so it stays outside the question.
func TestBrowserFixingTheWordsNeedsNoOpen(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, true, c.eval(t, `return (`+visible+`)(".ways .reword > summary")`),
		"fixing the words went behind the question")
}

// Stopping, from the pile, in one press.
func TestBrowserStoppingIsOnePress(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector('.hint a[href="/enough"]').click()`)
	c.until(t, "the stopping screen", `location.pathname === "/enough"`)
	require.Contains(t, c.eval(t, `return document.body.textContent`), "that will do")
}
