package web

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A photograph of the thing, instead of typing what it says.
//
// Camera-first, because the moment you want this is the moment you are stood
// in front of a letter holding a phone. The markup asks for the camera
// directly; a browser without one falls back to picking a file, which is the
// desktop case and needs nothing else said about it.
//
// Nowhere to put them is a supported state: with no directory configured the
// camera is never offered, the same way the screen never offers to subscribe
// without a push key. A control that cannot work is worse than one that was
// never drawn.

// Photos is the narrow surface the screen needs, satisfied structurally by
// *squirrel.Photos.
type Photos interface {
	Keep(r io.Reader, contentType string) (string, error)
	Open(name string) (*os.File, error)
}

// photoHandler serves one back.
//
// Behind the same guard as every other page: a photograph of a letter is at
// least as private as the note beside it, and this is the one route that hands
// back bytes nobody can guess the name of.
//
// It answers from the row rather than from the path, so the only names that
// can ever be opened are ones this product wrote down. The content type comes
// from the row too, and the row only ever holds one of the handful the store
// will keep — so nothing a browser once claimed is echoed back as a type.
func photoHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found || it.PhotoName == "" || opts.Photos == nil {
			http.NotFound(w, r)
			return
		}

		f, err := opts.Photos.Open(it.PhotoName)
		if err != nil {
			// The row says there is a photograph and the disk disagrees. That
			// is worth a log and a 404 rather than a 500: the pile still
			// works, and the note is still a note.
			//
			// The log is the half that was missing. On a single node with the
			// photographs on a volume beside the pod, a remount, a restore
			// that did not line up, or a deletion turns every affected
			// photograph into a quiet 404 — and without this line there is
			// nothing afterwards to tell you it happened, or how many.
			slog.Warn("a photograph the row expects is not on disk",
				"item", it.ID, "name", it.PhotoName, "error", err)
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", it.PhotoType)
		// Immutable: the name is random and a name never gets new bytes, so a
		// browser that has it never needs to ask again. Private, because this
		// is behind forward-auth and no shared cache has any business with it.
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		// A photograph is not a document, and nothing here should ever be
		// interpreted as one.
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, it.PhotoName, squirrel.PhotoModTime(f), f)
	}
}
