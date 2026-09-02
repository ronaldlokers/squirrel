//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

// On a phone one rack is on screen and the bay signs are the tabs above it.
// Drawn as a class rather than filtered on the server, so this is the only
// place the difference can be seen at all.
func TestBrowserThePhoneShowsOneBayAtATime(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			{ID: 1, RawText: "kaas", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
		},
		chores: []squirrel.Chore{{ID: 7, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 7}},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")

	c.until(t, "the tabs", `getComputedStyle(document.querySelector(".baytabs")).display === "grid"`)
	require.Equal(t, float64(1), c.eval(t, `return [...document.querySelectorAll(".rack")]
		.filter(r => getComputedStyle(r).display !== "none").length`),
		"more than one rack is on the phone at once")
	require.Equal(t, "notes", c.eval(t, `return [...document.querySelectorAll(".rack")]
		.find(r => getComputedStyle(r).display !== "none").dataset.bay`))

	c.navigate(t, srv.URL+"/?bay=chores")
	require.Equal(t, "chores", c.eval(t, `return [...document.querySelectorAll(".rack")]
		.find(r => getComputedStyle(r).display !== "none").dataset.bay`),
		"the tab did not change which rack is on screen")
}

// And the desk still shows all four, which is the other half of one page
// serving both.
func TestBrowserTheDeskShowsEveryBay(t *testing.T) {
	srv := screen(t, &fakeStore{})
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false,
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, float64(4), c.eval(t, `return [...document.querySelectorAll(".rack")]
		.filter(r => getComputedStyle(r).display !== "none").length`))
	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".baytabs")).display`))
}

func TestBrowserEveryBayIsOnTheScreen(t *testing.T) {
	srv := screen(t, &fakeStore{})
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, float64(4), c.eval(t, `return document.querySelectorAll(".baytab").length`))
	require.Empty(t, c.eval(t, `return [...document.querySelectorAll(".baytab")]
		.filter(t => t.getBoundingClientRect().right > innerWidth + 0.5)
		.map(t => t.textContent.trim())`),
		"a bay sits off the right edge of the phone")
	require.Equal(t, float64(0), c.eval(t,
		`return document.documentElement.scrollWidth - document.documentElement.clientWidth`),
		"the board scrolls sideways")
}

// The field was drawn to nothing and opened by pressing its glyph, from
// 2 September 2026 until the bar became chips the same day. It is the middle of
// the bar now and always open, which is the shape the reference app has and one
// press cheaper. What the old test protected — that a field you cannot see is a
// field you cannot search with — is the assertion below.
func TestBrowserTheFieldIsThereWithoutBeingAskedFor(t *testing.T) {
	srv := screen(t, &fakeStore{})
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the bar", `!!document.querySelector(".ops .find input")`)

	require.Greater(t, c.eval(t,
		`return document.querySelector(".ops .find input").getBoundingClientRect().width`).(float64),
		float64(120), "the find field is drawn too small to type in")
	require.Equal(t, float64(16), c.eval(t,
		`return parseFloat(getComputedStyle(document.querySelector(".ops .find input")).fontSize)`),
		"a field under 16px makes iOS zoom the page when it takes focus")
}

func TestBrowserAKeycapIsNotDrawnWhereThereIsNoKeyboard(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 7, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 7},
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?bay=chores")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})

	c.send(t, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 1})
	c.send(t, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": true, "configuration": "mobile"})
	c.navigate(t, srv.URL+"/?bay=chores")
	require.True(t, c.eval(t, `return matchMedia("(hover: none) and (pointer: coarse)").matches`).(bool),
		"the browser is not pretending to be a touch screen, so this measured nothing")
	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".stamp .k")).display`),
		"a key is drawn on a screen that cannot press one")

	c.send(t, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": false})
	c.send(t, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": false})
	c.navigate(t, srv.URL+"/?bay=chores")
	require.NotEqual(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".stamp .k")).display`),
		"the key is gone where there is a keyboard to press it")
}

func TestBrowserTheLitRackReachesTheFootOfTheScreen(t *testing.T) {
	srv := screen(t, &fakeStore{})
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")

	foot := c.eval(t, `return document.querySelector(".rack.in .channel").getBoundingClientRect().bottom`)
	under := c.eval(t, `return document.querySelector(".baytabs").getBoundingClientRect().top`)
	require.Greater(t, foot.(float64), under.(float64)-24,
		"the rack stops short and the rest of the screen is nothing")
}
