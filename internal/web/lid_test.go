//go:build browser

// The lid's two panels: the map and the search field.
//
// Both are <details>, which opens with no script at all — the floor. What the
// script adds is the part that makes a details behave like a menu: it closes
// when you press somewhere else, it closes on Escape, and two of them are
// never open at once.
package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// shown asks whether a thing is on the screen. Not whether it has an
// attribute: a panel with `open` removed and a stylesheet still displaying it
// is exactly the bug that made "closing Buddy does not work" survive three
// releases, and it passed every test that asked the DOM.
const shown = `(sel) => {
	const el = document.querySelector(sel);
	return !!el && el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true });
}`

func lid(t *testing.T) *cdp {
	t.Helper()
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, nil)
	c := browserAt(t, srv, "/pile")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 780, "deviceScaleFactor": 0, "mobile": true,
	})
	c.navigate(t, srv.URL+"/pile")
	return c
}

func TestBrowserTheMenuClosesWhenYouPressElsewhere(t *testing.T) {
	c := lid(t)

	c.eval(t, `document.querySelector("details.where > summary").click()`)
	c.until(t, "the map", `(`+shown+`)(".wherelist")`)

	// The card, which is the thing most likely to be under your thumb next.
	c.eval(t, `document.querySelector("#noteText").click()`)
	c.until(t, "the map to go", `!(`+shown+`)(".wherelist")`)
}

func TestBrowserTheSearchFieldClosesWhenYouPressElsewhere(t *testing.T) {
	c := lid(t)

	c.eval(t, `document.querySelector("details.findbox > summary").click()`)
	c.until(t, "the field", `(`+shown+`)(".findbox .find")`)

	c.eval(t, `document.querySelector("#noteText").click()`)
	c.until(t, "the field to go", `!(`+shown+`)(".findbox .find")`)
}

// Pressing inside must not close it — the field is inside the panel, and a
// menu that shuts when you reach for the thing it opened is worse than one
// that never shuts.
func TestBrowserPressingInsideAPanelKeepsItOpen(t *testing.T) {
	c := lid(t)

	c.eval(t, `document.querySelector("details.findbox > summary").click()`)
	c.until(t, "the field", `(`+shown+`)(".findbox .find")`)

	c.eval(t, `document.querySelector('.findbox input[type="search"]').click()`)
	c.eval(t, `return new Promise(r => setTimeout(r, 250))`)
	require.Equal(t, true, c.eval(t, `return (`+shown+`)(".findbox .find")`),
		"pressing the field closed the panel holding it")
}

// One at a time. Both hang off the same corner, so two open is two things to
// close and one of them covering the other.
func TestBrowserOnlyOnePanelIsOpenAtATime(t *testing.T) {
	c := lid(t)

	c.eval(t, `document.querySelector("details.where > summary").click()`)
	c.until(t, "the map", `(`+shown+`)(".wherelist")`)
	c.eval(t, `document.querySelector("details.findbox > summary").click()`)
	c.until(t, "the field", `(`+shown+`)(".findbox .find")`)

	require.Equal(t, false, c.eval(t, `return (`+shown+`)(".wherelist")`),
		"the map stayed open behind the search field")
}

// Escape, and the focus goes back to the control that opened it rather than
// being dropped on the page.
func TestBrowserEscapeClosesTheLidsPanels(t *testing.T) {
	c := lid(t)

	c.eval(t, `document.querySelector("details.where > summary").click()`)
	c.until(t, "the map", `(`+shown+`)(".wherelist")`)

	c.key(t, "Escape")
	c.until(t, "the map to go", `!(`+shown+`)(".wherelist")`)
	require.Equal(t, "SUMMARY", c.eval(t, `return document.activeElement.tagName`),
		"focus was dropped on the page rather than handed back")
}

// And with the script gone it is still a disclosure. The floor.
func TestBrowserThePanelsAreDisclosuresWithoutTheScript(t *testing.T) {
	c := lid(t)

	require.Equal(t, "DETAILS", c.eval(t,
		`return document.querySelector(".wherelist").closest("details").tagName`))
	require.Equal(t, "DETAILS", c.eval(t,
		`return document.querySelector(".findbox .find").closest("details").tagName`))
}
