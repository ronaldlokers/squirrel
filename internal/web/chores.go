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
		v := view{Here: "chores"}
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
			// The four this screen offers, like everywhere else it asks.
			d, ok := offered(every)
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
	http.Redirect(w, r, "/chores", http.StatusSeeOther)
}

func toChoreView(c squirrel.Chore) choreView {
	v := choreView{
		ID:    c.ID,
		Name:  c.Name,
		Every: cadence(c.EveryDays),
		Chip:  chipFor(c.EveryDays),
	}
	// What has not happened is not reported. A chore nobody has ever done has
	// a baseline anyway — its own birthday — and printing that as "last done"
	// would be a sentence about the person rather than about the chore.
	if c.EverDone {
		v.Last = lastDone(c.SinceDays)
	}
	// Only when there is one to say. A chore with no preference says nothing
	// rather than "any time", which would be a fact about the absence of a
	// setting rather than about the chore.
	if w := c.Ask.Words(); w != "" {
		v.When = strings.ToUpper(w)
	}
	return v
}

// cadence is the core's own words, upper-cased for the meta role. The casing
// is style; the vocabulary is not this package's to choose, because the chat
// says the same thing about the same chore and the two must agree.
func cadence(days int) string {
	return strings.ToUpper(squirrel.Cadence(days))
}

// chipFor is which of the four offered intervals this chore is currently set
// to, or nothing when it was defined from chat with an interval the picker
// does not offer — in which case no chip is marked and pressing one is a
// change like any other, rather than a lie about where things stand.
func chipFor(days int) string {
	switch days {
	case 1:
		return "every day"
	case 7:
		return "every week"
	case 14:
		return "every 2 weeks"
	case 30:
		return "every month"
	default:
		return ""
	}
}

// lastDone says roughly when, never how long. The buckets live in the core
// beside the rest of a chore's vocabulary; see squirrel.SinceWords for why
// there is no number here.
func lastDone(sinceDays int) string {
	return squirrel.SinceWords(sinceDays)
}

// newChoreHandler makes a chore from nothing.
//
// Until now a chore could only be made from a note, and the reasoning was
// sound: a chore usually starts life as a thought you had, and making one
// from a note keeps the two connected. What it could not do is the case where
// you already know — you are standing in the kitchen having just descaled the
// kettle, and the thing you want is for that to come back, not a note about
// wanting it to.
//
// The interval and the day-part are chosen from the chips that already exist,
// so nothing new was invented and nothing has to be typed in a format.
func newChoreHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/chores", http.StatusSeeOther)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			// Nothing to make. Silence rather than a scolding: an empty form
			// submitted by accident is not a mistake worth a sentence.
			http.Redirect(w, r, "/chores", http.StatusSeeOther)
			return
		}
		if len(name) > choreNameLimit {
			name = name[:choreNameLimit]
		}

		every, ok := offered(r.FormValue("every"))
		if !ok {
			http.Redirect(w, r, "/chores", http.StatusSeeOther)
			return
		}
		part, ok := squirrel.ParseDayPart(r.FormValue("part"))
		if !ok {
			part = squirrel.AnyPart
		}

		if _, err := s.UpsertChoreAsking(r.Context(), personID, name, every,
			squirrel.DefaultTolerance(every), squirrel.Asking{Part: part}); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/chores", http.StatusSeeOther)
	}
}

// choreNameLimit is a guard rather than a rule about how much you may say. A
// chore's name is read in a nudge, and a nudge is one line.
const choreNameLimit = 200

// offered turns one of the four chips into an interval, and refuses anything
// else.
//
// Parsing the value instead would be looser than it looks: ParseEvery is
// deliberately generous about what follows the unit, because in a chat room
// what follows is the chore's name. Fed a form value it makes "every fortnight
// or so" into a fortnight with "or so" as leftovers — which is a reasonable
// reading of a sentence and a wrong reading of a button that was never
// offered. The screen offers four things; these are the four.
func offered(every string) (time.Duration, bool) {
	switch strings.TrimSpace(every) {
	case "every day":
		return 24 * time.Hour, true
	case "every week":
		return 7 * 24 * time.Hour, true
	case "every 2 weeks":
		return 14 * 24 * time.Hour, true
	case "every month":
		return 30 * 24 * time.Hour, true
	}
	return 0, false
}
