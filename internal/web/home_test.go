package web

import (
	"strings"
	"testing"

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

func TestHomeHasTwoDoorsAndNothingElseToPress(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.Equal(t, 1, strings.Count(body, `href="/pile"`), "one door to the pile")
	require.Equal(t, 1, strings.Count(body, `href="/chores"`), "one door to the chores")
	// The lid's cross-link would be a third copy of a door.
	require.NotContains(t, body, `class="lidlink"`)
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
