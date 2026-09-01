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

	c.until(t, "the tabs", `getComputedStyle(document.querySelector(".baytabs")).display === "flex"`)
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
