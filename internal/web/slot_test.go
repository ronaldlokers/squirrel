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

// It used to mean the page did not move at all, because the slot was in the
// middle of a screen and posting it navigated you to the top of a fresh one.
// The dock is pinned and the answer arrives at the end of the conversation, so
// the page is *meant* to move — to the newest turn, which is what you just
// caused. What must not happen is a navigation.
func TestBrowserKeepingSomethingDoesNotTakeThePageAway(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.eval(t, `window.__stillHere = true; return 1`)
	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "the boiler again"; t.dispatchEvent(new Event("input")); return 1`)
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "Buddy to say it landed", landed)

	require.Len(t, sp.written, 1, "nothing reached the server")
	require.Equal(t, "the boiler again", sp.written[0].Text)

	// The assertion that matters, and the naive one — "a turn appeared" —
	// passes with the script deleted, because the redirect would put it there.
	require.Equal(t, true, c.eval(t, `return window.__stillHere === true`),
		"the page navigated; the swap did not happen")
	require.Equal(t, "/", c.eval(t, `return location.pathname + location.search`),
		"the capture navigated")
}

// And it says so as a turn, at the end of the conversation the press was made
// in. The answer used to be a word inside the box, which was where it belonged
// while the box was the whole interaction; the box is one end of a conversation
// now, and the other end is where answers come from.
func TestBrowserTheAnswerArrivesAsATurn(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "kaas"; t.dispatchEvent(new Event("input")); return 1`)
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "Buddy to say it landed", landed)

	// Whatever Buddy said, he said something. The wording varies by the day,
	// and since the box became a conversation it may be a sentence he wrote
	// rather than an acknowledgement at all — what this pins is that the
	// answer arrives as a turn rather than as a word inside the box.
	require.NotEmpty(t, c.eval(t,
		`return document.querySelector("#thread .turn:last-child .bub, #thread .turn:last-child .said").textContent.trim()`))
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

	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "Buddy to say it landed", landed)

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
	c.until(t, "Buddy to say what happened",
		`/not kept/i.test(document.querySelector("#thread .turn:last-child .bub, #thread .turn:last-child .said")?.textContent || "")`)

	require.Contains(t, c.eval(t,
		`return document.querySelector("#thread .turn:last-child .bub, #thread .turn:last-child .said").textContent`),
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

	// Buddy opens by asking how you are, so the conversation is not empty —
	// what must not happen is it growing.
	before := c.eval(t, `return document.querySelectorAll("#thread .turn").length`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.eval(t, `return new Promise(r => setTimeout(r, 400))`)

	require.Equal(t, before, c.eval(t, `return document.querySelectorAll("#thread .turn").length`),
		"an empty capture claimed something happened")
	require.Empty(t, sp.written)
	require.Equal(t, "/", c.eval(t, `return location.pathname + location.search`))
}

// thread.js announces the newest turn by reading it out of the DOM. Buddy's
// words stopped being a bubble on 26 August 2026, and the selector was left
// reading `.bub` alone for one commit — which found nothing for every turn
// Buddy appends, so the screen went quiet for exactly the person who cannot see
// it change. Nothing else in the suite noticed: the markup was right, the turn
// was there, and only the announcement was gone.
func TestBrowserTheLiveRegionSaysWhatBuddySaid(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "kaas"; t.dispatchEvent(new Event("input")); return 1`)
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "Buddy to say it landed", landed)

	said := c.eval(t,
		`return document.querySelector("#thread .turn:last-child .said").textContent.trim()`)
	require.NotEmpty(t, said)
	require.Equal(t, said, c.eval(t, `return document.getElementById("threadsay").textContent.trim()`),
		"the newest turn was not announced; the screen is silent for a screen reader")
}
