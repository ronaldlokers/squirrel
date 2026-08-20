// Package coach is the one place a model is allowed to speak.
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
