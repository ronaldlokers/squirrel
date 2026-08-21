//go:build browser

// Keeping something, without the page going anywhere.
//
// The complaint that produced these: "the page jumps up and you get a small
// text 'kept' outside of the box where you select the photo and add text".
// Both halves of that are one cause — the capture was a page navigation, so
// the browser left, came back at the top, and the answer to "did that work?"
// was rendered wherever the template happened to put it.
package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// scrolled is how far down the page is, so a test can say the page did not
// move rather than trusting that it did not.
const scrolled = `window.scrollY`

func TestBrowserKeepingSomethingDoesNotMoveThePage(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	// Somewhere other than the top, which is where you are when you have
	// scrolled past the doors to reach the slot.
	c.eval(t, `document.querySelector(".slot").scrollIntoView({block: "center"}); return 1`)
	where := c.eval(t, `return `+scrolled)

	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "the boiler again"; t.dispatchEvent(new Event("input")); return 1`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the slot to say it landed",
		`!document.querySelector("#slotsaid").hidden`)

	require.Len(t, sp.written, 1, "nothing reached the server")
	require.Equal(t, "the boiler again", sp.written[0].Text)

	require.Equal(t, where, c.eval(t, `return `+scrolled), "the page moved")
	require.Equal(t, "/", c.eval(t, `return location.pathname + location.search`),
		"the capture navigated")
}

// And it says so inside the box, not somewhere else on the page.
func TestBrowserTheSlotSaysItLandedInsideItself(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	require.Equal(t, true, c.eval(t, `
		return document.querySelector(".slot").contains(document.querySelector("#slotsaid"))`),
		"the answer is outside the box the action was in")

	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "kaas"; t.dispatchEvent(new Event("input")); return 1`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the slot to say it landed", `!document.querySelector("#slotsaid").hidden`)

	require.Equal(t, "kept", c.eval(t, `return document.querySelector("#slotsaid").textContent`))
	require.Equal(t, "", c.eval(t, `return document.querySelector(".slot textarea").value`),
		"the box kept the words after they were kept")
}

// A photograph goes the same way, and takes its preview with it.
func TestBrowserKeepingAPhotographClearsTheSlot(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the preview", `(`+visible+`)(".slot .gotphoto")`)
	c.until(t, "the photograph to be held", heldPhoto)

	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the slot to say it landed", `!document.querySelector("#slotsaid").hidden`)

	require.Len(t, sp.written, 1)
	require.NotEmpty(t, sp.written[0].PhotoName)
	require.Len(t, ph.kept, 1)
	require.Equal(t, false, c.eval(t,
		`return (`+visible+`)(".slot .gotphoto")`),
		"the preview stayed after the photograph was kept")
	require.Equal(t, false, c.eval(t, `return await (`+heldPhoto+`)`),
		"a photograph already kept is still being held")
}

// A failure keeps the words and says what happened, in the same place.
func TestBrowserAFailedCaptureKeepsWhatYouTyped(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{err: errTest}
	c, _ := openCamera(t, sp, ph)

	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "the tax letter"; t.dispatchEvent(new Event("input")); return 1`)
	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the preview", `(`+visible+`)(".slot .gotphoto")`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the slot to say what happened",
		`!document.querySelector("#slotsaid").hidden`)

	require.Contains(t, c.eval(t, `return document.querySelector("#slotsaid").textContent`),
		"photograph")
	require.Equal(t, "the tax letter", c.eval(t,
		`return document.querySelector(".slot textarea").value`),
		"a failed capture ate the words")
	require.Equal(t, true, c.eval(t,
		`return (`+visible+`)(".slot .gotphoto")`),
		"a failed capture ate the photograph")
	require.Empty(t, sp.written)
}

// An empty box, pressed, does nothing and says nothing. It did nothing.
func TestBrowserAnEmptySlotSaysNothing(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.eval(t, `return new Promise(r => setTimeout(r, 400))`)

	require.Equal(t, true, c.eval(t, `return document.querySelector("#slotsaid").hidden`),
		"an empty capture claimed something happened")
	require.Empty(t, sp.written)
	require.Equal(t, "/", c.eval(t, `return location.pathname + location.search`))
}
