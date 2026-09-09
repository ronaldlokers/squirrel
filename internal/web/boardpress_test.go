//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// One chore that wants you today, so the rail has a head, and five things you
// do once behind the fold, which is where a phone's strips live now. Answered
// just now, so the check-in is not drawn: these are tests about where things
// sit, and the faces would move them without saying anything about either.
func aRackOfNotes() *fakeStore {
	f := &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
		usually: map[int64]squirrel.Usually{1: {Part: squirrel.Morning}},
		chores: []squirrel.Chore{{
			ID: 1, Name: "water the plants", Active: true, EverDone: true,
			Every: 24 * time.Hour, EveryDays: 1, SinceDays: 1,
		}},
	}
	for i := int64(1); i <= 5; i++ {
		f.items = append(f.items, squirrel.Item{
			ID: i, RawText: "note " + string(rune('a'+i-1)), State: squirrel.ItemOpen,
			Kind: squirrel.ItemTask, ReceivedAt: time.Now(),
		})
	}
	return f
}

// aLongRail is a day with more on it than one screen holds, so the deck has
// somewhere to scroll to.
func aLongRail() *fakeStore {
	f := aRackOfNotes()
	for i := int64(10); i < 26; i++ {
		f.chores = append(f.chores, squirrel.Chore{
			ID: i, Name: "one more thing", Active: true, EverDone: true,
			Every: 24 * time.Hour, EveryDays: 1, SinceDays: 1,
		})
		f.usually[i] = squirrel.Usually{Part: squirrel.Morning}
	}
	return f
}

// unfolded opens the phone's fold, which is the only place a strip is on a
// phone since the rail replaced the racks. The head is drawn whole and the
// rows hanging off the line carry no answers at all, by the design.
func unfolded(t *testing.T, c *cdp) {
	t.Helper()
	c.until(t, "the fold", `!!document.querySelector(".fold > summary")`)
	c.eval(t, `document.querySelector(".fold > summary").click(); return 1`)
	c.until(t, "the strips", `!!document.querySelector(".dayrail .strip.answerable")`)
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
	return c.eval(t, `return Math.round(document.querySelectorAll(".dayrail .strip.answerable")[`+
		string(rune('0'+nth))+`].querySelector(".stamps").getBoundingClientRect().height)`).(float64)
}

