package web

import (
	"fmt"
	"net/http"
	"strings"
)

// The deck shows one card. The second row is never rendered; it is read only so
// that "is there more" can be answered without a count, which is the same
// device OpenItems uses and for the same reason.
const pileLimit = 1

// Mux is the routing surface the screen needs from the shared server.
type Mux interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
}

// Mount registers the screen, or refuses. An empty identity is not a
// misconfiguration to warn about and continue past: the pile is every thought
// you have ever had at this bot.
func Mount(m Mux, s Store, opts Options) error {
	if opts.Identity == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_IDENTITY is empty")
	}
	if opts.Owner == nil {
		return fmt.Errorf("refusing to mount the pile: no owner")
	}
	// `{$}` and not `/`: a bare "/" is Go's catch-all, and the home screen would
	// then answer for every URL nobody else claimed — including the typos, which
	// would arrive looking like a working page.
	m.Get("/{$}", guard(opts, homeHandler(s, opts)))
	m.Get("/pile", guard(opts, pileHandler(s, opts)))
	// The slot. Behind the origin check like every other write here: the
	// identity says who is asking, sameOrigin says which page asked.
	m.Post("/capture", guard(opts, sameOrigin(captureHandler(s, opts))))
	m.Post("/mood", guard(opts, sameOrigin(moodHandler(s, opts))))
	// Both writes carry the origin check as well as the identity one: the
	// identity says who is asking, sameOrigin says which page asked.
	m.Post("/pile/act", guard(opts, sameOrigin(actHandler(s, opts))))
	m.Post("/pile/chore", guard(opts, sameOrigin(choreHandler(s, opts))))
	m.Post("/pile/fix", guard(opts, sameOrigin(fixHandler(s, opts))))
	m.Get("/chores", guard(opts, choresHandler(s, opts)))
	m.Get("/kept", guard(opts, keptHandler(s, opts)))
	m.Post("/chores/act", guard(opts, sameOrigin(choreActHandler(s, opts))))
	// The chores screen lived here for its whole life. A bookmark that dies
	// quietly is worse than a redirect nobody notices.
	m.Get("/pile/chores", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/chores", http.StatusMovedPermanently)
	}))
	// Outside the guard, like the worker below and for the same reason: a
	// browser fetches a manifest without the cookies that carry the identity,
	// and one that answers 403 leaves an installed app with no icon and no
	// explanation. It names the app and lists four PNGs — there is nothing in
	// it to protect.
	m.Get("/manifest.webmanifest", manifestHandler())
	// Not behind the guard: a browser fetches the worker without the cookies
	// that carry the identity, and a worker that 302s to a login page is a
	// worker that never installs. It contains no notes — only which files to
	// keep and what to say when the network is gone.
	m.Get("/sw.js", swHandler())
	m.Get("/static/", staticHandler())
	return nil
}

func pileHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		undo := undoFrom(r.URL.Query())
		if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
			searchInto(w, r, s, opts, personID, q, undo)
			return
		}
		// The cursor is skipping, and it lives here rather than in the store's
		// idea of the pile: a note skipped past is untouched, and reloading
		// without the parameter is how you get back to the top.
		after := cursorFrom(r.URL.Query())
		items, more, err := s.OpenItemsAfter(r.Context(), personID, after, pileLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{More: more, Undo: undo, After: after}
		if len(items) == 0 {
			// Nothing older is not an empty pile. Everything skipped past is
			// still open, and a page that said "nothing in the pile" here
			// would be describing the cursor rather than the pile.
			if after != 0 {
				render(w, "bottom", v)
				return
			}
			render(w, "empty", v)
			return
		}
		n := toView(items[0])
		v.Note = &n
		render(w, "pile", v)
	}
}

// searchLimit caps the result list. The cap is what makes "there is more"
// truthful; without it the line would appear over a complete list, which is a
// false claim in the one place the counting rule is most likely to leak.
const searchLimit = 6

// choreHits caps the chores a search answers with. Short on purpose: a chore
// list is short, and this is a way to reach one rather than a second list to
// read.
const choreHits = 3

func searchInto(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID int64, q string, undo *undoView) {
	items, more, err := s.SearchItems(r.Context(), personID, q, searchLimit)
	if err != nil {
		fail(w, err)
		return
	}
	// One search, both kinds of thing. The lid carries one field on every
	// screen, and a person typing a word has not first decided whether the word
	// belongs to a note or to a chore.
	chores, err := s.SearchChores(r.Context(), personID, q, choreHits)
	if err != nil {
		fail(w, err)
		return
	}
	v := view{Query: q, More: more, Undo: undo}
	for _, c := range chores {
		v.Chores = append(v.Chores, toChoreView(c))
	}
	for _, it := range items {
		v.Results = append(v.Results, toView(it))
	}
	render(w, "results", v)
}
