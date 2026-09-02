//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func aRackOfNotes() *fakeStore {
	f := &fakeStore{}
	for i := int64(1); i <= 5; i++ {
		f.items = append(f.items, squirrel.Item{
			ID: i, RawText: "note " + string(rune('a'+i-1)), State: squirrel.ItemOpen,
			Kind: squirrel.ItemNote, ReceivedAt: time.Now(),
		})
	}
	return f
}

func touching(t *testing.T, c *cdp) {
	t.Helper()
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.send(t, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 1})
	c.send(t, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": true, "configuration": "mobile"})
}

func stampsTall(c *cdp, t *testing.T, nth int) float64 {
	t.Helper()
	return c.eval(t, `return Math.round(document.querySelectorAll(".rack.in .strip.answerable")[`+
		string(rune('0'+nth))+`].querySelector(".stamps").getBoundingClientRect().height)`).(float64)
}

func TestBrowserAStripOpensWhenYouPressIt(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)

	require.Equal(t, float64(0), stampsTall(c, t, 0), "a strip arrives with its answers already out")
	require.Equal(t, "false", c.eval(t, `return document.querySelector(".opener").getAttribute("aria-expanded")`))

	c.eval(t, `document.querySelectorAll(".rack.in .strip.answerable")[0].querySelector(".what").click(); return 1`)
	c.until(t, "the stamps", `document.querySelectorAll(".rack.in .strip.answerable")[0]
		.querySelector(".stamps").getBoundingClientRect().height > 30`)
	require.Equal(t, "true", c.eval(t, `return document.querySelector(".opener").getAttribute("aria-expanded")`))

	c.eval(t, `document.querySelectorAll(".rack.in .strip.answerable")[1].querySelector(".what").click(); return 1`)
	c.until(t, "the first to shut", `document.querySelectorAll(".rack.in .strip.answerable")[0]
		.querySelector(".stamps").getBoundingClientRect().height < 1`)
	require.Equal(t, float64(1), c.eval(t, `return document.querySelectorAll(".strip.answerable.open").length`),
		"two strips are open at once")
}

func TestBrowserPressingAStampDoesNotShutTheStrip(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)

	c.eval(t, `document.querySelectorAll(".rack.in .strip.answerable")[0].querySelector(".what").click(); return 1`)
	c.until(t, "the stamps", `document.querySelectorAll(".rack.in .strip.answerable")[0]
		.querySelector(".stamps").getBoundingClientRect().height > 30`)

	c.eval(t, `document.querySelectorAll(".rack.in .strip.answerable")[0].querySelector(".stamp").click(); return 1`)
	c.until(t, "the strike", `!!document.querySelector(".strip.struck")`)
	require.Equal(t, float64(1), c.eval(t, `return document.querySelectorAll(".strip.answerable.open").length`),
		"pressing a stamp shut the strip it was on")
}

func TestBrowserEscapeShutsTheOpenStrip(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)

	c.eval(t, `document.querySelectorAll(".rack.in .strip.answerable")[0].querySelector(".what").click(); return 1`)
	c.until(t, "the stamps", `!!document.querySelector(".strip.answerable.open")`)

	c.key(t, "Escape")
	c.until(t, "nothing open", `!document.querySelector(".strip.answerable.open")`)
}

func TestBrowserWithNoScriptEveryStripStillCarriesItsAnswers(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.send(t, "Emulation.setScriptExecutionDisabled", map[string]any{"value": true})
	c.navigate(t, srv.URL+"/")

	require.False(t, c.eval(t, `return document.documentElement.classList.contains("presses")`).(bool),
		"the script ran, so this measured nothing")
	require.Greater(t, stampsTall(c, t, 0), float64(30),
		"with the script off a strip cannot be answered at all")
	require.Greater(t, stampsTall(c, t, 3), float64(30))
}

func TestBrowserTheKeysOpenTheStripTheyReach(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "bins out", Active: true, EverDone: true, Every: 14 * 24 * time.Hour, EveryDays: 14, SinceDays: 3},
		{ID: 2, PersonID: 1, Name: "water the ferns", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7},
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?bay=chores")
	touching(t, c)
	c.navigate(t, srv.URL+"/?bay=chores")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)

	c.key(t, "d")
	c.until(t, "the first chore to open", `document.querySelector(".strip.answerable.open")
		?.querySelector(".what").textContent.trim().startsWith("bins out")`)

	c.key(t, "ArrowDown")
	c.until(t, "the second chore to open", `document.querySelector(".strip.answerable.open")
		?.querySelector(".what").textContent.trim().startsWith("water the ferns")`)

	c.key(t, "d")
	c.until(t, "the strike", `!!document.querySelector(".strip.struck")`)
	require.Eventually(t, func() bool { return len(f.completed) == 1 },
		4*time.Second, 50*time.Millisecond, "the key did not act on the chore a press had opened")
}

