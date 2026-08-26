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
		"POST /find/open",
		"POST /open",
		"POST /mood",
		"POST /now/act",
		"POST /now/stuck",
		"POST /pile/act",
		"POST /place/fresh",
		"POST /pile/later",
		"POST /pile/often",
		"POST /pile/reword",
		"POST /pile/why",
		"POST /pile/more",
		"POST /pile/undo",
		"POST /timer",
		"POST /pile/chore",
		"POST /pile/fix",
		"POST /pile/split",
		"POST /buddy/say",
		"POST /buddy/ask",
		"POST /find/ask",
		"POST /knowing",
		"POST /knowing/forget",
		"POST /chores/ask",
		"POST /chores/name",
		"POST /tasks/ask",
		"POST /pile/ask",
		"POST /buddy/badly",
		"POST /buddy/do",
		"GET /coach",
		"POST /steps",
		"GET /moods",
		"POST /held/act",
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
		// The way in, and the three routes that work it. The only routes
		// outside the guard besides the manifest, the worker and the static
		// files — necessarily, since a person with no session has to be able
		// to get one.
		"GET /auth",
		"POST /auth/in",
		"GET /auth/callback",
		"POST /auth/out",
	} {
		require.Contains(t, m.routes, route, "the route table lost %s", route)
	}
	require.Len(t, m.routes, 56, "a route was added without being pinned here")
}

// Both old addresses now answer with the conversation — see
// TestTheOldCoachURLsRedirect in coach_test.go. The query string is dropped
// rather than carried: it named the screen to come back to, and there is one.

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
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
	}))
}

func TestMountRefusesWithoutAnOwner(t *testing.T) {
	require.Error(t, Mount(newTestMux(), &fakeStore{}, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
	}))
}