func TestBrowserAStripOpensWhenYouPressIt(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)
	unfolded(t, c)

	require.Equal(t, float64(0), stampsTall(c, t, 0), "a strip arrives with its answers already out")
	require.Equal(t, "false", c.eval(t, `return document.querySelector(".dayrail .opener").getAttribute("aria-expanded")`))

	c.eval(t, `document.querySelectorAll(".dayrail .strip.answerable")[0].querySelector(".what").click(); return 1`)
	c.until(t, "the stamps", `document.querySelectorAll(".dayrail .strip.answerable")[0]
		.querySelector(".stamps").getBoundingClientRect().height > 30`)
	require.Equal(t, "true", c.eval(t, `return document.querySelector(".dayrail .opener").getAttribute("aria-expanded")`))

	c.eval(t, `document.querySelectorAll(".dayrail .strip.answerable")[1].querySelector(".what").click(); return 1`)
	c.until(t, "the first to shut", `document.querySelectorAll(".dayrail .strip.answerable")[0]
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
	unfolded(t, c)

	c.eval(t, `document.querySelectorAll(".dayrail .strip.answerable")[0].querySelector(".what").click(); return 1`)
	c.until(t, "the stamps", `document.querySelectorAll(".dayrail .strip.answerable")[0]
		.querySelector(".stamps").getBoundingClientRect().height > 30`)

	c.eval(t, `document.querySelectorAll(".dayrail .strip.answerable")[0].querySelector(".stamp").click(); return 1`)
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
	unfolded(t, c)

	c.eval(t, `document.querySelectorAll(".dayrail .strip.answerable")[0].querySelector(".what").click(); return 1`)
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
	f := &fakeStore{items: []squirrel.Item{
		task(1, "book the MOT", squirrel.ItemOpen),
		task(2, "ring the vet back", squirrel.ItemOpen),
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)
	unfolded(t, c)

	c.key(t, "d")
	require.False(t, c.eval(t, `return !!document.querySelector(".strip.answerable.open")`).(bool),
		"a letter opened a strip before any strip had focus")

	c.key(t, "ArrowDown")
	c.until(t, "the first to open", `document.querySelector(".strip.answerable.open")
		?.querySelector(".what").textContent.trim().startsWith("book the MOT")`)

	c.key(t, "ArrowDown")
	c.until(t, "the second to open", `document.querySelector(".strip.answerable.open")
		?.querySelector(".what").textContent.trim().startsWith("ring the vet back")`)

	c.key(t, "d")
	c.until(t, "the strike", `!!document.querySelector(".strip.struck")`)
	require.Eventually(t, func() bool { return f.states[2] != "" },
		4*time.Second, 50*time.Millisecond, "the key did not act on the row a press had opened")
}

// The board's first screen is the work, and the deck scrolls past it. The
// pulled strip held the top until 9 September 2026, when the offer moved onto
// the row it is about; what has to give way now is the rail's own head.
func TestBrowserTheRailScrollsPastItsHead(t *testing.T) {
	f := aLongRail()
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the head", `!!document.querySelector(".atnow")`)

	deckTop := c.eval(t, `return Math.round(document.querySelector(".deck").getBoundingClientRect().top)`)
	require.Greater(t,
		c.eval(t, `return Math.round(document.querySelector(".atnow").getBoundingClientRect().bottom)`).(float64),
		deckTop.(float64), "the head is not on screen, so this measured nothing")

	c.eval(t, `document.querySelector(".deck").scrollTop = 600; return 1`)
	c.until(t, "the scroll", `document.querySelector(".deck").scrollTop > 0`)

	require.LessOrEqual(t,
		c.eval(t, `return Math.round(document.querySelector(".atnow").getBoundingClientRect().bottom)`).(float64),
		deckTop.(float64), "the head held the top of the board instead of giving way")
}

func TestBrowserTheBaysAreABarAtTheFoot(t *testing.T) {
	f := aLongRail()
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the bar", `getComputedStyle(document.querySelector(".addbar")).position === "fixed"`)

	foot := c.eval(t, `return Math.round(innerHeight - document.querySelector(".addbar").getBoundingClientRect().bottom)`)
	require.Less(t, foot.(float64), float64(16), "the bar is not floating at the foot of the screen")
	require.Greater(t, c.eval(t, `return Math.round(document.querySelector(".addbar").getBoundingClientRect().left)`).(float64),
		float64(0), "the bar runs edge to edge rather than floating clear of them")

	c.eval(t, `document.querySelector(".deck").scrollTop = 600; return 1`)
	c.until(t, "the scroll", `document.querySelector(".deck").scrollTop > 0`)
	require.Equal(t, foot,
		c.eval(t, `return Math.round(innerHeight - document.querySelector(".addbar").getBoundingClientRect().bottom)`),
		"the bar scrolled away with the rack")

	require.Equal(t, "add something", c.eval(t,
		`return document.querySelector(".addbar .words").placeholder`),
		"the bar at the foot is not the way in to the writer")
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
		c.eval(t, `return Math.round(document.querySelector(".addbar").getBoundingClientRect().top)`).(float64),
		"the floating bar covers the tray rather than clearing it")
}

// The chevron column rule went with the racks on the phone. Every row behind
// the fold is a thing you do once and wears the same mark, so there is no
// second width for a chevron to line up against; the desk has no chevrons at
// all. TestBrowserAStripOpensWhenYouPressIt still pins the chevron itself.

func TestBrowserTheStampsDoNotFlashOpenOnTheWayIn(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "press mode", `document.documentElement.classList.contains("presses")`)
	unfolded(t, c)

	c.until(t, "the easing", `document.documentElement.classList.contains("eased")`)
	require.NotEqual(t, "0s", c.eval(t, `return getComputedStyle(
		document.querySelector(".strip.answerable .acts")).transitionDuration`),
		"a press opens the strip with no motion at all")

	c.eval(t, `document.documentElement.classList.remove("eased"); return 1`)
	require.Equal(t, "0s", c.eval(t, `return getComputedStyle(
		document.querySelector(".strip.answerable .acts")).transitionDuration`),
		"the collapse carries its own motion, so the strips animate shut on the way in")
}

// The writer at the foot is paper, not smoke. The floating bar was a smoked
// purple pill while it was the tab bar's successor; the comps draw the writer
// as a field you type into, and a field has to be stock you can read ink on.
func TestBrowserTheWriterAtTheFootIsPaperAndRunsToTheEdges(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the bar", `getComputedStyle(document.querySelector(".addbar")).position === "fixed"`)

	require.Equal(t, "12px", c.eval(t, `return getComputedStyle(document.querySelector(".addbar")).left`),
		"the writer does not run to the phone's edges")
	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".addbar")).backdropFilter`),
		"the writer is smoked, so what you are typing sits over the board rather than on paper")
	require.Contains(t, c.eval(t, `return getComputedStyle(document.querySelector(".addbar")).backgroundColor`),
		"255", "the writer is not paper")
}

