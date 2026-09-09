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
	require.Equal(t, "now", c.eval(t, `return [...document.querySelectorAll(".rack")]
		.find(r => getComputedStyle(r).display !== "none").dataset.bay`))

	c.navigate(t, srv.URL+"/?bay=weekly")
	require.Equal(t, "weekly", c.eval(t, `return [...document.querySelectorAll(".rack")]
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
		.filter(r => getComputedStyle(r).display !== "none").length`),
		"the desk draws a rack it has no use for, or is missing one")
	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector('.rack[data-bay="now"]')).display`),
		"now is a cut across the racks that come back, and the desk is showing those")
	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".baytabs")).display`))
}

func TestBrowserEveryBayIsOnTheScreen(t *testing.T) {
	srv := screen(t, &fakeStore{})
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, float64(6), c.eval(t, `return document.querySelectorAll(".baytab").length`))
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
	c := browserAt(t, srv, "/?bay=weekly")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})

	c.send(t, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 1})
	c.send(t, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": true, "configuration": "mobile"})
	c.navigate(t, srv.URL+"/?bay=weekly")
	require.True(t, c.eval(t, `return matchMedia("(hover: none) and (pointer: coarse)").matches`).(bool),
		"the browser is not pretending to be a touch screen, so this measured nothing")
	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".stamp .k")).display`),
		"a key is drawn on a screen that cannot press one")

	c.send(t, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": false})
	c.send(t, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": false})
	c.navigate(t, srv.URL+"/?bay=weekly")
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

	// Against the racks area it is in, rather than against the pill at the
	// foot: what sits between them — the tray, the check-in, the pill's own
	// reserve — moves without this promise changing, and measuring past it
	// made this a test about the tray.
	foot := c.eval(t, `return document.querySelector(".rack.in .channel").getBoundingClientRect().bottom`)
	under := c.eval(t, `return document.querySelector(".racks").getBoundingClientRect().bottom`)
	require.Greater(t, foot.(float64), under.(float64)-24,
		"the rack stops short and the rest of the racks area is nothing")
}

// The resting count only draws on a wiped or frazzled day, which no other
// screen in the appearance record is, so it is pinned here instead: it exists,
// it is quieter than the rows above it, and it never reads as a backlog.
func TestBrowserTheRestingCountIsDrawnAndIsQuiet(t *testing.T) {
	f := &fakeStore{
		checkin:  &squirrel.Checkin{Mood: squirrel.MoodWiped, SaidAt: time.Now()},
		capacity: squirrel.CapacityLow,
		chores: []squirrel.Chore{
			{ID: 1, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 7, EverDone: true},
			{ID: 2, Name: "water the plants", Active: true, EveryDays: 7, SinceDays: 2, EverDone: true},
			{ID: 3, Name: "wipe the sills", Active: true, EveryDays: 7, SinceDays: 1, EverDone: true},
		},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	c.navigate(t, srv.URL+"/")

	c.until(t, "the board", `!!document.querySelector(".rack")`)
	require.NotNil(t, c.eval(t, `return document.querySelector(".resting")`),
		"a wiped day asked as much of you as any other")
	require.Equal(t, "2 more further into the week",
		c.eval(t, `return document.querySelector(".resting").textContent.trim()`))
	require.Equal(t, "12px", c.eval(t, `return getComputedStyle(document.querySelector(".resting")).fontSize`),
		"the count is drawn at the rows' own weight, so it reads as one of them")
	require.NotEqual(t, c.eval(t, `return getComputedStyle(document.querySelector(".strip .what")).color`),
		c.eval(t, `return getComputedStyle(document.querySelector(".resting")).color`),
		"the count is as loud as a thing you could act on")

	for _, backlog := range []string{"outstanding", "left", "still", "waiting", "behind", "overdue"} {
		require.NotContains(t, c.eval(t, `return document.querySelector(".resting").textContent`), backlog,
			"the count reads as a backlog")
	}
}
