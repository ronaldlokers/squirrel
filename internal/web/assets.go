package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFS embed.FS

// staticHandler serves the stylesheet, the script, the two fonts and the mark.
//
// A year of caching with no fingerprint in the filename is deliberate and
// bounded: this is one screen behind an ipAllowList used by one person, and a
// changed asset is one hard reload away. Fingerprinting would mean a build
// step, and this binary has none.
func staticHandler(opts Options) http.HandlerFunc {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	prefix := strings.TrimSuffix(opts.Path, "/") + "/static/"
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.StripPrefix(prefix, files).ServeHTTP(w, r)
	}
}
