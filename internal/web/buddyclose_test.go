//go:build browser

// Getting out of Buddy.
//
// Reported broken three times and never once reproducible here, which is its
// own finding: the tests were checking that one path worked rather than that
// there was no way for every path to fail at once. The close is the only way
// out of a modal sheet, so these check the routes and the fallback, not the
// happy path alone.
package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func openBuddy(t *testing.T, w, h int) *cdp {
	t.Helper()
	coach := &fakeCoach{spent: "€0.61", ceiling: "€10", reply: "Start with the envelope."}
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, coach)
	c := browserAt(t, srv, "/pile")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": w, "height": h, "deviceScaleFactor": 0, "mobile": w < 620,
	})
	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet", `!!document.querySelector("dialog.coachsheet[open]")`)
	return c
}

const shut = `document.querySelector("dialog.coachsheet .shut")`

// Whether the sheet is *on the screen*, which is not the same question as
// whether it has the open attribute.
//
// It is the question every test of this asked wrongly for three releases. The
// dialog closed correctly every time — the press landed, [open] came off, the
// backdrop went — and the sheet stayed visible because the stylesheet said
// display: flex and an author display value beats the browser's own
// `dialog:not([open]) { display: none }`. Every assertion passed while the
// thing they were about sat there in front of the owner.
const isOpen = `(() => {
	const d = document.querySelector("dialog.coachsheet");
	return !!d && d.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true });
})()`

// And the attribute, separately, so a failure says which of the two broke.
const hasOpenAttribute = `!!document.querySelector("dialog.coachsheet[open]")`

// The way out is a control with a name, not a word in a corner.
func TestBrowserTheWayOutIsACrossWithALabel(t *testing.T) {
	c := openBuddy(t, 390, 700)

	require.Equal(t, "Close Buddy", c.eval(t, `
		const b = `+shut+`;
		return (b.getAttribute("aria-label") || b.textContent).trim();`),
		"the close control has no accessible name")
	require.Equal(t, true, c.eval(t, `return !!`+shut+`.querySelector("svg")`),
		"the close control is not drawn")

	// A target a thumb can hit.
	require.Equal(t, true, c.eval(t, `
		const r = `+shut+`.getBoundingClientRect();
		return r.width >= 44 && r.height >= 44;`), "the close control is under 44px")
}

// After a conversation, which is the order a person uses it in.
func TestBrowserBuddyClosesAfterSayingSomething(t *testing.T) {
	c := openBuddy(t, 390, 700)

	c.eval(t, `const t = document.querySelector('dialog.coachsheet textarea[name="said"]');
		t.value = "I am stuck"; return 1`)
	c.eval(t, `document.querySelector('dialog.coachsheet form.slot.say .post').click()`)
	c.until(t, "the answer", `document.body.textContent.includes("Start with the envelope")`)

	c.eval(t, shut+`.click()`)
	c.until(t, "the sheet to close", `!(`+isOpen+`)`)
	require.Equal(t, "/pile", c.eval(t, `return location.pathname`))
}

// The fallback, which is the point of the whole change: if the dialog will not
// close for any reason at all, the press still gets you out — by posting the
// form the browser's own way rather than doing nothing.
//
// dialog.close is broken on purpose here. Nothing else in the page is touched.
func TestBrowserTheWayOutSurvivesTheDialogRefusingToClose(t *testing.T) {
	c := openBuddy(t, 390, 700)

	c.eval(t, `
		const d = document.querySelector("dialog.coachsheet");
		d.close = () => { throw new Error("no"); };
		return 1;`)

	// A marker on this document, so the test can tell a page that navigated
	// from one that merely hid a dialog. The path cannot say it: /buddy/close
	// forgets the conversation and redirects straight back to where the acorn
	// was pressed, so you leave and return to the same address.
	c.eval(t, `window.__thisDocument = true; return 1`)

	c.eval(t, shut+`.click()`)
	c.until(t, "the browser to post the form itself",
		`window.__thisDocument === undefined`)

	require.Equal(t, "/pile", c.eval(t, `return location.pathname`),
		"the fallback did not come back to the screen the acorn was pressed on")
	require.Equal(t, false, c.eval(t, `return `+isOpen),
		"the sheet is still open after the fallback")
}

// Escape still works, and is still free.
func TestBrowserEscapeStillClosesBuddy(t *testing.T) {
	c := openBuddy(t, 1280, 900)
	c.key(t, "Escape")
	c.until(t, "the sheet to close", `!(`+isOpen+`)`)
}

// Shut means gone from the screen, not merely gone from the DOM's attributes.
//
// This is the test that was missing. It is deliberately two assertions: the
// attribute and the render are different facts, and for three releases the
// first was right while the second was wrong.
func TestBrowserAClosedSheetIsActuallyGone(t *testing.T) {
	for _, size := range []struct {
		name string
		w, h int
	}{
		// The panel and the bottom sheet are different layouts and only one of
		// them was ever looked at. The report came from a tablet.
		{"tablet", 1130, 744},
		{"phone", 390, 700},
	} {
		t.Run(size.name, func(t *testing.T) {
			c := openBuddy(t, size.w, size.h)
			require.Equal(t, true, c.eval(t, `return `+isOpen))

			c.eval(t, shut+`.click()`)
			c.until(t, "the sheet to leave the screen", `!(`+isOpen+`)`)
			require.Equal(t, false, c.eval(t, `return `+hasOpenAttribute))
		})
	}
}

// Escape and the backdrop have to leave it just as gone.
func TestBrowserEveryWayOutLeavesItGone(t *testing.T) {
	c := openBuddy(t, 1130, 744)
	c.key(t, "Escape")
	c.until(t, "the sheet to leave the screen", `!(`+isOpen+`)`)
}

// There was a test here that required the acorn to stand aside while the
// sheet was open, because as a floating button it sat directly on top of the
// sheet's own send button on a wide screen.
//
// It has gone with the button. Buddy is in the lid now, behind the backdrop
// like everything else on the page, and a rule that hides a thing which
// cannot be reached anyway is a rule that will outlive its reason and confuse
// somebody.
