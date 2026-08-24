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

// The manifest answers without an identity, the way the worker does. Behind the
// guard it returned 403 to the one fetch that has no cookies to send, which
// leaves an installed app showing a letter tile and saying nothing about why.
func TestTheManifestDoesNotNeedAnIdentity(t *testing.T) {
	m := mounted(t, &fakeStore{})
	r := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	w := httptest.NewRecorder()
	m.routes["GET /manifest.webmanifest"](w, r)

	require.Equal(t, http.StatusOK, w.Code, "no header, still answered")
	require.Contains(t, w.Body.String(), "Squirrel")
}

// The installed app opens at home and its worker scopes to the whole screen.
// start_url was /pile until v0.10.0; an app installed before that still opens
// on the deck, which still serves, until the manifest is refetched.
func TestTheManifestKnowsWhereTheScreenIs(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/manifest.webmanifest", nil)

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
	require.Equal(t, "/", m.StartURL)
	require.Equal(t, "/", m.Scope)
	require.Equal(t, "standalone", m.Display)
	require.NotEmpty(t, m.Icons)
	for _, icon := range m.Icons {
		require.True(t, strings.HasPrefix(icon.Src, "/static/"), icon.Src)
	}
}

// A worker served from /static/ would only ever control /static/. Serving it
// from the root is what lets it answer for every screen — and, because the root
// is where it comes from, it needs no header to be allowed to.
func TestTheWorkerIsServedWhereItCanControlTheScreen(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/sw.js", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "javascript")
	require.Contains(t, w.Body.String(), "addEventListener")
	require.Empty(t, w.Header().Get("Service-Worker-Allowed"),
		"a worker from /sw.js already scopes to /; the header widened a scope that no longer needs widening")
}

// The pile is state. A cached page would show notes that have already been
// triaged, which is the two views disagreeing with each other — so the worker
// caches what cannot go stale and says so when it cannot reach the rest.
func TestTheWorkerNeverCachesThePileItself(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/sw.js", nil).Body.String()

	require.Contains(t, body, `url.pathname.includes("/static/")`,
		"only assets are ever put in the cache")
	require.Contains(t, body, "Nothing has been lost",
		"and when the network is gone it says so honestly")
}

func TestThePageOffersItselfForInstalling(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/kept", nil).Body.String()

	require.Contains(t, body, `rel="manifest" href="/manifest.webmanifest`)
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
	h := swHandler()
	r := httptest.NewRequest("GET", "/sw.js", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.NotContains(t, w.Header().Get("Cache-Control"), "31536000",
		"a worker nobody can replace is a worker you live with forever")
}

// The attribute that decides whether an installed app has an icon at all.
//
// A manifest is fetched with credentials omitted unless this says otherwise —
// same-origin makes no difference — so behind forward-auth the browser gets a
// login redirect, fails to parse it, and falls back to a letter tile. It logs
// nothing and never asks for the icons, which is why the icons looked wrong
// when the manifest was the thing that never arrived.
func TestTheManifestIsFetchedWithTheSession(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/kept", nil).Body.String()

	require.Contains(t, body, `rel="manifest"`)
	require.Regexp(t, `rel="manifest"[^>]*crossorigin="use-credentials"`, body)
}
