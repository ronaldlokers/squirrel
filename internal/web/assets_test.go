package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticServesTheStylesheetWithALongCache(t *testing.T) {
	h := staticHandler(Options{Path: "/pile"})
	r := httptest.NewRequest("GET", "/pile/static/pile.css", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/css")
	require.Contains(t, w.Header().Get("Cache-Control"), "max-age=")
	require.Contains(t, w.Body.String(), "--card: #fdecd4")
}

func TestStaticDoesNotEscapeItsDirectory(t *testing.T) {
	h := staticHandler(Options{Path: "/pile"})
	r := httptest.NewRequest("GET", "/pile/static/../../etc/passwd", nil)
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

// A year is the right cache for a file that exists. A miss is not a file, and
// caching one would keep a typo in a template pointing at nothing long after
// the asset it names has shipped.
func TestAMissingAssetIsNotCachedForAYear(t *testing.T) {
	h := staticHandler(Options{Path: "/pile"})
	r := httptest.NewRequest("GET", "/pile/static/nope.css", nil)
	w := httptest.NewRecorder()
	h(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Empty(t, w.Header().Get("Cache-Control"))
}
