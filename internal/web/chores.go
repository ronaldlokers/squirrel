package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The chores screen exists because a chore was invisible: it only ever
// appeared when it nudged you, and the nudge is the one moment you are least
// able to say "actually, not this one, ever". Seeing them is what makes
// retiring a real option rather than a command you have to remember.
//
// It shows what a chore is and when it was last done. It does not show how
// many there are, how many are due, or how late anything is — the rule the
// pile lives by does not stop at the pile.
func choresHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		chores, err := s.ActiveChores(r.Context(), personID)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Path: opts.Path}
		for _, c := range chores {
			v.Chores = append(v.Chores, toChoreView(c))
		}
		render(w, "chores", v)
	}
}

// choreActHandler is the three things you can do to a chore, and they are all
// the same shape as the pile's: a form POST answered with a 303.
//
// `every` and `act` are separate fields rather than one vocabulary, because
// changing how often something comes back is not a transition — the chore is
// the same chore afterwards, which is exactly why the interval chips can post
// straight to it.
func choreActHandler(s Store, opts Options) http.HandlerFunc {
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

		// The id arrives from a form, so it is checked against what this person
		// actually has rather than trusted. ActiveChores is already scoped to
		// the person and there is one of them, so this costs a query nobody
		// notices and closes the hole a bare id would open.
		chores, err := s.ActiveChores(r.Context(), personID)
		if err != nil {
			fail(w, err)
			return
		}
		var c squirrel.Chore
		for _, candidate := range chores {
			if candidate.ID == id {
				c = candidate
				break
			}
		}
		if c.ID == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if every := strings.TrimSpace(r.FormValue("every")); every != "" {
			_, d, ok := squirrel.ParseEvery(every + " " + intervalSentinel)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			// Upsert by name is how the chat command changes an interval too,
			// and the unique index makes it the same row rather than a second
			// chore that says nearly the same thing.
			if _, err := s.UpsertChore(r.Context(), personID, c.Name, d, squirrel.DefaultTolerance(d)); err != nil {
				fail(w, err)
				return
			}
			backToChores(w, r, opts)
			return
		}

		switch r.FormValue("act") {
		case "done":
			// The same write a tap on a nudge makes, and the source says which
			// surface said so — the events table is the only place that
			// distinction survives.
			if err := s.RecordCompletion(r.Context(), c.ID, personID, "screen", time.Now()); err != nil {
				fail(w, err)
				return
			}
		case "retire":
			if err := s.DeactivateChore(r.Context(), c.ID); err != nil {
				fail(w, err)
				return
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		backToChores(w, r, opts)
	}
}

func backToChores(w http.ResponseWriter, r *http.Request, opts Options) {
	http.Redirect(w, r, opts.Path+"/chores", http.StatusSeeOther)
}

func toChoreView(c squirrel.Chore) choreView {
	return choreView{
		ID:    c.ID,
		Name:  c.Name,
		Every: "every " + plural(c.EveryDays, "day"),
		Last:  lastDone(c.SinceDays),
	}
}

// lastDone says when, never how late. "3 days ago" is a fact about a chore;
// "2 days overdue" is a fact about you, and this product does not make those.
func lastDone(sinceDays int) string {
	switch {
	case sinceDays <= 0:
		return "today"
	case sinceDays == 1:
		return "yesterday"
	default:
		return plural(sinceDays, "day") + " ago"
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return strconv.Itoa(n) + " " + unit
	}
	return strconv.Itoa(n) + " " + unit + "s"
}
