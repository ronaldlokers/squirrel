package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// choreActHandler is the three things you can do to a chore, all the same shape
// as the pile's. `every` and `act` are separate fields because changing how often
// something comes back is not a transition — the chore is the same chore
// afterwards.
func choreActHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		personID, known := personOf(r)
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

		// The picker's answer: a number and a unit, composed into the same
		// sentence the four fixed chips post. The two lanes cannot disagree,
		// because they produce the same string for the same rhythm.
		if count, unit := r.FormValue("count"), r.FormValue("unit"); count != "" || unit != "" {
			d, ok := composeEvery(count, unit)
			if !ok {
				// Neither was offered. Nothing is done and nothing is said.
				http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
				return
			}
			made, err := s.UpsertChore(r.Context(), personID, c.Name, d, squirrel.DefaultTolerance(d))
			if err != nil {
				fail(w, err)
				return
			}
			said := "every " + count + " " + unit
			// And the day, if one was named. Applied after the interval and
			// never instead of it: SetChoreRhythm writes the equivalent
			// interval too, so everything that renders "how often" keeps
			// working and only the *due* rule changes.
			if day, weeks, ok := dayAnswered(r.FormValue("day"), unit, count); ok {
				if err := s.SetChoreRhythm(r.Context(), personID, made.ID, day, weeks); err != nil {
					fail(w, err)
					return
				}
				said = saidOnADay(day, weeks)
			} else if c.OnADay() {
				// A day was named once and is not any more, so put it back on an interval.
				//
				// `c` and not `made`: UpsertChore's RETURNING does not carry the rhythm columns,
				// so the chore it hands back always looks like an interval one.
				if err := s.SetChoreRhythm(r.Context(), personID, made.ID, 0, 0); err != nil {
					fail(w, err)
					return
				}
			}
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: said},
				{Who: squirrel.SpeakerBuddy, Words: c.Name + " comes back " + said + " now."},
			}), backToTheRoom(r))
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
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: every},
				{Who: squirrel.SpeakerBuddy, Words: c.Name + " comes back " + every + " now."},
			}), backToTheRoom(r))
			return
		}

		act := r.FormValue("act")
		switch act {
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
		// What the two of you said about it, after the write, because a
		// conversation must not claim something happened that did not.
		answerWith(w, r, keepSaid(r.Context(), s, personID, saidAboutAChore(act, c.Name)), backToTheRoom(r))
	}
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

// newChoreHandler makes a chore from nothing, for the case where you already know
// — standing in the kitchen having just descaled the kettle, wanting that to come
// back rather than a note about wanting it to.
//
// The interval and day-part come from chips that already exist, so nothing has to
// be typed in a format.
func newChoreHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
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

		// A number and a unit from the picker, or one of the four strings the
		// screen's form used to post. Both go through a parser that only
		// accepts what was actually offered — see offered() for why parsing
		// the value loosely is looser than it looks.
		every, ok := composeEvery(r.FormValue("count"), r.FormValue("unit"))
		if !ok {
			every, ok = offered(r.FormValue("every"))
		}
		if !ok {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		part, ok := squirrel.ParseDayPart(r.FormValue("part"))
		if !ok {
			part = squirrel.AnyPart
		}

		c, err := s.UpsertChoreAsking(r.Context(), personID, name, every,
			squirrel.DefaultTolerance(every), squirrel.Asking{Part: part})
		if err != nil {
			fail(w, err)
			return
		}
		if day, weeks, ok := dayAnswered(r.FormValue("day"),
			r.FormValue("unit"), r.FormValue("count")); ok {
			if err := s.SetChoreRhythm(r.Context(), personID, c.ID, day, weeks); err != nil {
				fail(w, err)
				return
			}
			c.Weekday, c.Weeks = day, weeks
		}
		// The chore you just made, as a card, so it is on the screen rather
		// than somewhere you have to go and look at.
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: name + " — " + saidRhythm(c.Every)},
			madeAChore(c),
		}), backToTheRoom(r))
	}
}

// choreNameLimit is a guard rather than a rule about how much you may say. A
// chore's name is read in a nudge, and a nudge is one line.
const choreNameLimit = 200

// offered turns one of the four chips into an interval and refuses anything else.
//
// Parsing the value would be looser than it looks: ParseEvery is generous about
// what follows the unit, so "every fortnight or so" becomes a fortnight with "or
// so" as leftovers — a reasonable reading of a sentence and a wrong reading of a
// button that was never offered.
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

// oftenHandler puts the question on the table.
//
// It writes rather than renders, like everything else here: a question that is
// not in the record is a question the record cannot show you answering.
func oftenHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}

		// The id arrives from a form, so it is checked against what this person
		// actually has rather than trusted — the same check the act handler
		// makes, and for the same reason.
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

		count, unit := rhythmOf(c.Every)
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "how often — " + c.Name},
			askHowOften("/chores/act",
				map[string]string{"id": strconv.FormatInt(c.ID, 10)}, count, unit, dayChosen(c)),
		}), backToTheRoom(r))
	}
}

// dayChosen is the day a chore already comes back on, in the picker's own words,
// so the question opens on what is true. The same device rhythmOf uses.
func dayChosen(c squirrel.Chore) string {
	if !c.OnADay() {
		return ""
	}
	return strings.ToLower(c.Weekday.String())[:3]
}

// dayAnswered reads the picker's third row and refuses it unless the rest of the
// answer makes a day mean anything.
//
// Only against one or two weeks. A day is meaningless on "every 3 days" and would
// silently become fortnightly on "every 6 months".
//
// Refusing is not a failure: the interval was already written.
func dayAnswered(said, unit, count string) (time.Weekday, int, bool) {
	if unit != "weeks" {
		return 0, 0, false
	}
	weeks, err := strconv.Atoi(count)
	if err != nil || weeks < 1 || weeks > 2 {
		return 0, 0, false
	}
	day, ok := squirrel.DayNamed(said)
	if !ok {
		return 0, 0, false
	}
	return day, weeks, true
}

// saidOnADay is the rhythm in the words a person would use: "every other
// thursday", not "every 2 weeks on thursday".
func saidOnADay(day time.Weekday, weeks int) string {
	name := strings.ToLower(day.String())
	if weeks == 2 {
		return "every other " + name
	}
	return "every " + name
}
