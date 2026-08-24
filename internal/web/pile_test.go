package web

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTheRouteTable pins the shape of the screen rather than describing it. The
// table is the spec's table, and a route that moves has to move here first.
func TestTheRouteTable(t *testing.T) {
	m := mounted(t, &fakeStore{})

	for _, route := range []string{
		"GET /{$}",
		"POST /capture",
		"POST /find",
		"POST /open",
		"POST /mood",
		"POST /now/act",
		"POST /now/stuck",
		"POST /pile/act",
		"POST /pile/later",
		"POST /pile/often",
		"POST /pile/reword",
		"POST /pile/why",
		"POST /pile/undo",
		"POST /timer",
		"POST /pile/chore",
		"POST /pile/fix",
		"POST /pile/split",
		"GET /buddy",
		"POST /buddy/say",
		"POST /buddy/close",
		"POST /buddy/badly",
		"POST /buddy/do",
		"GET /coach",
		"POST /steps",
		"GET /held",
		"GET /moods",
		"POST /held/act",
		"GET /kept",
		"GET /enough",
		"POST /tasks/act",
		"POST /tasks/new",
		"POST /chores/act",
		"POST /chores/often",
		"POST /chores/new",
		"GET /pile/chores",
		"GET /manifest.webmanifest",
		"GET /sw.js",
		"GET /static/",
		// What is coming, one of them, and the two things you can do to one.
		"GET /at/{id}",
		"POST /at/make",
		"POST /at/new",
		"POST /at/open",
		"POST /at/{id}/note",
		"POST /at/{id}/detach",
	} {
		require.Contains(t, m.routes, route, "the route table lost %s", route)
	}
	require.Len(t, m.routes, 44, "a route was added without being pinned here")
}

// Buddy was /coach for the release it shipped in, and the same rule applies:
// a bookmark that dies quietly is worse than a redirect nobody notices.
func TestTheOldCoachURLRedirects(t *testing.T) {
	m := mounted(t, &fakeStore{})

	w := m.call(t, "GET", "/coach", nil)
	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/buddy", w.Header().Get("Location"))

	// Carrying whatever it was asked with, so the acorn's own link survives.
	w = m.call(t, "GET", "/coach?from=%2F", nil)
	require.Equal(t, "/buddy?from=%2F", w.Header().Get("Location"))
}

// The chores screen lived at /pile/chores for its whole life, and a bookmark
// that dies quietly is worse than a redirect nobody notices.
func TestTheOldChoresURLRedirects(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/pile/chores", nil)

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	// Home, since the chores stopped being a page on 24 August 2026. The
	// redirect stays because the URL is in somebody's history.
	require.Equal(t, "/", w.Header().Get("Location"))
}

// A screen that captures with nowhere durable to put the words is the gap
// this closes, so it refuses at mount rather than at the first capture — which
// is the worst possible moment to find out.
func TestMountRefusesWithoutASpool(t *testing.T) {
	require.Error(t, Mount(newTestMux(), &fakeStore{}, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 },
	}))
}

func TestMountRefusesWithoutAnOwner(t *testing.T) {
	require.Error(t, Mount(newTestMux(), &fakeStore{}, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
	}))
}

// The deck came out on 24 August 2026 and what it did lives in the
// conversation. What it was tested for lives there too — see thread_test.go for
// one note at a time, the empty pile, and searching — except for these, which
// went with the screen:
//
//   * TestPileHasNoCaptureBox. The deck had no slot because home had the only
//     one. There is one dock now and it is on every view, so the rule it
//     guarded is retired rather than moved.
//   * TestSearchEscapesTheQuery. The query is not echoed into a page any
//     more; it is a turn, and TestTheSlotEscapesWhatItGivesBack is what pins
//     a turn's words being escaped.
