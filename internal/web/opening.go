package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy says the first thing. The bar is meaningfulness rather than a budget:
// something on today or tomorrow, notes waiting, a chore due. Nothing worth
// saying means nothing said.

// openingLine is what Buddy would open with, or empty when nothing is worth
// saying. openingTurn fingerprints it, which is what stops the record filling
// with the same sentence: it speaks again when what it would say has changed.
func openingLine(loc *time.Location, w squirrel.Waiting, soon []squirrel.Moment, on time.Time) string {
	// Ordered by how little of your choosing it is. A fixed point will happen
	// whether or not you look; a chore came back on its own; the pile is
	// yours, and is last because a pile of undecided things is not urgent by
	// being large.
	switch {
	case len(soon) > 0:
		m := soon[0]
		when := "today"
		if !sameDayIn(loc, m.Starts, on) {
			when = "tomorrow"
		}
		// In the person's clock. The store hands this back converted since
		// 25 August 2026, and this says so again rather than relying on it:
		// the sentence names a time somebody will act on, and there is no
		// cheaper insurance than one call.
		at := m.Starts
		if loc != nil {
			at = at.In(loc)
		}
		return m.Label + " " + when + ", at " + at.Format("15:04") + "."
	case w.Chores > 0:
		return "Something has come back round."
	case w.Pile > 0:
		return "There are things in the pile you have not decided about."
	}
	return ""
}

// sameDayIn is whether two instants fall on the same day where the person is.
// Both are moved into one location first, and that location is given rather than
// assumed: YearDay reads whatever zone the value carries, so a row straight out
// of the database is UTC and answers about a different day.
func sameDayIn(loc *time.Location, a, b time.Time) bool {
	if loc != nil {
		a, b = a.In(loc), b.In(loc)
	}
	return a.YearDay() == b.YearDay() && a.Year() == b.Year()
}

// endsAsking is whether the conversation is waiting for an answer. Narrower than
// endsOpen, which refuses to put anything down while there is anything on the
// table — right for the offer, wrong here: the opening line carries a chip, so it
// ends open by its own hand and would never speak again.
func endsAsking(turns []squirrel.Turn) bool {
	if len(turns) == 0 {
		return false
	}
	last := turns[len(turns)-1]
	if last.Who != squirrel.SpeakerBuddy || len(last.Shown) == 0 {
		return false
	}
	var sh drawn
	if err := json.Unmarshal(last.Shown, &sh); err != nil {
		// A turn that cannot be read is treated as a question. Saying nothing
		// is the safe direction; the other one talks over it.
		return true
	}
	// Faces counts: the check-in is a question with its answers drawn on it. Left out
	// when this was written, so the opening line landed on top of "how do you feel?"
	// and the two alternated down the screen.
	return sh.Faces || sh.Pick != nil || sh.Cal != nil || sh.Say != nil || sh.Cut != nil
}

// openingTurn is that line as a turn, or nothing. The mark travels in the turn's
// own record, so "have I said this" is answered by reading the conversation
// rather than by keeping state beside it.
func openingTurn(ctx context.Context, s Store, opts Options, personID int64, turns []squirrel.Turn) (squirrel.Turn, bool) {
	// Where you got to comes first: if you were part way through the pile forty
	// minutes ago, that is the most useful sentence this screen has.
	//
	// The exit ramp before even that, because it is the only one about something
	// happening now.
	if turn, ok := exitRampTurn(ctx, s, personID); ok {
		return turn, true
	}
	if turn, ok := whereYouGotTo(ctx, s, personID); ok {
		return turn, true
	}
	// Then something you set aside that has gone quiet. After where you got
	// to, because a run in progress is a thing you were doing a minute ago and
	// this is a thing nobody has touched for three weeks.
	if turn, ok := goneQuietTurn(ctx, s, personID, turns); ok {
		return turn, true
	}

	waiting, err := s.Waiting(ctx, personID, now())
	if err != nil {
		// A count that cannot be read is a line not drawn. Nothing here is
		// worth an error page: you came to talk, not to be told the database
		// is unwell.
		slog.Error("reading what is waiting, to open with", "error", err)
		return squirrel.Turn{}, false
	}
	soon, err := s.Upcoming(ctx, personID, now(), 1)
	if err != nil {
		slog.Error("reading what is coming, to open with", "error", err)
		soon = nil
	}
	// Only what is close enough to act on. Upcoming is everything ahead, and
	// an appointment in nine days is not a thing that needs attention now.
	soon = withinADay(soon, now())

	words := openingLine(opts.Location, waiting, soon, now())
	if words == "" {
		return squirrel.Turn{}, false
	}
	mark := fingerprint(words)
	if saidAlready(turns, mark) {
		return squirrel.Turn{}, false
	}

	sh := drawn{Opened: mark}
	// And what it is about, so the line is a way in rather than an
	// announcement. A sentence about the agenda with no way to the agenda is a
	// notification.
	switch {
	case len(soon) > 0:
		sh.Chips = []turnChip{{Label: "the agenda", Href: "/r/at"}}
	case waiting.Chores > 0:
		sh.Chips = []turnChip{{Label: "the chores", Href: "/r/chores"}}
	default:
		sh.Chips = []turnChip{{Label: "the pile", Href: "/r/pile"}}
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing what Buddy opened with", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words, Shown: body}, true
}

// whereYouGotTo offers you back the run you were part way through.
//
// `carry on` and `start fresh` are answers about you rather than about a note, so
// they are chips rather than buttons on a card, and neither is drawn louder.
//
// A run that has aged out is not mentioned at all — see squirrel.KeepingPlace.
func whereYouGotTo(ctx context.Context, s Store, personID int64) (squirrel.Turn, bool) {
	run, found, err := s.RunFor(ctx, personID, now())
	if err != nil {
		// The same failure as any other read here: a line not drawn. You came
		// to talk, not to be told the database is unwell.
		slog.Error("reading where you got to", "error", err)
		return squirrel.Turn{}, false
	}
	if !found {
		return squirrel.Turn{}, false
	}

	sh := drawn{
		// Marked like any other opening, so it is said once rather than every
		// time the page is drawn.
		Opened: "place:" + run.Place,
		Chips: []turnChip{
			{Label: "carry on", Href: "/r/" + run.Place},
			{Label: "start fresh", Action: "/place/fresh"},
		},
	}
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing where you got to", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{
		Who:   squirrel.SpeakerBuddy,
		Words: "You were part way through " + placeCalled(run.Place) + ", " + agoInWords(run.Since) + ".",
		Shown: body,
	}, true
}

