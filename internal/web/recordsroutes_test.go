package web

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"time"
)

var aRouteInProse = regexp.MustCompile("`(/[a-z0-9{}/_.-]*)`")

var saidOnPurpose = map[string]string{
	"/r/everything":        "named as the address the retired room used to have",
	"/buddy/badly":         "named as the press that retired with the room",
	"/v1/responses":        "somebody else's route: the model API this calls out to",
	"/v1/chat/completions": "somebody else's route: the model API this calls out to",
}

func theRoutesMounted(t *testing.T) map[string]bool {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, &fakeStore{}, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Location: time.Local,
	}))
	out := map[string]bool{"/": true}
	for pattern := range m.routes {
		path := pattern
		if _, after, found := strings.Cut(pattern, " "); found {
			path = after
		}
		out[strings.TrimSuffix(path, "{$}")] = true
		out[path] = true
	}
	return out
}

func TestTheBindingRecordsNameNoRouteThatIsGone(t *testing.T) {
	mounted := theRoutesMounted(t)

	for _, name := range []string{"../../PRODUCT.md", "../../DESIGN.md", "../../docs/roadmap.md"} {
		body, err := os.ReadFile(name)
		require.NoError(t, err)

		for _, m := range aRouteInProse.FindAllStringSubmatch(string(body), -1) {
			route := m[1]
			if strings.Contains(route, ".") || strings.HasSuffix(route, "/") {
				continue
			}
			if _, allowed := saidOnPurpose[route]; allowed {
				continue
			}
			if mounted[route] || mounted[route+"/"] || mounted[route+"/{$}"] {
				continue
			}
			require.Fail(t,
				"a binding record names a route that is not mounted",
				"%s says %s. Either the record is out of date, or the route was "+
					"renamed and nothing followed it. If the record names it "+
					"deliberately as something gone, or it belongs to another "+
					"service, add it to saidOnPurpose with the reason.", name, route)
		}
	}
}

func TestNothingIsExemptedThatIsActuallyMounted(t *testing.T) {
	mounted := theRoutesMounted(t)

	for route, why := range saidOnPurpose {
		require.False(t, mounted[route] || mounted[route+"/"] || mounted[route+"/{$}"],
			"%s is exempted as %q and is mounted. The exemption is stale, and while "+
				"it stands nothing checks what the records say about that route.",
			route, why)
	}
}
