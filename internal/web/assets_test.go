package web

import (
	"net/http"
	"net/http/httptest"
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

// The door art is part of the screen, not part of the comp: a home page whose
// illustrations 404 is a home page with two empty slots.
func TestTheDoorArtIsServed(t *testing.T) {
	m := mounted(t, &fakeStore{})

	for _, name := range []string{"door-pile.png", "door-chores.png"} {
		w := m.call(t, "GET", "/static/"+name, nil)
		require.Equal(t, http.StatusOK, w.Code, name)
		require.NotEmpty(t, w.Body.Bytes(), name)
	}
}

func TestFontsAreEmbedded(t *testing.T) {
	for _, name := range []string{"recursive.woff2", "inter-900.woff2", "logo.png"} {
		b, err := staticFS.ReadFile("static/" + name)
		require.NoError(t, err, name)
		require.NotEmpty(t, b, name)
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
	body := mounted(t, f).call(t, "GET", "/kept", nil).Body.String()

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
