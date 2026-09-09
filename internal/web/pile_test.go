package web

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTheRouteTable pins the shape of the screen rather than describing it. The
// table is the spec's table, and a route that moves has to move here first.
func TestTheRouteTable(t *testing.T) {
	m := mounted(t, &fakeStore{})

	for _, route := range []string{
		"GET /{$}",
		"GET /r/{room}",
		"POST /capture",
		"GET /board",
		"POST /board/act",
		"POST /board/undo",
		"POST /board/new",
		"POST /board/now",
		"POST /board/capture",
		"POST /board/chore",
		"POST /board/mood",
		"POST /board/notuseful",
		"POST /board/ask",
		"POST /board/fix",
		"GET /notes",
		"GET /me",
		"GET /me/face",
		"POST /me/forget",
		"GET /coach",
		"GET /buddy",
		"POST /steps",
		"GET /at/{id}",
		"GET /moods",
		"GET /knowing",
		"GET /pile/chores",
		"POST /timer",
		// The way in, and the three routes that work it. The only routes
		// outside the guard besides the manifest, the worker and the static
		// files — necessarily, since a person with no session has to be able
		// to get one.
		"GET /auth",
		"POST /auth/in",
		"GET /auth/callback",
		"POST /auth/out",
		"GET /manifest.webmanifest",
		"GET /sw.js",
		"GET /static/",
	} {
		require.Contains(t, m.routes, route, "the route table lost %s", route)
	}
	require.Len(t, m.routes, 33, "a route was added without being pinned here")
}

// And the count above is the whole table rather than a number somebody bumped.
// GET /buddy was mounted, unlisted, and paid for by raising the total.
func TestTheRouteTableNamesEveryRoute(t *testing.T) {
	m := mounted(t, &fakeStore{})
	named := map[string]bool{}
	for _, route := range routesPinned(t) {
		named[route] = true
	}
	for route := range m.routes {
		require.True(t, named[route], "%s is mounted and not in the table above", route)
	}
}

// routesPinned reads the table above out of this file's own source, so the
// check below cannot drift from the list a person reads.
func routesPinned(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("pile_test.go")
	require.NoError(t, err)
	body := string(src)
	from := strings.Index(body, "for _, route := range []string{")
	require.Positive(t, from)
	to := strings.Index(body[from:], "\n\t} {")
	require.Positive(t, to)

	var out []string
	for _, m := range regexp.MustCompile(`"((?:GET|POST) [^"]+)"`).FindAllStringSubmatch(body[from:from+to], -1) {
		out = append(out, m[1])
	}
	return out
}

// The chores screen lived at /pile/chores for its whole life, and a bookmark
// that dies quietly is worse than a redirect nobody notices.
func TestTheOldChoresURLRedirects(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/pile/chores", nil)

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/?bay=daily", w.Header().Get("Location"))
}

// Everything Mount refuses to start without, and the refusal each one gives.
//
// Asserted on which refusal fires rather than on there being one: these were
// two tests with identical bodies, so the second passed on the first's check
// and neither the gate nor the login had a test at all. A missing dependency
// that fails as a different missing dependency is a test that passes for a
// reason nobody chose.
//
// Refused at mount rather than at first use, because first use is the worst
// moment to find out.
func TestMountRefusesWithoutWhatItNeeds(t *testing.T) {
	whole := func() Options {
		return Options{
			RequiredGroup: "squirrel-users", Gate: &Gate{},
			Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
			Login:    aTestLogin,
		}
	}
	require.NoError(t, Mount(newTestMux(), &fakeStore{}, whole()),
		"the whole set does not mount, so nothing below is testing what it says")

	for _, missing := range []struct {
		what string
		drop func(*Options)
		says string
	}{
		{"the required group", func(o *Options) { o.RequiredGroup = "" }, "WEB_REQUIRED_GROUP is empty"},
		{"the way in", func(o *Options) { o.Gate = nil }, "no way in"},
		{"the sessions", func(o *Options) { o.Sessions = nil }, "no sessions"},
		{"the login", func(o *Options) { o.Login = nil }, "turns a login into a person"},
	} {
		opts := whole()
		missing.drop(&opts)
		err := Mount(newTestMux(), &fakeStore{}, opts)
		require.Error(t, err, "it mounted without %s", missing.what)
		require.Contains(t, err.Error(), missing.says,
			"without %s it refused for some other reason", missing.what)
	}
}

// The helper resolves an overlap the way the server does.
//
// It picked the longest pattern, so "/r/{room}" beat "/r/everything" by one
// character and every test asking for Buddy's room reached the generic handler
// while the server reached his own. A helper that answers a different question
// from the product is worse than no helper.
func TestTheTestMuxPrefersTheSpecificRoute(t *testing.T) {
	m := newTestMux()
	m.Get("/r/everything", func(http.ResponseWriter, *http.Request) {})
	m.Get("/r/{room}", func(http.ResponseWriter, *http.Request) {})
	m.Get("/{$}", func(http.ResponseWriter, *http.Request) {})

	require.Equal(t, "GET /r/everything", m.route(t, "GET", "/r/everything"))
	require.Equal(t, "GET /r/{room}", m.route(t, "GET", "/r/chores"))
	require.Equal(t, "GET /{$}", m.route(t, "GET", "/"))
}
