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
	if opts.PersonID == 0 {
		return fmt.Errorf("refusing to mount the pile: no owner")
	}
	m.Get(opts.Path, guard(opts, pileHandler(s, opts)))
	m.Post(opts.Path+"/act", guard(opts, actHandler(s, opts)))
	m.Post(opts.Path+"/chore", guard(opts, choreHandler(s, opts)))
	m.Get(opts.Path+"/static/", staticHandler(opts))
	return nil
}

func pileHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
			searchInto(w, r, s, opts, q)
			return
		}
		items, more, err := s.OpenItems(r.Context(), opts.PersonID, pileLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Path: opts.Path, More: more}
		if len(items) == 0 {
			render(w, "empty", v)
			return
		}
		n := toView(items[0])
		v.Note = &n
		render(w, "pile", v)
	}
}

// searchInto is filled in by Task 8; the deck above must render without it.
func searchInto(w http.ResponseWriter, r *http.Request, s Store, opts Options, q string) {}
