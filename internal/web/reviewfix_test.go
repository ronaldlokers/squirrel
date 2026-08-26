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

	// And in the dock, where the ground under a button is the slot rather than
	// a card. The sheet was the third cream surface a key could reach and it
	// went on 25 August 2026; these two are what is left, and they are the two
	// that were wrong when this was written.
	c.navigate(t, srv.URL+"/")
	tabTo(t, c, ".dock .post")
	onSlot := contrast(t, c, ".dock .post", "outline-color", ".dock .slot")
	require.GreaterOrEqual(t, onSlot, 3.0,
		"the ring in the dock measures %.2f:1 against the slot it sits in", onSlot)
}

// Three about the deck's card: a decision pending inside its hold, the chore
// disclosure's width on a phone, and the way out being shut while a stamp ran.
// The conversation has no hold and no disclosure, so none of the three has
// anything left to be true of.
