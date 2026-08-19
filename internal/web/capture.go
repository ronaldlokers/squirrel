package web

import (
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
// What it costs is real and is not hidden: the Campfire room stops being the
// complete record, and there is no spool behind this write. The chat's 👀 means
// the words reached disk before anything else could go wrong; here there is no
// such stage, so an unreachable database is a note that was never taken. The
// answer is to say so loudly and give the words back, which is what the failure
// path below does — never a redirect that looks like success.
func captureHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			// The same 503 the rest of the screen gives when nobody knows
			// whose pile this is: a redirect here would look like the words
			// went somewhere.
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			backToHome(w, r, "", true)
			return
		}
		// Verbatim, like every other capture in this system: never trimmed of
		// its meaning, only of the whitespace a keyboard adds at the ends.
		text := strings.TrimSpace(r.FormValue("text"))
		if text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}

		_, err := s.InsertItem(r.Context(), squirrel.Item{
			Transport:  "screen",
			PersonID:   &personID,
			RawText:    text,
			Payload:    []byte(squirrel.ScreenCapture),
			ReceivedAt: time.Now(),
		})
		if err != nil {
			// The words go back to the page rather than into a log. A capture
			// box that clears on failure is a capture box that eats thoughts.
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
