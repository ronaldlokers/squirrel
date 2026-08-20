package coach

import (
	"context"
	"fmt"
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
