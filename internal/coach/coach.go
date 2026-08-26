// Package coach is the one place a model is allowed to speak.
//
// On screen it is called **Buddy**, and the line is worth stating because it
// is a line about authorship rather than about features: **anything a model
// wrote is Buddy's, and anything the rules produced is Squirrel's.** The
// picker's own clause, the ladder's fixed sentences, the nudge that fires
// because a chore is due — those are Squirrel. The clause a model chose, the
// steps it broke a thing into, the wording it gave a nudge, and every word in
// the sheet are Buddy.
//
// The package keeps its name. `internal/coach` is what this *is* — the seam a
// model reaches the product through — and Buddy is what it is *called*.
//
// It is deliberately small and deliberately optional. Everything it does has a
// deterministic answer underneath it — the picker chooses, the ladder answers
// "I can't start", the asking windows decide when to interrupt — and those
// answers are not replaced by this package. They are the floor it stands on.
//
// The zero value is NoCoach, which answers "not available" to everything. That
// is not a degraded mode to apologise for: with no key configured, no network,
// or a month's budget spent, the product works exactly as it did before this
// package existed. Every caller must be written so that is true.
//
// Nothing here imports internal/squirrel, and internal/squirrel does not import
// this. The wiring happens in internal/boot, the same way transports are wired,
// so the core cannot grow a dependency on a model being reachable.
package coach

import (
	"context"
	"errors"
	"time"
)

// ErrUnavailable is what every method returns when there is no coach: no key,
// no network, the budget spent, or the answer failed its shape check.
//
// Callers do not distinguish between those. The question a caller asks is "did
// I get something usable", and the answer is either a reply or this — which is
// why it is one error rather than a taxonomy nobody would branch on.
var ErrUnavailable = errors.New("no coach available")

// And these say which, for the log and for nobody else.
//
// The single sentinel above is right and stays: a caller's question is "did I
// get something usable", and a taxonomy nobody branches on is a taxonomy
// nobody should carry. But the person who runs this does branch on it, at
// three in the morning, and every reason collapsing to one line is how Buddy
// stays broken for a fortnight while its replies quietly get blander.
//
// A model id that has been retired reads exactly like a network blip
// otherwise, and the two want opposite things done about them.
var (
	// ErrProviderRefused is the provider answering, and saying no: a status
	// that is not 200. A retired model, a revoked key, a rate limit.
	ErrProviderRefused = errors.New("the provider refused")
	// ErrProviderUnreachable is the provider not answering at all.
	ErrProviderUnreachable = errors.New("the provider could not be reached")
	// ErrProviderNonsense is the provider answering with something that is not
	// a completion — nearly always a proxy or an auth page in front of it.
	ErrProviderNonsense = errors.New("the provider answered with something else")
)

// Why names the reason for a log line, or "" when the error is not one of
// these. It is deliberately not exported as a type: this is a string for a
// human to read in a log, not a thing to switch on.
func Why(err error) string {
	switch {
	case errors.Is(err, ErrProviderRefused):
		return "refused"
	case errors.Is(err, ErrProviderUnreachable):
		return "unreachable"
	case errors.Is(err, ErrProviderNonsense):
		return "nonsense"
	default:
		return ""
	}
}

// Now is the only context sent on every call, and it is small on purpose.
//
// Capacity is derived rather than raw: the model is told "ok" or "low", never
// "wiped" and never a history. That is the difference between a signal and a
// diagnosis, and it is the line the check-in's own migration draws.
type Now struct {
	// Clock is wall time, "15:04". The model has no clock of its own.
	Clock string
	// PartOfDay is "morning", "afternoon" or "evening".
	PartOfDay string
	// Capacity is "ok" or "low".
	Capacity string
	// FreeUntil is minutes until the next fixed point, or nil when nothing is
	// coming. Nil is not "plenty of time" — it is "nothing was typed".
	FreeUntil *int
	// LandedBadly is the last few answers this person said did not land, in
	// the model's own words, so it can be shown what does not work here rather
	// than told in an instruction nobody can check.
	//
	// Examples, never a count: how often is a fact about the person, and rule
	// 2 forbids one on any surface — including this one, which the person
	// never reads.
	LandedBadly []string
	// Knowing is what a weekly read of the record concluded about how this
	// person works. Shown to the model and never to the person mid-sentence —
	// see knowsYou.
	//
	// Sentences rather than fields, for the reason the table gives: a schema
	// decides in advance what is worth knowing about somebody, and the useful
	// observations are the ones nobody thought to add a column for.
	Knowing []string
}

// Turn is one thing said to the coach.
type Turn struct {
	// PersonID is whose budget this spends. Carried on the turn rather than
	// checked by the caller, so that no call site can forget the ceiling.
	PersonID int64
	// Kind is which surface asked: "chat", "sheet", "overwhelm". It reaches
	// the log and nothing else, and it is what makes "which surface produces
	// the answers that land badly" a question with an answer.
	Kind string
	// Deep asks for the escalation tier. The caller decides, because the
	// caller is the only thing that knows whether this is a routine turn or
	// one where judgement matters. Overwhelmed() is what sets it today.
	Deep bool
	Now  Now
	// Said is what the person typed, verbatim. Never trimmed of meaning.
	Said string
	// Subject is what is on screen, when there is something: the offer's text,
	// the task being started. Empty when the turn is not about anything in
	// particular.
	Subject string
	// Recent is the last few exchanges, oldest first, and it is bounded by the
	// caller rather than by this package. See Window.
	Recent []Exchange
	// CanOpen says the surface that asked can draw one of the places. The
	// screen can — a place is cards in a turn there, which is exactly what the
	// menu draws. Chat cannot: a place there would be the list the guard
	// exists to refuse.
	//
	// The surface states it rather than this package inferring it from Kind,
	// because what a surface can draw is a fact about the surface and nothing
	// here can know it.
	CanOpen bool
}

