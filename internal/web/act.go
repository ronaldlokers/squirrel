package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The form's vocabulary, and the whole of it. A map rather than a switch so
// that an unknown action is a lookup miss instead of a default branch someone
// later fills in with something destructive.
var actions = map[string]squirrel.ItemState{
	"done": squirrel.ItemDone,
	"keep": squirrel.ItemKept,
	"drop": squirrel.ItemDropped,
	"open": squirrel.ItemOpen,
}

// intervalSentinel stands in for a chore name while an interval is parsed.
//
// The literal is copied from apply.go rather than exported from it: it is an
// internal detail of how ParseEvery is reused, and two callers agreeing on a
// word is cheaper than a package's API growing to say it.
const intervalSentinel = "chore-name-placeholder"

// back sends the browser to the pile with a 303. See Other and not 302: the
// method must become GET, so that a reload after triaging re-reads the pile
// instead of re-submitting the transition.
//
// The undo hint travels in the query string rather than a session, because
// this binary has no sessions and the screen is stateless by construction.
func back(w http.ResponseWriter, r *http.Request, opts Options, undo url.Values) {
	target := opts.Path
	if q := strings.TrimSpace(r.FormValue("q")); q != "" {
		undo.Set("q", q)
	}
	// Where you were, if you had skipped. Without this, acting on the third
	// note down quietly returns you to the first.
	if after := strings.TrimSpace(r.FormValue("after")); after != "" {
		undo.Set("after", after)
	}
	if len(undo) > 0 {
		target += "?" + undo.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func actHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		personID, known := opts.person()
		if !known {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		state, ok := actions[r.FormValue("act")]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Writing the state a note already holds is a no-op rather than an
		// error; SetItemState says so itself, and this handler must not add a
		// check that turns a retry into a failure.
		if err := s.SetItemState(r.Context(), it.ID, state, time.Now()); err != nil {
			fail(w, err)
			return
		}
		back(w, r, opts, url.Values{
			"undo":  {strconv.FormatInt(it.ID, 10)},
			"was":   {string(it.State)},
			"state": {string(state)},
		})
	}
}

func choreHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		personID, known := opts.person()
		if !known {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// ParseEvery wants "every <interval> <name>" and returns the name from
		// the same string. Here the name is the note, so a word that is not a
		// unit is appended and only the duration is kept — the same sentinel
		// trick apply.go documents, and for the same reason: without it,
		// "every" alone borrows the next word as its unit and silently creates
		// a chore nobody asked for.
		_, every, ok := squirrel.ParseEvery(strings.TrimSpace(r.FormValue("every")) + " " + intervalSentinel)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, found, err := s.PromoteItem(r.Context(), personID, id, every)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		back(w, r, opts, url.Values{
			"undo":  {strconv.FormatInt(id, 10)},
			"was":   {string(squirrel.ItemOpen)},
			"state": {"chore"},
		})
	}
}