func TestBrowserThePulledStripGivesWay(t *testing.T) {
	f := aRackOfNotes()
	for i := int64(10); i < 30; i++ {
		f.items = append(f.items, squirrel.Item{
			ID: i, RawText: "one more thing", State: squirrel.ItemOpen,
			Kind: squirrel.ItemNote, ReceivedAt: time.Now(),
		})
	}
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"}
	f.chores = []squirrel.Chore{{ID: 4, Name: "water the plants", Active: true, EveryDays: 7, SinceDays: 7}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the pulled strip", `!!document.querySelector(".pulled")`)

	deckTop := c.eval(t, `return Math.round(document.querySelector(".deck").getBoundingClientRect().top)`)
	require.Greater(t,
		c.eval(t, `return Math.round(document.querySelector(".pulled").getBoundingClientRect().bottom)`).(float64),
		deckTop.(float64), "the pulled strip is not on screen, so this measured nothing")

	c.eval(t, `document.querySelector(".deck").scrollTop = 600; return 1`)
	c.until(t, "the scroll", `document.querySelector(".deck").scrollTop > 0`)

	require.LessOrEqual(t,
		c.eval(t, `return Math.round(document.querySelector(".pulled").getBoundingClientRect().bottom)`).(float64),
		deckTop.(float64), "the pulled strip held the top of the board instead of giving way")
}

func TestBrowserTheBaysAreABarAtTheFoot(t *testing.T) {
	f := aRackOfNotes()
	for i := int64(10); i < 30; i++ {
		f.items = append(f.items, squirrel.Item{
			ID: i, RawText: "one more thing", State: squirrel.ItemOpen,
			Kind: squirrel.ItemNote, ReceivedAt: time.Now(),
		})
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?bay=notes")
	touching(t, c)
	c.navigate(t, srv.URL+"/?bay=notes")
	c.until(t, "the bar", `getComputedStyle(document.querySelector(".baytabs")).display === "grid"`)

	foot := c.eval(t, `return Math.round(innerHeight - document.querySelector(".baytabs").getBoundingClientRect().bottom)`)
	require.Less(t, foot.(float64), float64(16), "the bar is not floating at the foot of the screen")
	require.Greater(t, c.eval(t, `return Math.round(document.querySelector(".baytabs").getBoundingClientRect().left)`).(float64),
		float64(0), "the bar runs edge to edge rather than floating clear of them")

	c.eval(t, `document.querySelector(".deck").scrollTop = 600; return 1`)
	c.until(t, "the scroll", `document.querySelector(".deck").scrollTop > 0`)
	require.Equal(t, foot,
		c.eval(t, `return Math.round(innerHeight - document.querySelector(".baytabs").getBoundingClientRect().bottom)`),
		"the bar scrolled away with the rack")

	require.Equal(t, "notes", c.eval(t, `return document.querySelector(".baytab.in").getAttribute("href").split("=")[1]`),
		"the bar lights a bay you are not in")
	require.Equal(t, `["notes","chores","tasks","agenda"]`, c.eval(t, `return JSON.stringify(
		[...document.querySelectorAll(".baytab img")].map(i => i.getAttribute("src").split("bay-")[1].split(".png")[0]))`),
		"a bay wears another bay's icon")
}

func TestBrowserTheBarSitsUnderTheTray(t *testing.T) {
	f := aRackOfNotes()
	f.triaged = []squirrel.Item{{
		ID: 91, RawText: "the washing machine one", State: squirrel.ItemDone, Kind: squirrel.ItemNote,
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the tray", `!!document.querySelector(".tray")`)

	require.LessOrEqual(t,
		c.eval(t, `return Math.round(document.querySelector(".tray").getBoundingClientRect().bottom)`).(float64),
		c.eval(t, `return Math.round(document.querySelector(".baytabs").getBoundingClientRect().top)`).(float64),
		"the floating bar covers the tray rather than clearing it")
}

func TestBrowserEveryChevronSitsInTheSameColumn(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "remove settled dust", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 7},
		{ID: 2, Name: "take a shower", Active: true, Every: 24 * time.Hour, EveryDays: 1, SinceDays: 1},
		{ID: 3, Name: "wash the windows", Active: true, Every: 28 * 24 * time.Hour, EveryDays: 28, SinceDays: 28},
		{ID: 4, Name: "water the plants", Active: true, Every: 3 * 24 * time.Hour, EveryDays: 3, SinceDays: 3},
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?bay=chores")
	touching(t, c)
	c.navigate(t, srv.URL+"/?bay=chores")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)

	require.Greater(t, c.eval(t, `return new Set([...document.querySelectorAll(".rack.in .strip .mark")]
		.map(m => Math.round(m.getBoundingClientRect().width))).size`).(float64), float64(1),
		"every rhythm is the same width, so this measured nothing")

	require.Equal(t, float64(1), c.eval(t, `return new Set([...document.querySelectorAll(".rack.in .opener")]
		.map(o => Math.round(o.getBoundingClientRect().left))).size`),
		"the chevrons step in and out with the rhythm beside them")
}

func TestBrowserTheStampsDoNotFlashOpenOnTheWayIn(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)

	c.until(t, "the easing", `document.documentElement.classList.contains("eased")`)
	require.NotEqual(t, "0s", c.eval(t, `return getComputedStyle(
		document.querySelector(".strip.answerable .stamps")).transitionDuration`),
		"a press opens the strip with no motion at all")

	c.eval(t, `document.documentElement.classList.remove("eased"); return 1`)
	require.Equal(t, "0s", c.eval(t, `return getComputedStyle(
		document.querySelector(".strip.answerable .stamps")).transitionDuration`),
		"the collapse carries its own motion, so the strips animate shut on the way in")
}
