package squirrel

import (
	"context"
	"fmt"
	"time"
)

// The one thing.
//
// This is the file that changes what Squirrel is. Everything before it stored
// well and chose nothing: the pile is arrival order, the tasks are arrival
// order, the chores are alphabetical, and the only thing in the product that
// ever picked one thing out was the nudge — one chore, once a day, in chat.
// So the person who opened Squirrel while overwhelmed was asked which of three
// boxes they would like to open, which is three decisions charged to the
// resource that is already short.
//
// The rules below are deterministic and fixed in order. That is the whole
// design and not an implementation detail: when the picker is wrong you can
// read six rules and see why it was wrong, which is a thing you cannot do with
// anything that scores or learns or generates. It also means it works when
// everything else is down, and that it can be explained in one clause — which
// is a requirement, not a nicety. An offer nobody can account for is a
// demand.
//
// It never changes what is true. A chore is exactly as due on a low day, and
// nothing here hides anything from someone who goes looking; every screen the
// product had before this still shows everything it did.

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

// PickNow is the rules, in order, with the capacity gate in front of them.
//
// The gate first: a fresh wiped or frazzled reading drops rules 4 and 5 —
// everything Squirrel would have raised on its own initiative — and keeps 1
// through 3, which are the world's business and yours. A low day offers what
// is fixed and what you were already doing, and nothing else.
//
// `showAnyway` is how that stops being a wall. Someone who says they are wiped
// and then wants to work must not find the product has decided for them, so
// the screen carries a quiet "show me anyway" that calls this with the gate
// lifted, once, without persisting anything.
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
	skip, err := s.Suppressed(ctx, personID, startOfDay(now))
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

	// Rule 4 — a chore that is due, and inside the window where raising it is
	// worth doing. Being due and being worth interrupting for are two
	// questions, and Asking.Open is where they part company; the nudge already
	// makes exactly this distinction and this must agree with it.
	//
	// Not PickChore's weighted draw. That randomness is right for a message
	// that arrives unasked — it is what stops the same overdue thing being
	// named every morning — and wrong for a screen you opened on purpose,
	// where a different answer on every reload reads as the product changing
	// its mind. So: the most overdue of the ones that are worth raising, which
	// DueChores already orders first.
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

	// Rule 5 — the oldest thing you decided to do.
	//
	// Oldest, where every list in this product is newest-first, and the
	// difference is deliberate. A list is read, and the newest is the one you
	// still remember writing; an offer is acted on, and the oldest open task is
	// the one that has been quietly avoided. That is the one worth handing you
	// with a way to start it.
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

// pickAgain is rule 3: what you were on before you got up.
//
// Ahead of anything Squirrel would raise and behind a running timer, which is
// the order the person's own attention is already in. It survives the capacity
// gate for the same reason the timer does — picking something back up is not
// Squirrel's initiative, it is yours from an hour ago.
//
// The words never mention finishing. "You were on this" is a fact; "you did
// not finish this" is a sentence about the person, and the difference is the
// whole reason this is safe to show.
func (s *Store) pickAgain(ctx context.Context, personID int64, now time.Time) (Offer, bool, error) {
	t, found, err := s.LastFocus(ctx, personID, now)
	if err != nil || !found {
		return Offer{}, false, err
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

// taskPickDepth is how far back the picker will reach for a task.
//
// A cap rather than a scan, and the number matters less than that there is
// one: it reads newest-first and walks backwards, so this is "the oldest of
// your most recent thirty decisions" rather than "the oldest thing you ever
// decided". Something decided four months ago and never done is not a thing to
// hand someone this morning; it is a thing to find in the list and un-decide.
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

// Decider is the seam a model reaches this decision through, or nil.
//
// A func of primitives, like the coach's other seam and for the same reason:
// this package must not import internal/coach. It takes the picker's own
// answer and hands back one to render instead, or says it has nothing.
//
// mayAsk is whether this caller is allowed to spend a call. False means "use
// an answer you already have, or say you have nothing" — which is what makes
// a surface that must be free to open able to show the same thing the paying
// surface shows, without ever becoming a reason to pay.
//
// Every caller must be written so that nil, and false, and a model that never
// answers are all ordinary. PickNow is the whole answer whenever this is not
// available, which is most of the time by design.
type Decider func(ctx context.Context, personID int64, pickedKind string, pickedRef int64,
	mayAsk bool) (kind string, refID int64, text, because string, ok bool)

// JudgementHelps reports whether a model is allowed near this offer.
//
// Only three of the five kinds. A running timer is a thing you are already
// doing rather than a thing that was picked, and a fixed point is the one
// thing here the world imposed rather than the product suggested — a model
// second-guessing either would be overruling a rule with an opinion.
//
// It lives in the core rather than in the wiring so both surfaces ask the same
// question, and so the rule is next to the rules it protects.
func JudgementHelps(kind OfferKind) bool {
	switch kind {
	case OfferChore, OfferTask, OfferAgain:
		return true
	}
	return false
}

// StartOfDay is the same, for internal/boot: the coach's read tools use the
// picker's own window for today's refusals, and two definitions of "today"
// would put them out of step across a DST boundary.
func StartOfDay(t time.Time) time.Time { return startOfDay(t) }

// Refuse records a "not now" and is the only thing that suppresses an offer.
//
// It costs one press, it changes nothing about the thing itself, and it is
// never followed by a question. A chore refused here is exactly as due as it
// was — this is not a snooze and does not touch the chore's clock, because the
// picker's memory and the nudge's are two different budgets and merging them
// would let one press silence the other surface.
func (s *Store) Refuse(ctx context.Context, personID int64, kind OfferKind, refID int64, at time.Time) error {
	return s.RecordAnswer(ctx, personID, kind, refID, AnswerLater, at)
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
