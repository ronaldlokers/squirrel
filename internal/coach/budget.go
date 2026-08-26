package coach

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// The monthly ceiling.
//
// What it protects against: Squirrel's own bugs — a retry loop, a context that
// grows unnoticed, a phase that calls the model more than the estimate said. It
// cannot protect against a stolen key, which is what the provider's own hard
// spend limit is for.
//
// Crossing it is not an error state: it degrades to the deterministic floor for
// the rest of the month, which is what makes a ceiling safe to set low.

// Answer is one recorded call, for the log and the budget. The whole exchange is
// kept indefinitely: it is what makes a bad answer inspectable and a model change
// evaluable. It is also a permanent record of bad moments and what a machine said
// about them.
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
	// CeilingFor is the monthly ceiling in micro-euros for one person. Nil, or zero,
	// means no ceiling.
	//
	// A function rather than a number because a demo account must not spend a month's
	// allowance: one process-wide ceiling would make two demo accounts into two
	// monthly ceilings, both yours to pay.
	CeilingFor func(personID int64) int64
}

func (b Budget) ceiling(personID int64) int64 {
	if b.CeilingFor == nil {
		return 0
	}
	return b.CeilingFor(personID)
}

// FlatCeiling is one ceiling for everybody. It is what a single-person
// deployment wants and what most tests want; the two-tier version lives in
// internal/boot, where the owner is known.
func FlatCeiling(micros int64) func(int64) int64 {
	return func(int64) int64 { return micros }
}

// MonthStart is the first instant of now's calendar month, locally. The budget
// is a calendar month rather than a rolling thirty days because that is what a
// provider's invoice is, and two windows that nearly agree are worse than one.
func MonthStart(now time.Time) time.Time {
	y, m, _ := now.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, now.Location())
}

// Allows reports whether a call may be made, and what has been spent.
//
// It fails closed: if the spend cannot be read, no call is made. The opposite of
// everywhere else here, deliberately — elsewhere the failure costs a feature, and
// this one could cost money without limit.
func (b Budget) Allows(ctx context.Context, personID int64, now time.Time) (bool, int64) {
	if b.Log == nil {
		return false, 0
	}
	spent, err := b.Log.CoachSpentSince(ctx, personID, MonthStart(now))
	if err != nil {
		return false, 0
	}
	ceiling := b.ceiling(personID)
	if ceiling <= 0 {
		return true, spent
	}
	return spent < ceiling, spent
}

// Permit is proof that the month's ceiling was checked before a paid call. It
// carries nothing; its whole job is to be impossible to produce without asking.
//
// The floor used to hold because six methods each remembered an identical
// four-line check — correct in all six, enforced in none.
// completionWithTools will not compile without one of these, so the floor is a
// property of the type system rather than of anyone's memory.
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
//
// Every call site defers it rather than releasing at the end: the end is not
// the only exit.
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
// logged. `instead` names what takes over, because crossing the ceiling is the
// system working.
//
// One paid call at a time. The ceiling is checked before a call and the spend
// recorded after, so two requests arriving together could both read "under
// ceiling" and both spend.
//
// A channel rather than a mutex, because this one has to be able to give up: a
// reservation that is never settled hangs every future call. If the holder takes
// longer than any real call could, the next caller says so and goes anyway.
//
// Single replica, one person. Two pods would need this in the database.
var spending = make(chan struct{}, 1)

// spendWait is longer than a model call and shorter than a person's patience. A
// var rather than a const so a test can make it short: a ninety-second escape
// hatch is one no test would wait for, leaving the safety net unproven.
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
			"spent_micros", spent, "ceiling_micros", b.ceiling(personID))
		return Permit{}, ErrUnavailable
	}
	return Permit{held: held}, nil
}

// Spent is what this month has cost and what the ceiling is, in micro-euros, and
// whether the question could be answered.
//
// The only accruing number allowed on a screen: money rather than a score,
// bounded by a ceiling you set, and a fact about a machine.
//
// It fails quiet, not closed — unlike Allows, where the same failure must stop a
// call.
func (b Budget) Spent(ctx context.Context, personID int64, now time.Time) (int64, int64, bool) {
	if b.Log == nil {
		return 0, 0, false
	}
	spent, err := b.Log.CoachSpentSince(ctx, personID, MonthStart(now))
	if err != nil {
		return 0, 0, false
	}
	return spent, b.ceiling(personID), true
}

// Euros renders micro-euros to the cent, rounded up so the figure never reads
// lower than what was spent.
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
