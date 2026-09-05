package squirrel

import (
	"context"
	"fmt"
	"time"
)

// The one thing.
//
// The rules are deterministic and fixed in order, so when the picker is wrong you
// can read them and see why. It never changes what is true: a chore is exactly as
// due on a low day.

// OfferKind is which rule produced an offer. It is also the suppression key's
// first half, and the word the screen and the chat both use for it.
type OfferKind string

const (
	// OfferMoment is a fixed point the world imposed, inside the window where
	// leaving matters. See moments.go for the rule it is allowed under.
	OfferMoment OfferKind = "moment"
	// OfferTimer is the thing you are already doing. Nothing is chosen: you
	// chose, and the picker's job is to not talk over you.
	OfferTimer OfferKind = "timer"
	// OfferAgain is the breadcrumb — what you were on before you got up.
	OfferAgain OfferKind = "again"
	OfferChore OfferKind = "chore"
	OfferTask  OfferKind = "task"
)

// Offer is one thing, with the clause that explains it.
//
// Because is mandatory and is checked by a test rather than trusted: an offer
// that cannot say why it is the offer is the thing this file exists not to
// produce.
type Offer struct {
	Kind OfferKind
	// RefID is the chore or item behind it, and zero for an offer that names
	// no row — a running timer is a thing you are doing rather than a row that
	// was picked.
	RefID int64
	// What is offered, in the person's own words wherever there are any: a
	// task is the sentence they typed, a chore is the name they gave it.
	Text string
	// Because is one clause, lower case, no full stop. "you decided this on
	// tuesday", "it is bin day", "you were on this".
	Because string
	// Chore and Item are the row behind the offer when there is one, for
	// callers that need more than the text. Exactly one is set, or neither.
	Chore *Chore
	Item  *Item
}

// Key is the suppression key for this offer, and the value a refusal is
// recorded against.
func (o Offer) Key() string { return suppressionKey(o.Kind, o.RefID) }

// PickNow is the rules in order, with the capacity gate in front.
//
// A fresh wiped or frazzled reading drops rules 4 and 5 — everything Squirrel
// would raise on its own initiative — and keeps 1 through 3, which are the
// world's business and yours.
//
// showAnyway lifts the gate once, without persisting anything.
func (s *Store) PickNow(ctx context.Context, personID int64, now time.Time, showAnyway bool) (Offer, bool, error) {
	capacity := s.Capacity(ctx, personID, now)

	// Rule 1 — a fixed point, inside the window where leaving matters. It is
	// ahead of the running timer because the world's appointment outranks
	// anything you or Squirrel chose, and it is the one rule the capacity gate
	// below never touches.
	if o, found, err := s.pickMoment(ctx, personID, now); err != nil || found {
		return o, found, err
	}

	// Refusals are read once and shared by every rule below. The window is
	// today, locally — "not now" means today, because tomorrow is a fresh
	// question and a refusal that outlived the day would quietly become a
	// second kind of retiring.
	skip, err := s.Suppressed(ctx, personID, s.today(now))
	if err != nil {
		return Offer{}, false, err
	}

	// Rule 2 — what you are already doing. Ahead of everything Squirrel would
	// raise, because a product that suggests a second thing while you are in
	// the middle of the first is the interruption it exists to reduce.
	if t, found, err := s.CurrentTimer(ctx, personID); err != nil {
		return Offer{}, false, err
	} else if found {
		return Offer{
			Kind:    OfferTimer,
			Text:    t.Label,
			Because: "you are on this",
		}, true, nil
	}

	// Rule 3 — the breadcrumb: what you were on before you got up.
	if o, found, err := s.pickAgain(ctx, personID, now); err != nil || found {
		return o, found, err
	}

	// Rules 4 and 5 are Squirrel's own initiative, and the gate stops here.
	if capacity == CapacityLow && !showAnyway {
		return Offer{}, false, nil
	}

	// Rule 4 — a chore that is due and inside the window where raising it is worth
	// doing. Asking.Open is where due and worth-interrupting-for part company, and
	// the nudge makes the same distinction.
	//
	// Not PickChore's weighted draw: randomness is right for a message that arrives
	// unasked and wrong for a screen you opened, where a different answer per reload
	// reads as the product changing its mind.
	due, err := s.DueChores(ctx, personID, now)
	if err != nil {
		return Offer{}, false, err
	}
	for i, c := range due {
		if !c.Ask.Open(now) {
			continue
		}
		if skip[suppressionKey(OfferChore, c.ID)] {
			continue
		}
		return Offer{
			Kind:    OfferChore,
			RefID:   c.ID,
			Text:    c.Name,
			Because: choreBecause(c),
			Chore:   &due[i],
		}, true, nil
	}

	// Rule 5 — the oldest thing you decided to do. Oldest, where every list here is
	// newest-first: a list is read and the newest is what you remember writing; an
	// offer is acted on, and the oldest open task is the one quietly avoided.
	tasks, _, err := s.Tasks(ctx, personID, taskPickDepth)
	if err != nil {
		return Offer{}, false, err
	}
	for i := len(tasks) - 1; i >= 0; i-- {
		if skip[suppressionKey(OfferTask, tasks[i].ID)] {
			continue
		}
		return Offer{
			Kind:    OfferTask,
			RefID:   tasks[i].ID,
			Text:    tasks[i].RawText,
			Because: taskBecause(tasks[i], now),
			Item:    &tasks[i],
		}, true, nil
	}

	// Rule 6 — nothing, which is a normal answer and not an empty state. The
	// caller renders no region at all rather than an encouraging sentence:
	// there is nothing here to be behind on.
	return Offer{}, false, nil
}

