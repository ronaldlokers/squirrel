package coach

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// The monthly ceiling.
//
// What this protects against, precisely: Squirrel's own bugs. A retry loop, a
// context that grows without anyone noticing, a phase that turns out to call
// the model far more often than the estimate said. It cannot protect against a
// stolen key — nothing running inside this process can — which is why the hard
// spend limit on the provider's own project is the control that matters, and
// why the secret's comment says so.
//
// Crossing the ceiling is not an error state. It degrades to the deterministic
// floor for the rest of the month: the picker chooses, the ladder answers, and
// everything keeps working. That is what makes a ceiling safe to set low.

// Answer is one recorded call, for the log and the budget.
//
// The whole exchange is kept, indefinitely, by decision: it is what makes a bad
// answer inspectable afterwards and what makes changing the model in
// configuration evaluable at all. It is also a permanent record of bad moments
// and what a machine said about them, kept on the same reasoning that keeps the
// check-in history.
type Answer struct {
	Kind      string
	Model     string
	Prompt    string
	Reply     string
	InTokens  int
	OutTokens int
	// CostMicros is what it cost in micro-euros, priced at record time. Stored
	// rather than derived so that a later price change does not silently
	// rewrite what last month cost.
	CostMicros int64
	// Used says whether the reply reached a human. A guard rejection is
	// recorded and paid for; it just never arrived.
	Used bool
	At   time.Time
}

// Log is the narrow surface the budget consumes. Declared here rather than
// imported: Go satisfies interfaces structurally, so *squirrel.Store fits this
// without either package importing the other, the same way internal/web's
// Store and transport.Sink already work.
type Log interface {
	RecordCoachAnswer(ctx context.Context, personID int64, a Answer) error
	CoachSpentSince(ctx context.Context, personID int64, since time.Time) (int64, error)
}

// Budget answers one question: is there room left this month.
type Budget struct {
	Log Log
	// CeilingMicros is the monthly ceiling in micro-euros. Zero means no
	// ceiling, which is a supported choice — the provider's own spend limit is
	// the real backstop and someone may reasonably prefer only that.
	CeilingMicros int64
}

// MonthStart is the first instant of now's calendar month, locally. The budget
// is a calendar month rather than a rolling thirty days because that is what a
// provider's invoice is, and two windows that nearly agree are worse than one.
func MonthStart(now time.Time) time.Time {
	y, m, _ := now.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, now.Location())
}

// Allows reports whether a call may be made, and what has been spent so far.
//
// It fails **closed**: if the spend cannot be read, no call is made. That is
// the opposite of how everything else in this product handles a database it
// cannot reach, and the exception is deliberate — everywhere else the failure
// costs a feature, and here it could cost money without limit. The coach needs
// the database for its context anyway, so a read that fails would have
// produced a poor answer regardless.
func (b Budget) Allows(ctx context.Context, personID int64, now time.Time) (bool, int64) {
	if b.Log == nil {
		return false, 0
	}
	spent, err := b.Log.CoachSpentSince(ctx, personID, MonthStart(now))
	if err != nil {
		return false, 0
	}
	if b.CeilingMicros <= 0 {
		return true, spent
	}
	return spent < b.CeilingMicros, spent
}

// Permit is proof that the month's ceiling was checked before a paid call was
// made. It carries nothing and does nothing; its whole job is to be impossible
// to produce without asking.
//
// Rule 10 says the deterministic answer is never deleted and becomes the floor:
// no key, no network, or a month's budget spent must leave a product that works
// exactly as it did before the model existed. That was true, and it was true
// because six methods each remembered an identical four-line check — correct in
// all six, and enforced in none. The seventh was one plausible feature away,
// and the roadmap has two of those planned.
//
// So the check is not remembered any more. `completionWithTools` will not
// compile without one of these, which means the floor is a property of the
// type system rather than of anyone's memory.
type Permit struct {
	// held says this permit is holding the gate, so releasing it means
	// something. A permit issued after the wait expired holds nothing and
	// releasing it must not free somebody else's turn.
	held bool
}

// Release gives the gate back. Safe to call twice and safe to call on a permit
// that never held it, because the call sites use `defer` and a rule that only
// works when six people remember it is the rule this file exists to stop
// needing.
func (p *Permit) Release() {
	if p == nil || !p.held {
		return
	}
	p.held = false
	select {
	case <-spending:
	default:
	}
}

