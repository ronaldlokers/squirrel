package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheManifestKnowsWhereTheScreenIs(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/pile/manifest.webmanifest", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "manifest+json")

	var m struct {
		Name     string                 `json:"name"`
		StartURL string                 `json:"start_url"`
		Scope    string                 `json:"scope"`
		Display  string                 `json:"display"`
		Icons    []struct{ Src string } `json:"icons"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	require.Equal(t, "Squirrel", m.Name)
	require.Equal(t, "/pile", m.StartURL)
	require.Equal(t, "/pile", m.Scope)
	require.Equal(t, "standalone", m.Display)
	require.NotEmpty(t, m.Icons)
	for _, icon := range m.Icons {
		require.True(t, strings.HasPrefix(icon.Src, "/pile/static/"), icon.Src)
	}
}

// A worker served from /pile/static/ would only ever control /pile/static/.
// Serving it from the screen's own path is what lets it answer for the screen.
func TestTheWorkerIsServedWhereItCanControlTheScreen(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/pile/sw.js", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "javascript")
	require.Contains(t, w.Body.String(), "addEventListener")
	require.Equal(t, "/pile", w.Header().Get("Service-Worker-Allowed"),
		"without this the worker controls everything under the screen except the screen")
}

// The pile is state. A cached page would show notes that have already been
// triaged, which is the two views disagreeing with each other — so the worker
// caches what cannot go stale and says so when it cannot reach the rest.
func TestTheWorkerNeverCachesThePileItself(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile/sw.js", nil).Body.String()

	require.Contains(t, body, `url.pathname.includes("/static/")`,
		"only assets are ever put in the cache")
	require.Contains(t, body, "Nothing has been lost",
		"and when the network is gone it says so honestly")
}

func TestThePageOffersItselfForInstalling(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `rel="manifest" href="/pile/manifest.webmanifest`)
	require.Contains(t, body, `name="theme-color"`)
	require.Contains(t, body, `rel="apple-touch-icon"`)
}

func TestTheIconsAreEmbedded(t *testing.T) {
	for _, name := range []string{"icon-192.png", "icon-512.png", "apple-touch-icon.png"} {
		b, err := staticFS.ReadFile("static/" + name)
		require.NoError(t, err, name)
		require.NotEmpty(t, b, name)
	}
}

func TestTheWorkerIsNotBehindTheYearLongCache(t *testing.T) {
	h := swHandler(Options{Path: "/pile"})
	r := httptest.NewRequest("GET", "/pile/sw.js", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.NotContains(t, w.Header().Get("Cache-Control"), "31536000",
		"a worker nobody can replace is a worker you live with forever")
}