// exitRampTurn is the one interruption this product allows itself, and only
// because you asked for it.
//
// It says how long rather than that the timer ran out, and offers a place to stop
// rather than telling you to.
//
// Marked said in the same breath as being drawn. Everything else in this opening
// is idempotent; this one writes, because "once" is the entire safety property.
func exitRampTurn(ctx context.Context, s Store, personID int64) (squirrel.Turn, bool) {
	t, found, err := s.RampDue(ctx, personID, now())
	if err != nil {
		slog.Error("reading the exit ramp", "error", err)
		return squirrel.Turn{}, false
	}
	if !found {
		return squirrel.Turn{}, false
	}
	if err := s.RampSaid(ctx, personID, now()); err != nil {
		// Not said rather than said twice. An interruption that repeats is the
		// version of this that gets switched off after two days.
		slog.Error("marking the exit ramp said", "error", err)
		return squirrel.Turn{}, false
	}

	body, err := json.Marshal(drawn{
		Opened: "ramp:" + t.Started.Format(time.RFC3339),
		Cards: []cardView{{
			Title: "a good place to stop is after this bit",
			Meta:  "you asked me to say something",
			Acts: []actView{
				{Label: "STOPPING", Style: "did", Action: "/timer",
					Fields: map[string]string{"stop": "1", "from": "home"}},
				{Label: "20 more minutes", Action: "/timer",
					Fields: map[string]string{"minutes": "20", "label": t.Label, "from": "home"}},
				{Label: "leave me alone", Style: "stop", Action: "/timer",
					Fields: map[string]string{"hush": "1", "from": "home"}},
			},
		}},
	})
	if err != nil {
		slog.Error("drawing the exit ramp", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{
		Who: squirrel.SpeakerBuddy,
		Words: "You have been on " + t.Label + " for " +
			onItInWords(now().Sub(t.Started)) + ".",
		Shown: body,
	}, true
}

// onItInWords is hours and minutes, never a decimal and never seconds: "2h 40m"
// is read at a glance where "160 minutes" has to be converted.
func onItInWords(d time.Duration) string {
	mins := int(d.Minutes())
	if mins < 60 {
		return strconv.Itoa(mins) + "m"
	}
	return strconv.Itoa(mins/60) + "h " + strconv.Itoa(mins%60) + "m"
}

// goneQuietTurn mentions something you set aside that nobody has touched since.
//
// A mention, not a task. The three answers are unequal in effort and equal in
// standing: `still waiting` costs one press and moves the clock. If saying
// "still" were harder, this would push you to close things.
//
// Marked like every other opening, so it is said once per conversation. Ignored
// entirely it comes back another day.
func goneQuietTurn(ctx context.Context, s Store, personID int64, turns []squirrel.Turn) (squirrel.Turn, bool) {
	held, found, err := s.GoneQuiet(ctx, personID, now())
	if err != nil {
		slog.Error("reading what has gone quiet", "error", err)
		return squirrel.Turn{}, false
	}
	if !found {
		return squirrel.Turn{}, false
	}
	mark := "quiet:" + strconv.FormatInt(held.ID, 10)
	if saidAlready(turns, mark) {
		return squirrel.Turn{}, false
	}

	id := strconv.FormatInt(held.ID, 10)
	sh := drawn{
		Opened: mark,
		Cards: []cardView{{
			Kind:  "held",
			Title: held.Text,
			Photo: parkedPhoto(held),
			// The reason and how long, in the card's own quiet line. Elapsed
			// time on a thing somebody else owes you is a fact, not a score —
			// it is the countdown pointed backwards.
			Meta: held.Words() + " · " + waitedInWords(held.Since),
		}},
		Chips: []turnChip{
			{Label: "still waiting", Action: "/held/act",
				Fields: map[string]string{"id": id, "act": "still"}},
			{Label: "chase it", Action: "/held/act",
				Fields: map[string]string{"id": id, "act": "back"}},
			{Label: "let it go", Action: "/pile/act",
				Fields: map[string]string{"id": id, "act": "drop", "was": string(held.State)}},
		},
	}
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing what has gone quiet", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{
		Who: squirrel.SpeakerBuddy,
		// No question mark, and no "should you". It says the fact and stops.
		Words: "Something you set aside has gone quiet.",
		Shown: body,
	}, true
}

// parkedPhoto is the picture a parked note carries, or empty: a note with only a
// photograph is a good note, and a card that dropped it would show an empty row.
//
// Not `heldPhoto`, which is taken by a helper that only builds under the browser
// tag — so that collision was invisible to `go build`.
func parkedPhoto(h squirrel.HeldItem) string {
	if h.PhotoName == "" {
		return ""
	}
	return "/photo/" + strconv.FormatInt(h.ID, 10)
}

// waitedInWords is weeks and months, never days past a fortnight: "23 days" is a
// measurement and "about three weeks" is a remark.
func waitedInWords(since time.Duration) string {
	days := int(since.Hours() / 24)
	switch {
	case days >= 60:
		return "about " + strconv.Itoa(days/30) + " months"
	case days >= 14:
		return "about " + strconv.Itoa(days/7) + " weeks"
	case days >= 7:
		return "over a week"
	}
	return strconv.Itoa(days) + " days"
}

// placeCalled is the door in the words the menu uses for it.
func placeCalled(place string) string {
	switch place {
	case squirrel.RunPile:
		return "the pile"
	case "chores":
		return "the chores"
	case "tasks":
		return "the tasks"
	}
	return "something"
}

// agoInWords never gives a clock time. "40 minutes ago" is a fact about the gap;
// "you stopped at 14:12" is a record of your afternoon.
func agoInWords(since time.Duration) string {
	switch {
	case since < 2*time.Minute:
		return "a moment ago"
	case since < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(since.Minutes()))
	case since < 2*time.Hour:
		return "about an hour ago"
	}
	return fmt.Sprintf("about %d hours ago", int(since.Hours()))
}