// The gap under the pill, measured with the inset the phone actually reports.
//
// Four defects in this bar were called invisible to CI on the grounds that
// env(safe-area-inset-bottom) reads zero in a headless browser. That was wrong:
// Emulation.setSafeAreaInsetsOverride sets it, and this project was already
// using it for the dock. Everything that was checked by rendering with a stub
// and looking is checkable here.
func TestBrowserThePillClearsTheHomeIndicator(t *testing.T) {
	// Enough to run past the foot of the screen: what is under test is what
	// the last thing on a full page reserves, and a page with ground to spare
	// under it answers the question with the ground rather than the rule.
	f := aRackOfNotes()
	for i := int64(1); i <= 12; i++ {
		f.chores = append(f.chores, squirrel.Chore{
			ID: 200 + i, Name: "one more thing", Active: true,
			EveryDays: 7, SinceDays: 7, EverDone: true,
		})
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.send(t, "Emulation.setSafeAreaInsetsOverride", map[string]any{
		"insets": map[string]any{"top": 59, "left": 0, "right": 0, "bottom": 34},
	})
	c.navigate(t, srv.URL+"/")
	c.until(t, "the bar", `getComputedStyle(document.querySelector(".addbar")).position === "fixed"`)

	// The reference app leaves 21 CSS px of ground under its bar, measured off
	// two screenshots. The 5px hard shadow is part of the object, so the box
	// sits 26 from the foot and the eye sees 21.
	require.Equal(t, float64(26), c.eval(t,
		`return Math.round(innerHeight - document.querySelector(".addbar").getBoundingClientRect().bottom)`),
		"the pill sits at a different height above the home indicator than it was measured to")

	// And the rail does not pad for the indicator as well, which is what put a
	// band of ground between the two on 2 September. The rail clears the pill
	// and nothing more, so the last row scrolls out from under it.
	require.LessOrEqual(t, c.eval(t,
		`const rail = document.querySelector(".dayrail");
		 return Math.round(parseFloat(getComputedStyle(rail).paddingBottom) -
			(innerHeight - document.querySelector(".addbar").getBoundingClientRect().top))`).(float64),
		float64(24), "something above the pill is reserving the indicator's band as well")
}

func TestBrowserTheBarReservesTheTopInsetAndNoMore(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.send(t, "Emulation.setSafeAreaInsetsOverride", map[string]any{
		"insets": map[string]any{"top": 59, "left": 0, "right": 0, "bottom": 34},
	})
	c.navigate(t, srv.URL+"/")
	c.until(t, "the bar", `!!document.querySelector(".ops .chip.bell")`)

	// The inset and nothing on top of it: the status bar's own band is the
	// margin, and a second one under it is space this screen cannot spare.
	require.Equal(t, float64(59), c.eval(t,
		`return Math.round(document.querySelector(".ops .chip.bell").getBoundingClientRect().top)`),
		"the bar reserves something other than exactly the top inset")
}
