package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sort"
)

//go:embed static
var staticFS embed.FS

// assetVersion is what makes the year-long cache below safe: it goes in every
// asset URL the templates write, so the URL changes exactly when the bytes do.
//
// This is not decoration. v0.7.0 shipped without it and the screen arrived
// broken: the markup is never cached, so a browser rendered new HTML against
// the stylesheet and script it already had — a link with no styling and a
// button whose handler did not exist yet, until someone knew to hard-reload.
//
// It is the content rather than the build, so two builds of the same tree
// agree and a rollout does not refetch everything for nothing.
var assetVersion = stampOf(staticFS)

// stampOf hashes every embedded file, name and contents, in a fixed order.
//
// Names are part of the hash so that renaming a file changes the stamp even
// when nothing inside it did — the fonts are referenced from inside the
// stylesheet rather than from a template, so replacing one means renaming it.
func stampOf(files embed.FS) string {
	sum := sha256.New()

	var names []string
	_ = fs.WalkDir(files, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		names = append(names, p)
		return nil
	})
	sort.Strings(names)

	for _, name := range names {
		f, err := files.Open(name)
		if err != nil {
			continue
		}
		_, _ = io.WriteString(sum, path.Base(name))
		_, _ = io.Copy(sum, f)
		_ = f.Close()
	}
	return hex.EncodeToString(sum.Sum(nil))[:10]
}

// staticHandler serves the stylesheet, the script, the two fonts and the mark.
//
// A year of caching is safe because every URL that reaches these files carries
// assetVersion, so a changed file is a changed URL. The stamp is a query
// string rather than a filename, which keeps the files themselves plain and
// needs no build step — this binary has none, and it does not want one.
func staticHandler() http.HandlerFunc {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.StripPrefix("/static/", files).ServeHTTP(w, r)
	}
}
