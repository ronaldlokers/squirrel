package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
var assetVersion = stampOf(assetsFS())

// devDir is empty in every build that ships.
//
// Templates and static files are compiled in, which is right for a binary with
// no build step and wrong for looking at a change: editing pile.css does
// nothing to a running process, so impeccable's live mode, the detector's
// overlay and any by-hand test of the service worker all had nowhere to run.
// When this is set, both are read from disk instead and nothing is cached.
//
// Only cmd/devscreen sets it, through EnableDevelopment in dev.go, and both
// are behind the `dev` build tag — so a production build does not contain the
// code that could turn this on. See docs/roadmap.md.
var devDir string

// assetsFS is what /static/ is served from: the embedded copy, or the working
// tree when devDir is set.
func assetsFS() fs.FS {
	if devDir != "" {
		return os.DirFS(filepath.Join(devDir, "static"))
	}
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

// stamp is the asset version to write into a URL.
//
// Fixed for a shipped binary, because the bytes cannot change under it. Read
// fresh on every call in development, so an edited stylesheet arrives at the
// browser rather than being answered from a year-long cache.
func stamp() string {
	if devDir != "" {
		return stampOf(assetsFS())
	}
	return assetVersion
}

// stampOf hashes every embedded file, name and contents, in a fixed order.
//
// Names are part of the hash so that renaming a file changes the stamp even
// when nothing inside it did — the fonts are referenced from inside the
// stylesheet rather than from a template, so replacing one means renaming it.
func stampOf(files fs.FS) string {
	sum := sha256.New()

	// From the root of what it is given, which is the static directory itself
	// — assetsFS subs into it. Walking a "static" prefix here hashed nothing
	// once that changed, and an empty hash is a constant stamp under a
	// year-long cache: exactly the v0.7.0 failure described above.
	var names []string
	_ = fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
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
	return func(w http.ResponseWriter, r *http.Request) {
		if devDir != "" {
			// Read per request, and never cached: the point of development is
			// that the file on disk is the file you get.
			w.Header().Set("Cache-Control", "no-store")
			http.StripPrefix("/static/", http.FileServer(http.FS(assetsFS()))).ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.StripPrefix("/static/", http.FileServer(http.FS(assetsFS()))).ServeHTTP(w, r)
	}
}
