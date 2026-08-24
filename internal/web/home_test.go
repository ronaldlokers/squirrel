package web

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The load-bearing test. Home reads nothing, so a full pile and an empty one
// are the same bytes — which is the guard against this screen growing a
// preview, a badge or a count by accident. Any of the three would break it.
func TestHomeIsTheSameWhateverThePileHolds(t *testing.T) {
	full := mounted(t, &fakeStore{items: []squirrel.Item{
		note(1, "the tyre is flat", squirrel.ItemOpen),
		note(2, "ring the dentist", squirrel.ItemOpen),
	}})
	empty := mounted(t, &fakeStore{})

	require.Equal(t,
		empty.call(t, "GET", "/", nil).Body.String(),
		full.call(t, "GET", "/", nil).Body.String())
}

// Home has nothing on it that needs the database, so an unreachable one is not
// this screen's problem to report.
func TestHomeStandsUpWithoutTheDatabase(t *testing.T) {
	w := mounted(t, &fakeStore{err: errTest}).call(t, "GET", "/", nil)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "the pile")
	require.Contains(t, w.Body.String(), "the chores")
}

func TestHomeHasItsDoorsAndNothingElseToPress(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	// One door each, counted as doors rather than as links to a place.
	//
	// It used to count hrefs, on the rule that the lid must not carry a third
	// copy of a door. The rule holds and the counting no longer can: the lid's
	// map names all three places, so /pile appears twice in this markup and
	// only one of them is a door.
	//
	// What the rule was actually about is furniture — a second way in, sitting
	// on the screen, competing with the first. The map is behind a press and
	// is identical on every screen, so it is not that.
	require.Equal(t, 4, strings.Count(body, `class="door"`), "four doors, no more")
	require.Contains(t, body, `<details class="where">`,
		"the map is on the screen rather than behind a press")
}

// Everywhere else the mark is the way back. Home is where it points, so on home
// it is not a link.
func TestTheMarkGoesHomeFromTheOtherScreens(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.Contains(t, m.call(t, "GET", "/pile", nil).Body.String(), `<a class="brand" href="/">`)
	require.Contains(t, m.call(t, "GET", "/chores", nil).Body.String(), `<a class="brand" href="/">`)
	require.NotContains(t, m.call(t, "GET", "/", nil).Body.String(), `<a class="brand"`)
}

// The count rule does not stop at the deck. The capture rule did stop, on
// purpose: the owner overruled it on 20 August 2026 and the slot is the
// result, so what is pinned here is that there is exactly one of them and it
// is the slot.
func TestHomeNeverEmitsACount(t *testing.T) {
	m := mounted(t, &fakeStore{items: []squirrel.Item{
		note(1, "one", squirrel.ItemOpen),
		note(2, "two", squirrel.ItemOpen),
		note(3, "three", squirrel.ItemOpen),
	}})
	body := m.call(t, "GET", "/", nil).Body.String()
	lower := strings.ToLower(body)

	for _, total := range []string{"3 notes", "3 more", "(3)", "1 of ", "waiting"} {
		require.NotContains(t, lower, total)
	}
	// One slot, and one search field. Two places to type, and neither is a
	// second pile.
	require.Equal(t, 1, strings.Count(body, "<textarea"))
	require.Equal(t, 1, strings.Count(body, `action="/capture"`))
	typeable := strings.Count(body, "<input") - strings.Count(body, `<input type="hidden"`)
	require.Equal(t, 1, typeable, "the search field")
}

// / is the home screen and nothing else. A bare "/" pattern would be Go's
// catch-all, and a typo would arrive looking like a working page.
func TestHomeDoesNotAnswerForEverything(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.NotContains(t, m.routes, "GET /")
	require.Contains(t, m.routes, "GET /{$}")
}

// Three doors now, and they are still equals: one grid, identical cells, and
// nothing on any of them that depends on what is behind it.
func TestHomeHasFourDoors(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	for _, label := range []string{"the pile", "the tasks", "the chores", "the agenda"} {
		require.Contains(t, body, label)
	}
	require.Contains(t, body, "what you decided")
	// The one statement the screen makes is that they are equals, so nothing
	// may mark one of them out.
	//
	// Four since 24 August 2026. The fourth is what is coming, and it cost a
	// rule: PRODUCT.md refused a browsable list of appointments for this
	// product's whole life. What replaced the rule is that the list holds only
	// what is still ahead.
	require.Equal(t, 4, strings.Count(body, `class="door"`))
}

func TestHomeHasAWayToWhatIsComing(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `href="/at"`)
}

// The doors are equals, and a door is the most tempting place in the product to
// put a number on.
func TestTheDoorsStillCountNothing(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour)},
	}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	doors := body[strings.Index(body, `class="doors"`):]
	doors = doors[:strings.Index(doors, "</nav>")]

	// The words a person reads, not the markup around them: every door has
	// carried an image width since there were two of them, and a rule about
	// counting is a rule about what is shown.
	label := regexp.MustCompile(`<span class="(?:name|what)">([^<]*)</span>`)
	for _, m := range label.FindAllStringSubmatch(doors, -1) {
		require.NotRegexp(t, `\d`, m[1], "a door counted something: %q", m[1])
	}
	require.Len(t, label.FindAllStringSubmatch(doors, -1), 8, "four doors, a name and a line each")
}
