package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy says the first thing.
//
// Every turn in this product was a reply. You arrived, and the conversation
// sat there until you pressed something — which is not what a conversation is,
// and is the reason the thread reads as a menu rendered in bubbles.
//
// The bar is meaningfulness rather than a budget. "Every visit, but only if it
// is meaningful" is the whole rule: something on today or tomorrow, notes
// waiting to be decided about, a chore that is due. Nothing worth saying means
// nothing said, and that is the common case on a quiet afternoon.
//
// It is deliberately not a summary of everything. A line that reads out all
// four numbers is the rail said twice, and the rail is already there — this
// says the one thing that most wants attention, or it says nothing.

// openingLine is what Buddy would open with, and a fingerprint of the facts it
// was built from.
//
// The fingerprint is what stops the record filling with the same sentence.
// Appending on every load is the defect the offer had for an afternoon; the
// offer solved it by refusing to talk over an open turn, which is not enough
// here because this speaks when nothing is open at all. So it speaks when what
// it would say has actually changed.
func openingLine(loc *time.Location, w squirrel.Waiting, soon []squirrel.Moment, on time.Time) (words, mark string) {
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
		words = m.Label + " " + when + ", at " + at.Format("15:04") + "."
	case w.Chores > 0:
		words = "Something has come back round."
	case w.Pile > 0:
		words = "There are things in the pile you have not decided about."
	default:
		return "", ""
	}
	return words, mark
}

// sameDayIn is whether two instants fall on the same day where the person is.
//
// Both are moved into one location first, and that location is given rather
// than assumed. YearDay reads whatever zone the value happens to carry, so
// comparing a row straight out of the database — UTC — against a local clock
// answers about two different days. An appointment at half past midnight is
// "today" for two hours either side of the boundary if you get this wrong,
// which is the one way this line could make somebody leave the house.
func sameDayIn(loc *time.Location, a, b time.Time) bool {
	if loc != nil {
		a, b = a.In(loc), b.In(loc)
	}
	return a.YearDay() == b.YearDay() && a.Year() == b.Year()
}

// endsAsking is whether the conversation is waiting for an answer.
//
// Narrower than endsOpen on purpose. endsOpen refuses to put anything down
// while there is anything at all on the table, which is right for the offer —
// it hands you a job, and two jobs is one too many. The opening line is not a
// job: it says what is true. What it must not do is talk over a question, and
// its own guard against repeating itself is the fingerprint rather than this.
//
// Using endsOpen here would have made the line a one-off: the opening carries
// a chip to the place it is about, so it ends open by its own hand and would
// never speak again.
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
	// Faces counts. The check-in is a question with its answers drawn on it,
	// exactly like the picker, and it was left out when this was written —
	// so the opening line landed on top of "how do you feel?" and the two of
	// them alternated down the screen. Reported from a phone with three
	// unanswered check-ins on it.
	return sh.Faces || sh.Pick != nil || sh.Cal != nil || sh.Say != nil || sh.Cut != nil
}

// openingTurn is that line as a turn, or nothing at all.
//
// The mark travels in the turn's own record, so "have I said this already" is
// answered by reading the conversation rather than by keeping state beside it.
// A conversation that needs a second store to know what it said is two records
// that can disagree.
func openingTurn(ctx context.Context, s Store, opts Options, personID int64, turns []squirrel.Turn) (squirrel.Turn, bool) {
	// Where you got to comes first, before anything about what is waiting.
	//
	// If you were part way through the pile forty minutes ago, that is the most
	// useful sentence this screen has — more useful than the dentist, because
	// the dentist will still be there after you have been told. Everything
	// below is what Buddy opens with when there is no run to come back to.
	if turn, ok := whereYouGotTo(ctx, s, personID); ok {
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

	words, _ := openingLine(opts.Location, waiting, soon, now())
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
		sh.Chips = []turnChip{{Label: "the agenda", Action: "/open",
			Fields: map[string]string{"where": "at"}}}
	case waiting.Chores > 0:
		sh.Chips = []turnChip{{Label: "the chores", Action: "/open",
			Fields: map[string]string{"where": "chores"}}}
	default:
		sh.Chips = []turnChip{{Label: "the pile", Action: "/open",
			Fields: map[string]string{"where": "pile"}}}
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
// Not `placeTurn` — that is taken, by the thing that draws a door. The fourth
// name collision in this package (`.face`, `.say`, `.tcard` were the others),
// and the reason to keep saying so is that each one compiled somewhere before
// it failed somewhere else.
//
// It is not a question about the pile, and the difference matters: `carry on`
// and `start fresh` are both answers about *you* rather than about a note, so
// they are chips rather than buttons on a card. After an interruption either
// one can be the honest answer, which is why neither is drawn louder.
//
// A run that has aged out is not mentioned at all — see squirrel.KeepingPlace.
// The silence is the feature.
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
			{Label: "carry on", Action: "/open", Fields: map[string]string{"where": run.Place}},
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

// agoInWords is how long ago, rounded to something a person would say.
//
// Never a clock time. "40 minutes ago" is a fact about the gap; "you stopped at
// 14:12" is a record of your afternoon, which is the thing this table exists
// not to keep.
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

// saidAlready walks back for an opening with the same mark.
//
// The whole conversation that is on screen, not only the last turn: you talk
// to it in between, so the opening is rarely the newest thing by the time you
// come back. What it must not do is say the same sentence twice in a row with
// three of your own turns in between.
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

// alreadyAsking is whether the conversation already holds a check-in nobody
// has answered.
//
// The whole conversation on screen, not only the last turn: Buddy says other
// things after asking — the opening line, what a door drew — so by the time
// you come back the question is rarely the newest thing. What stops it being
// asked forever is that answering writes a reading, and a fresh reading is
// what checkinTurn already refuses on.
//
// Bounded to the turns in hand, which is the page. A question older than the
// page you are looking at is one you have scrolled past and will not answer,
// and asking again there is a new question rather than a repeat.
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
