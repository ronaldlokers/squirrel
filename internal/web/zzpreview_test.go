//go:build browser

package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewDump(t *testing.T) {
	out := os.Getenv("PREVIEW_DIR")
	if out == "" {
		t.Skip("set PREVIEW_DIR")
	}
	m := &serveMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, everyScreen(), Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{},
	}))
	require.NoError(t, os.MkdirAll(filepath.Join(out, "static"), 0o755))
	entries, _ := os.ReadDir("static")
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join("static", e.Name()))
		_ = os.WriteFile(filepath.Join(out, "static", e.Name()), b, 0o644)
	}
	stamped := regexp.MustCompile(`/static/([A-Za-z0-9._-]+)\?v=[0-9a-f]+`)
	for name, path := range map[string]string{"at": "/at"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("X-Authentik-Username", "ronald")
		w := httptest.NewRecorder()
		m.mux.ServeHTTP(w, r)
		body := stamped.ReplaceAllString(w.Body.String(), "static/$1")
		body = strings.ReplaceAll(body, `href="/static/`, `href="static/`)
		body = strings.ReplaceAll(body, `src="/static/`, `src="static/`)
		require.NoError(t, os.WriteFile(filepath.Join(out, name+".html"), []byte(body), 0o644))
	}
}