// Ask answers with a Permit, or with ErrUnavailable and the reason already
// logged. `instead` names what takes over when it does — the picker, the
// ladder, the rules — because crossing the ceiling is the system working and
// the log should read like it.
// One paid call at a time.
//
// The ceiling is checked before a call and the spend is recorded after it, so
// two requests arriving together could both read "under ceiling" and both
// spend — a webhook redelivery landing while the sheet is mid-answer is the
// realistic version. The docstring at the top of this file names a retry loop
// as the thing it guards against, and a plain check-then-act does not guard
// against that at all.
//
// A channel rather than a mutex, because this one has to be able to give up.
// The alternative designs each put a second obligation on six call sites —
// reserve here, settle there — and a reservation that is never settled hangs
// every future call, which is worse than the overshoot it prevents. This
// cannot hang: if the holder takes longer than any real call could, the next
// caller says so and goes anyway. The bad case degrades to the behaviour that
// exists today rather than to a product that has stopped.
//
// Single replica, one person. Two pods would need this in the database, and
// the day there are two pods this comment is where to start.
var spending = make(chan struct{}, 1)

// spendWait is longer than a model call and shorter than a person's patience.
// A wait this long means something is genuinely wrong, and the log says so.
// A var rather than a const so a test can make it short. The escape hatch is
// the part of this design that stops a forgotten release becoming a hang, and
// a ninety-second escape hatch is one no test would ever wait for — which
// would leave the safety net itself unproven.
var spendWait = 90 * time.Second

func (b Budget) Ask(ctx context.Context, personID int64, now time.Time, instead string) (Permit, error) {
	held := false
	select {
	case spending <- struct{}{}:
		held = true
	case <-ctx.Done():
		return Permit{}, ctx.Err()
	case <-time.After(spendWait):
		// Going anyway. The ceiling is still checked below — what is given up
		// is only the guarantee that nobody checks it at the same moment, and
		// the cost of that is one call's overshoot rather than a hang.
		slog.Warn("a coach call held the budget gate too long; checking the ceiling without it",
			"waited", spendWait)
	}

	if ok, spent := b.Allows(ctx, personID, now); !ok {
		if held {
			<-spending
		}
		// Info, not error. Crossing the ceiling is the system working: the
		// deterministic answers take over for the rest of the month and
		// nothing about the product stops.
		slog.Info("the coach is over its budget for the month; "+instead,
			"spent_micros", spent, "ceiling_micros", b.CeilingMicros)
		return Permit{}, ErrUnavailable
	}
	return Permit{held: held}, nil
}

// Spent is what this month has cost so far and what the ceiling is, both in
// micro-euros, and whether the question could be answered at all.
//
// It exists so one surface can show it. This is the only number in the product
// that accrues and is allowed on a screen, and the exception is narrow on
// purpose: it is money rather than a score, it is bounded by a ceiling you set
// rather than open-ended, and it is a fact about a machine rather than about
// you. Rule 2 bans the counter that counts what remains of *your* work. This
// counts what a model has spent of *its* allowance.
//
// It fails quiet, not closed. A spend that cannot be read is a line that is
// not drawn — unlike Allows, where the same failure has to stop a call.
func (b Budget) Spent(ctx context.Context, personID int64, now time.Time) (int64, int64, bool) {
	if b.Log == nil {
		return 0, 0, false
	}
	spent, err := b.Log.CoachSpentSince(ctx, personID, MonthStart(now))
	if err != nil {
		return 0, 0, false
	}
	return spent, b.CeilingMicros, true
}

// Euros renders micro-euros the way money is written, to the cent.
//
// Rounded up rather than to nearest, so the figure never reads lower than what
// was actually spent. Being told €0.00 after a month of calls would be worse
// than being told €0.01.
func Euros(micros int64) string {
	cents := (micros + 9_999) / 10_000
	return fmt.Sprintf("€%d.%02d", cents/100, cents%100)
}

// Record stores what a call cost, whether or not its answer was used. A reply
// the guard threw away was still paid for, and a budget that only counted the
// good ones would be wrong in the direction that costs money.
func (b Budget) Record(ctx context.Context, personID int64, a Answer) error {
	if b.Log == nil {
		return nil
	}
	a.CostMicros = Cost(a.Model, a.InTokens, a.OutTokens)
	return b.Log.RecordCoachAnswer(ctx, personID, a)
}