// Exchange is one round of a conversation, for the rolling window.
type Exchange struct {
	Said    string
	Replied string
	At      time.Time
}

// Window is how much conversation is carried between calls, and how long it
// survives.
//
// Three exchanges and half an hour, both small deliberately. A coach that
// remembers the whole day grows a prompt through the day, which is the cost
// curve this architecture exists to avoid — and a coach that remembers nothing
// cannot hear "no, something else".
const (
	WindowSize = 3
	WindowAge  = 30 * time.Minute
)

// Reply is what came back, after the guard.
type Reply struct {
	// Text is what to show. It has already passed Guard, so a caller may
	// render it without checking anything.
	Text string
	// Model is which model produced it, for the log.
	Model string
	// InTokens and OutTokens are what it cost, for the budget.
	InTokens, OutTokens int
	// Did is what actually changed, in the application's words rather than
	// the model's. A model saying "done" is not evidence anything happened;
	// this is written after the write succeeded.
	Did []string
	// Propose is a thing the coach wants to do and may not do on its own, or
	// nil. The caller renders it as one press and applies it only if pressed.
	Propose *Proposal
	// Open is one of the places, when the coach asked for it to be shown, and
	// empty otherwise. The caller draws it; this package neither knows nor
	// cares what a place looks like.
	//
	// It needs no permission because it changes nothing. Opening the tasks is
	// the same act as pressing the menu, and the worst a wrong one costs is a
	// scroll.
	Open string
}

// Coach is the surface the product asks. Methods are added as the phases that
// need them land; each one answers ErrUnavailable from NoCoach.
type Coach interface {
	// Answer is a conversational turn: the sheet, `!coach`, and the overwhelm
	// turn.
	Answer(ctx context.Context, t Turn) (Reply, error)
	// Decide is what to do now. It reads through tools and hands back one
	// thing it was actually shown; ErrUnavailable means the picker chooses,
	// which is what happened before this method existed and what happens
	// whenever it does not work.
	Decide(ctx context.Context, personID int64) (Decision, error)
	// Smaller breaks one thing into steps. It is the one method that returns
	// a list, and it is safe because nothing renders one: the sequence is
	// stored and handed back a step at a time.
	Smaller(ctx context.Context, personID int64, task, blocker string) ([]string, error)
	// Split proposes the separate things in one note. It returns strings and
	// nothing else — it has no way to write anything, which is decision 8
	// stated as a property rather than an intention.
	Split(ctx context.Context, personID int64, text string) ([]string, error)
	// ShouldInterrupt is a veto on something the rules already allowed, and
	// optional wording for it. It cannot cause an interruption that would not
	// otherwise have happened, because nothing else is ever passed to it — and
	// it fails open, so an absent coach leaves the nudge exactly as it was.
	ShouldInterrupt(ctx context.Context, personID int64, about string, n Now) (string, bool)
	// Reads answers what was typed into the box and says whether it was a
	// thought worth keeping or a question it has just answered.
	//
	// ErrUnavailable means the box behaves exactly as it did before this
	// existed: kept, and "Kept." said back. Every failure lands on the old
	// guarantee rather than on a lost thought.
	Reads(ctx context.Context, personID int64, said string, n Now) (string, bool, error)
	// Learn reads the record of the conversation back and says what it shows
	// about how this person works. Once a week, and everything about it is
	// optional: ErrUnavailable leaves Squirrel knowing whatever it knew, which
	// for a month was nothing at all.
	Learn(ctx context.Context, personID int64, record []string) ([]string, error)
}

// NoCoach is the zero value and the default build. It is not a stub for tests
// — it is the shipping configuration whenever a key is absent, and the reason
// every caller has a deterministic path.
type NoCoach struct{}

func (NoCoach) Answer(context.Context, Turn) (Reply, error) {
	return Reply{}, ErrUnavailable
}

func (NoCoach) Decide(context.Context, int64) (Decision, error) {
	return Decision{}, ErrUnavailable
}

func (NoCoach) Learn(context.Context, int64, []string) ([]string, error) {
	return nil, ErrUnavailable
}

func (NoCoach) Reads(context.Context, int64, string, Now) (string, bool, error) {
	return "", true, ErrUnavailable
}

func (NoCoach) Smaller(context.Context, int64, string, string) ([]string, error) {
	return nil, ErrUnavailable
}

func (NoCoach) Split(context.Context, int64, string) ([]string, error) {
	return nil, ErrUnavailable
}

// NoCoach lets every interruption through, unchanged. Alone among these it
// does not answer "unavailable": there is nothing to be unavailable for, since
// the rules already decided and the nudge worked on its own for months before
// any of this.
func (NoCoach) ShouldInterrupt(context.Context, int64, string, Now) (string, bool) {
	return "", true
}

// Trim keeps the newest exchanges that are still recent enough to be about
// now. It lives here rather than in a caller so both surfaces cannot disagree
// about how long a conversation lasts.
func Trim(recent []Exchange, now time.Time) []Exchange {
	fresh := make([]Exchange, 0, WindowSize)
	for _, e := range recent {
		if now.Sub(e.At) < WindowAge {
			fresh = append(fresh, e)
		}
	}
	if len(fresh) > WindowSize {
		fresh = fresh[len(fresh)-WindowSize:]
	}
	return fresh
}