// pickAgain is rule 3: what you were on before you got up. Ahead of anything
// Squirrel would raise and behind a running timer, and it survives the capacity
// gate because it is your initiative from an hour ago.
//
// The words never mention finishing: "you were on this" is a fact.
func (s *Store) pickAgain(ctx context.Context, personID int64, now time.Time) (Offer, bool, error) {
	t, found, err := s.LastFocus(ctx, personID, now)
	if err != nil || !found {
		return Offer{}, false, err
	}
	// Turned down, like anything else. This rule sits above the shared refusal set,
	// so "not now" on a breadcrumb once wrote the refusal and handed the same thing
	// straight back — reported as "the button does nothing".
	//
	// Asked as a time rather than through that set: a breadcrumb names a label, so
	// its key is `again:0` however many things you were on today, and suppressing on
	// the key would cost you everything you touched for the rest of the day.
	refused, err := s.RefusedSince(ctx, personID, OfferAgain, t.Ended)
	if err != nil {
		return Offer{}, false, err
	}
	if refused {
		return Offer{}, false, nil
	}
	return Offer{
		Kind:    OfferAgain,
		Text:    t.Label,
		Because: "you were on this " + agoWords(now.Sub(t.Ended)),
	}, true, nil
}

// agoWords says how long ago, softly, and stops at the hour because the
// breadcrumb does. The buckets are the Soft Elapsed Rule's own shape: a number
// attached to something unfinished goes up while nobody is looking.
func agoWords(d time.Duration) string {
	switch {
	case d < 5*time.Minute:
		return "a moment ago"
	case d < 20*time.Minute:
		return "a little while ago"
	default:
		return "earlier"
	}
}

// taskPickDepth is how far back the picker reaches for a task. It reads
// newest-first and walks backwards, so this is "the oldest of your most recent
// thirty decisions" rather than "the oldest thing you ever decided".
const taskPickDepth = 30

// choreBecause is the clause for a chore, and it never says late.
//
// A chore has a rhythm, so what makes it the offer is that its rhythm came
// round, not that time has been passing while you failed to act. "it is bin
// day" is a fact about the week; "three days overdue" is a fact about you.
func choreBecause(c Chore) string {
	if w := c.Ask.Words(); w != "" {
		return "you asked for this " + w
	}
	if !c.EverDone {
		return "you have not started this one yet"
	}
	return "last done " + SinceWords(c.SinceDays)
}