// withinADay is what is close enough to be worth mentioning unprompted.
func withinADay(soon []squirrel.Moment, on time.Time) []squirrel.Moment {
	out := soon[:0]
	for _, m := range soon {
		if m.Starts.After(on) && m.Starts.Before(on.Add(36*time.Hour)) {
			out = append(out, m)
		}
	}
	return out
}

// fingerprint is the line itself, which is the only thing that has to match.
// Two visits that would produce the same sentence are two visits where the
// second has nothing to add.
func fingerprint(words string) string { return strings.TrimSpace(words) }

// saidAlready walks back through the whole conversation on screen, not only the
// last turn: you talk in between, so the opening is rarely the newest thing.
func saidAlready(turns []squirrel.Turn, mark string) bool {
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Who != squirrel.SpeakerBuddy || len(turns[i].Shown) == 0 {
			continue
		}
		var sh drawn
		if err := json.Unmarshal(turns[i].Shown, &sh); err != nil {
			continue
		}
		if sh.Opened == "" {
			continue
		}
		// The most recent opening is the only one that matters. An older one
		// saying the same thing is a day you have already moved past.
		return sh.Opened == mark
	}
	return false
}

// alreadyAsking is whether the conversation already holds an unanswered check-in.
//
// The whole conversation on screen, not only the last turn. What stops it being
// asked forever is that answering writes a reading, which checkinTurn refuses on.
//
// Bounded to the turns in hand: a question older than the page is one you have
// scrolled past, and asking again there is a new question rather than a repeat.
func alreadyAsking(turns []squirrel.Turn) bool {
	for _, t := range turns {
		if t.Who != squirrel.SpeakerBuddy || len(t.Shown) == 0 {
			continue
		}
		var sh drawn
		if err := json.Unmarshal(t.Shown, &sh); err != nil {
			continue
		}
		if sh.Faces {
			return true
		}
	}
	return false
}
