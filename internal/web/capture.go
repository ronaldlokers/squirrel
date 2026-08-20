package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// captureLimit is longer than any thought anyone types on a phone and short
// enough that a stuck key cannot fill a table. It is a guard rather than a
// rule about how much you may say.
const captureLimit = 4000

// The slot in the lid of the box: you post a thought in without opening it.
//
// This screen refused to capture for its whole life, and the reasoning is
// worth keeping rather than deleting: two capture surfaces means two places to
// look for a thought, which is the problem this product exists to solve. The
// owner overruled it on 20 August 2026, choosing a direct write over a relay
// through Campfire. What makes that survivable is that both surfaces write the
// same row to the same table, so there is one pile with two doors into it.
//
// What it cost was real, and for one release it was not paid: there was no
// spool behind this write. The chat's 👀 means the words reached disk before
// anything else could go wrong, and here there was no such stage — so a live
// network and an unhealthy database was a note that was never taken. The
// screen said so loudly and gave the words back, which is honest and is not
// the same as durable, because a page is one reload from empty.
//
// It goes through the same spool the room's captures do now. Written, fsynced
// and renamed before anything says it was kept; the drain moves it on, and the
// drain has always known how to wait for a database. One durability mechanism
// for both doors rather than two that have to be kept in step.
//
// What that costs, stated: a note is in the pile a moment later rather than
// instantly — the drain runs every second by default. The slot is on home and
// the pile is a different screen, so the gap is invisible in practice; and the
// room has always worked this way.
//
// The Campfire room still stops being the complete record. That part of the
// original bargain stands.
func captureHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Still checked, and still refuses: the owner not being known yet means
		// the drain cannot resolve this capture to anybody either, so
		// accepting it would spool a note nobody owns.
		if _, ok := opts.person(); !ok {
			// The same 503 the rest of the screen gives when nobody knows
			// whose pile this is: a redirect here would look like the words
			// went somewhere.
			fail(w, errNoOwner)
			return
		}
		// A photograph arrives as multipart; words alone do not. Parsing the
		// multipart form handles both, and the memory bound is what keeps a
		// large upload from being held in RAM before the size check below sees
		// it.
		if err := r.ParseMultipartForm(1 << 20); err != nil && err != http.ErrNotMultipart {
			backToHome(w, r, "", true)
			return
		}
		if r.MultipartForm != nil {
			defer func() { _ = r.MultipartForm.RemoveAll() }()
		}
		// Verbatim, like every other capture in this system: never trimmed of
		// its meaning, only of the whitespace a keyboard adds at the ends.
		text := strings.TrimSpace(r.FormValue("text"))

		// The photograph goes to disk before the capture that references it,
		// and is fsynced there. So a spool entry never points at a file that
		// is not on the volume; the other order would give a note that renders
		// a broken picture, which is a worse thing to find than a missing one.
		//
		// An entry that fails to write after the bytes landed leaves a file
		// nothing references — litter on a volume, not a lost thought.
		photo, kind, err := keepPhoto(r, opts)
		if err != nil {
			backToHome(w, r, text, true)
			return
		}

		// Nothing said and nothing photographed is nothing to keep. A
		// photograph on its own is a capture — that is most of the point of
		// having one — so it is only the pair being empty that does nothing.
		if text == "" && photo == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}

		// Whose it is, said in the transport's own vocabulary rather than as a
		// person id: the drain resolves every capture's owner from its sender,
		// and this one is no different for being typed on the screen. boot
		// seeds the matching identity.
		sender := opts.Identity

		if _, err := opts.Spool.Write(squirrel.Capture{
			Transport:  squirrel.ScreenTransport,
			SenderID:   &sender,
			Text:       text,
			Payload:    []byte(squirrel.ScreenCapture),
			ReceivedAt: time.Now(),
			PhotoName:  photo,
			PhotoType:  kind,
		}); err != nil {
			// The words go back to the page rather than into a log. A capture
			// box that clears on failure is a capture box that eats thoughts.
			//
			// This is now the only way a capture can fail, and it means the
			// disk is unwritable — which is a different and much louder
			// problem than a database being briefly unreachable.
			backToHome(w, r, text, true)
			return
		}
		http.Redirect(w, r, "/?kept=1", http.StatusSeeOther)
	}
}

// backToHome returns to the slot, carrying the words when there are any to
// carry. 303 rather than 302: the method has to become GET so a reload does
// not repost.
func backToHome(w http.ResponseWriter, r *http.Request, text string, failed bool) {
	q := url.Values{}
	if failed {
		q.Set("nokeep", "1")
	}
	if text != "" {
		q.Set("said", text)
	}
	http.Redirect(w, r, "/?"+q.Encode(), http.StatusSeeOther)
}

// keepPhoto stores the photograph on the request, if there is one.
//
// No photograph is the ordinary case and is not an error. A photograph the
// store will not keep — the wrong kind, too big, empty — is an error, and the
// words come back with it rather than the capture quietly losing the picture
// and saying it was kept.
func keepPhoto(r *http.Request, opts Options) (name, kind string, err error) {
	if opts.Photos == nil || r.MultipartForm == nil {
		return "", "", nil
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		// No file part at all: words only, which is most captures.
		return "", "", nil
	}
	defer file.Close()

	// What the browser said it is, checked against the handful this keeps.
	// The type stored is the one from that list rather than the browser's own
	// string, so nothing it claimed is ever handed back as a content type.
	declared := header.Header.Get("Content-Type")
	if _, ok := squirrel.KnownKind(declared); !ok {
		return "", "", fmt.Errorf("not a photograph this keeps: %q", declared)
	}

	name, err = opts.Photos.Keep(file, declared)
	if err != nil {
		return "", "", err
	}
	return name, squirrel.PhotoKind(declared), nil
}