// taskBecause names the moment you decided, because that is the fact that
// makes a task a task. Never how long ago in days — that number goes up while
// nobody is looking, which is the shape this product is without.
func taskBecause(it Item, now time.Time) string {
	// The store hands this back in the person's clock since 25 August 2026, so
	// the conversion here is now a no-op on every path that reads a row. Kept
	// because this also runs on an Item a caller built rather than read — and
	// because it is what "each call site" looked like from the inside: correct,
	// local, and no help at all to the next person to hit the same thing.
	decided := it.ReceivedAt.In(now.Location())
	switch days := int(startOfDay(now).Sub(startOfDay(decided)).Hours() / 24); {
	case days <= 0:
		return "you decided this today"
	case days == 1:
		return "you decided this yesterday"
	case days < 7:
		return "you decided this on " + weekdayWord(decided.Weekday())
	default:
		return "you decided this a while back"
	}
}

func weekdayWord(w time.Weekday) string {
	return map[time.Weekday]string{
		time.Sunday: "sunday", time.Monday: "monday", time.Tuesday: "tuesday",
		time.Wednesday: "wednesday", time.Thursday: "thursday",
		time.Friday: "friday", time.Saturday: "saturday",
	}[w]
}

// startOfDay is local midnight for an instant. Built with time.Date rather
// than by truncating, so a day is the day the calendar says it is across a DST
// boundary rather than a fixed number of hours.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// StartOfDay is the same, for internal/boot: the coach's read tools use the
// picker's own window for today's refusals, and two definitions of "today"
// would put them out of step across a DST boundary.
func StartOfDay(t time.Time) time.Time { return startOfDay(t) }

// StartOfDayIn is the same, where the person is.
//
// The day a refusal belongs to is theirs, not the process's: "not now means
// today, because tomorrow is a fresh question", and on a container running UTC
// that today ended at 02:00 local in summer. See issue #148.
func StartOfDayIn(loc *time.Location, t time.Time) time.Time {
	if loc == nil {
		return startOfDay(t)
	}
	return startOfDay(t.In(loc))
}

func (s *Store) Refuse(ctx context.Context, personID int64, kind OfferKind, refID int64, at time.Time) error {
	return s.RecordAnswer(ctx, personID, kind, refID, AnswerLater, at)
}

func (s *Store) NotThisOne(ctx context.Context, personID int64, kind OfferKind, refID int64, at time.Time) error {
	return s.RecordAnswer(ctx, personID, kind, refID, AnswerWrong, at)
}

// Did records the completion an offer produced, through whichever store call
// the underlying thing already uses, and notes the answer.
//
// One function for both kinds, because the caller — a button on a card — knows
// what it pressed and should not have to know which table that lands in.
func (s *Store) Did(ctx context.Context, personID int64, o Offer, at time.Time) error {
	switch o.Kind {
	case OfferChore:
		if err := s.RecordCompletion(ctx, o.RefID, personID, "offer", at); err != nil {
			return err
		}
	case OfferTask:
		if err := s.SetItemState(ctx, o.RefID, ItemDone, at); err != nil {
			return err
		}
	case OfferMoment:
		// You left, or it is off. Nothing records which, because whether you
		// actually went is not this product's business — the job was to get you
		// out of the door on time, and it is over either way.
		if err := s.MomentDone(ctx, personID, o.RefID, at); err != nil {
			return err
		}
	default:
		// A breadcrumb and a running timer both name a label rather than a
		// row, and Squirrel does not know what a label was. Picking it back up
		// is the only thing either can offer, which is why neither carries a
		// way to mark it done.
		return fmt.Errorf("nothing to complete for a %s offer", o.Kind)
	}
	return s.RecordAnswer(ctx, personID, o.Kind, o.RefID, AnswerDid, at)
}
