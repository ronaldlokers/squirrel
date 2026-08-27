package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestStaticServesTheStylesheetWithALongCache(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRequest("GET", "/static/pile.css", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/css")
	require.Contains(t, w.Header().Get("Cache-Control"), "max-age=")
	require.Contains(t, w.Body.String(), "--card: #fdecd4")
}

func TestStaticDoesNotEscapeItsDirectory(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRequest("GET", "/static/../../etc/passwd", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestFontsAreEmbedded(t *testing.T) {
	for _, name := range []string{"recursive.woff2", "inter-900.woff2", "logo.png"} {
		b, err := staticFS.ReadFile("static/" + name)
		require.NoError(t, err, name)
		require.NotEmpty(t, b, name)
	}
}

// Every embedded asset is one a page asks for. They are compiled into the
// binary and hashed into assetVersion, so one that nothing names is bytes
// shipped to every browser and a version stamp that churns for nothing.
//
// The mood drawings are named `mood-{{.Mood}}.png` in the template, so the
// five of them are matched by prefix rather than by a literal that does not
// appear anywhere.
func TestEveryEmbeddedAssetIsAskedForSomewhere(t *testing.T) {
	names, err := fs.ReadDir(staticFS, "static")
	require.NoError(t, err)

	// Every page, plus the two things the server writes rather than renders:
	// the manifest names the app icons and nothing else does.
	m := mounted(t, &fakeStore{})
	asks := m.call(t, "GET", "/manifest.webmanifest", nil).Body.String()
	for _, page := range templates(t) {
		asks += page
	}
	for _, f := range []string{"static/pile.css", "static/pile.js", "static/sw.js", "static/thread.js"} {
		b, err := staticFS.ReadFile(f)
		require.NoError(t, err)
		asks += string(b)
	}

	for _, entry := range names {
		name := entry.Name()
		switch {
		case strings.HasSuffix(name, ".css"), strings.HasSuffix(name, ".js"),
			name == "OFL.txt", strings.HasPrefix(name, "mood-"):
			continue
		}
		require.Contains(t, asks, name, "%s is embedded and nothing asks for it", name)
	}
}

// A year is the right cache for a file that exists. A miss is not a file, and
// caching one would keep a typo in a template pointing at nothing long after
// the asset it names has shipped.
func TestAMissingAssetIsNotCachedForAYear(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRequest("GET", "/static/nope.css", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Empty(t, w.Header().Get("Cache-Control"))
}

// The year-long cache is only safe if the URL changes when the file does.
// Without this, a release ships new markup against a stylesheet and a script
// the browser already has, and the screen is quietly broken until someone
// thinks to hard-reload — which is exactly what happened to v0.7.0.
func TestAssetURLsCarryAVersion(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "pile.css?v="+assetVersion)
	require.Contains(t, body, "pile.js?v="+assetVersion)
	require.Contains(t, body, "logo.png?v="+assetVersion)
	require.NotEmpty(t, assetVersion)
}

// The stamp is a property of the bytes, not of the clock or the build: two
// identical binaries must agree, or a rollout would refetch everything for
// nothing.
func TestTheVersionIsTheContent(t *testing.T) {
	require.Equal(t, assetVersion, stampOf(staticFS))
}

// A stamped URL still has to serve the file, and it must not be a 404 because
// of a query string the file server never asked about.
func TestAStampedAssetStillServes(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRequest("GET", "/static/pile.css?v="+assetVersion, nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "--card: #fdecd4")
}

// The worker holds a capture when there is no network, and sends it when there
// is. This is the nearest honest substitute for the spool a direct write does
// not have.
func TestTheWorkerHoldsACaptureWithNoNetwork(t *testing.T) {
	m := mounted(t, &fakeStore{})
	body := m.call(t, "GET", "/sw.js", nil).Body.String()

	require.Contains(t, body, `pathname === "/capture"`, "it intercepts the capture")
	require.Contains(t, body, "indexedDB", "and keeps the words somewhere real")
	require.Contains(t, body, "/?held=1", "and the page is told")
	// Deleted only once its own write has landed — a queue that keeps what it
	// has delivered is a second pile.
	require.Contains(t, body, "del.delete(note.key)")
}
