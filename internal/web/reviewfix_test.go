//go:build browser

// Four things a cross-functional review found, three of which only exist in a
// browser.
//
// Each was reproduced before it was fixed: the sheet failing in silence was
// found by breaking `fetch` and watching nothing happen, the lost decision by
// pressing an action and then the way out and watching the store never hear
// about it, the keys behind the dialog by focusing the sheet's own cross and
// pressing `d`, and the focus ring by tabbing to a real control and measuring
// what the browser actually drew.
//
// So these do the same. They ask the browser what happened rather than reading
// the file, because in every one of the four cases the file looked right.
package web

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// contrast is the WCAG ratio between two `rgb(r, g, b)` strings, which is what
// getComputedStyle hands back. A focus indicator needs 3:1 against what it sits
// on (1.4.11), and reading it off the stylesheet is exactly how the two
// surfaces below were missed: the rule that fixes them is correct and simply
// does not name them.
// tabTo walks the focus there with the Tab key. It has to be the key: Chromium
// only matches `:focus-visible` after a keyboard interaction, so a scripted
// `.focus()` reads back the element's ordinary outline and would have measured
// a ring nobody ever sees. That is the same mistake as reading the stylesheet.
func tabTo(t *testing.T, c *cdp, sel string) {
	t.Helper()
	for i := 0; i < 60; i++ {
		if c.eval(t, fmt.Sprintf(`return document.activeElement?.matches(%q) === true`, sel)) == true {
			return
		}
		c.key(t, "Tab")
	}
	t.Fatalf("60 tabs never reached %s", sel)
}

func contrast(t *testing.T, c *cdp, sel, prop, against string) float64 {
	t.Helper()
	got := c.eval(t, fmt.Sprintf(`
		const lum = (css) => {
			const [r, g, b] = css.match(/\d+/g).map(Number).map(v => {
				const s = v / 255;
				return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
			});
			return 0.2126 * r + 0.7152 * g + 0.0722 * b;
		};
		const el = document.querySelector(%q);
		if (!el) throw new Error("no such element: " + %q);
		const a = lum(getComputedStyle(el).getPropertyValue(%q));
		const b = lum(getComputedStyle(el.closest(%q)).backgroundColor);
		const [hi, lo] = a > b ? [a, b] : [b, a];
		return Math.round(((hi + 0.05) / (lo + 0.05)) * 100) / 100;`,
		sel, sel, prop, against))
	ratio, ok := got.(float64)
	require.True(t, ok, "the ratio came back as %#v", got)
	return ratio
}

// Asking for help is the worst thing in this product to fail quietly at, and
// it was the only thing that did.
//
// The sheet's submit awaited a bare `fetch`, so a network that went while the
// question was in the air was an uncaught rejection: the button never moved,
// the box never cleared, and nothing appeared. The slot on home has said what
// happened since the day it was written, forty lines away in the same file.
func TestBrowserTheSheetSaysSoWhenItCannotReachBuddy(t *testing.T) {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	c, _ := openWith(t, f, &fakeCoach{reply: "one thing at a time."})

	// The sheet as it is actually met: opened over the deck by the acorn. The
	// standalone page is the scriptless path and posts by navigating, which
	// cannot fail this way.
	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet", `!!document.querySelector("dialog[open] .slot.say")`)

	// The network goes, in the one way the page cannot tell apart from a real
	// one: the request is made and never arrives.
	c.eval(t, `window.fetch = () => Promise.reject(new TypeError("Failed to fetch"))`)

	c.eval(t, `
		const box = document.querySelector(".slot.say textarea");
		box.value = "i cannot start any of it";
		box.form.requestSubmit(box.form.querySelector(".post"));`)
	c.until(t, "the sheet to say what happened",
		`!document.getElementById("saysaid")?.hidden`)

	require.Contains(t, c.eval(t, `return document.getElementById("saysaid").textContent`),
		"Your words are still here",
		"the sheet failed without saying so")
	require.Equal(t, "i cannot start any of it",
		c.eval(t, `return document.querySelector(".slot.say textarea").value`),
		"the words went with the failure")
	require.Equal(t, false, c.eval(t, `return document.querySelector(".slot.say .post").disabled`),
		"the way to try again is still disabled")
}

// A modal is modal for the keyboard too.
//
// Every letter on the deck is an action, and the keydown handler had no idea a
// dialog was open. With the sheet up and focus on its own cross — a button,
// which `typing()` does not exempt — pressing `d` stamped the card underneath,
// invisibly, because a modal was over it.
func TestBrowserKeysDoNotReachTheCardBehindTheSheet(t *testing.T) {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	c, _ := open(t, f)

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet", `!!document.querySelector("dialog[open]")`)
	c.eval(t, `document.querySelector("dialog[open] .shut").focus()`)

	c.key(t, "d")

	require.Equal(t, false,
		c.eval(t, `return document.getElementById("card")?.classList.contains("stamped") ?? false`),
		"a key pressed inside the sheet acted on the card behind it")
}

// The focus ring is visible on every surface a key can reach, not only the
// ones the override happened to name.
//
// `.chore` and `.sheet` take the same cream stock as the deck's card, and were
// left out of the rule that puts violet on cream — so the two most
// keyboard-dense surfaces outside the deck kept the orange-lit ring, at a
// measured 2.03:1 against the 3:1 a focus indicator owes.
func TestBrowserTheFocusRingIsVisibleOnEveryCreamSurface(t *testing.T) {
	f := aPile()
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true,
	}}
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	srv := screenWith(t, f, &fakeCoach{reply: "one thing at a time."})
	c := atChores(t, srv)

	tabTo(t, c, ".chore .abtn")
	onChore := contrast(t, c, ".chore .abtn", "outline-color", ".chore")
	require.GreaterOrEqual(t, onChore, 3.0,
		"the ring on a chore measures %.2f:1 against the card it sits on", onChore)

	// In the overlay, where the sheet really is cream stock. It is measured
	// there rather than on `/buddy` because the page route has no card behind
	// it any more — the conversation stands on the field — so `.sheet` is not
	// the ground there and measuring against it would be measuring against
	// nothing.
	c.navigate(t, srv.URL+"/")
	c.eval(t, `document.querySelector(".tobuddy").click(); return 1`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)
	tabTo(t, c, "dialog.coachsheet .sheet .post")
	inSheet := contrast(t, c, "dialog.coachsheet .sheet .post", "outline-color", "dialog.coachsheet .sheet")
	require.GreaterOrEqual(t, inSheet, 3.0,
		"the ring in Buddy's sheet measures %.2f:1 against the card it sits on", inSheet)

	// And on the page, where the ground under that same button is the slot.
	// The override keys off `.sheet` as an ancestor, so it still applies —
	// this is what proves it, rather than the treatment change quietly having
	// taken the ring back to orange-lit on a cream slot.
	c.navigate(t, srv.URL+"/buddy")
	tabTo(t, c, ".sheet .post")
	onSlot := contrast(t, c, ".sheet .post", "outline-color", ".sheet .slot")
	require.GreaterOrEqual(t, onSlot, 3.0,
		"the ring on Buddy's page measures %.2f:1 against the slot it sits in", onSlot)
}

// Three about the deck's card: a decision pending inside its hold, the chore
// disclosure's width on a phone, and the way out being shut while a stamp ran.
// The conversation has no hold and no disclosure, so none of the three has
// anything left to be true of.
