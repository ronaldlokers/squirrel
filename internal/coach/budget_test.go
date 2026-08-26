package coach_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// fakeLog stands in for the store. The real one is exercised against Postgres
// in internal/squirrel; what is being tested here is the decision, not the SQL.
type fakeLog struct {
	spent    int64
	err      error
	recorded []coach.Answer
}

func (f *fakeLog) RecordCoachAnswer(_ context.Context, _ int64, a coach.Answer) error {
	f.recorded = append(f.recorded, a)
	return nil
}

func (f *fakeLog) CoachSpentSince(_ context.Context, _ int64, _ time.Time) (int64, error) {
	return f.spent, f.err
}

var august = time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)

func TestMonthStartIsTheFirstOfTheMonthLocally(t *testing.T) {
	require.Equal(t,
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		coach.MonthStart(august))

	// A calendar month, not thirty days back: on the first, nothing has been
	// spent yet however busy the day before was.
	first := time.Date(2026, 9, 1, 0, 30, 0, 0, time.UTC)
	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), coach.MonthStart(first))
}

func TestBudgetAllowsUnderTheCeilingAndRefusesAtIt(t *testing.T) {
	b := coach.Budget{Log: &fakeLog{spent: 9_000_000}, CeilingFor: coach.FlatCeiling(10_000_000)}
	ok, spent := b.Allows(context.Background(), 1, august)
	require.True(t, ok)
	require.Equal(t, int64(9_000_000), spent)

	// Exactly at the ceiling is over it. A ceiling that lets one more call
	// through at the boundary is a ceiling that is off by one call, every
	// month, in the direction that costs money.
	at := coach.Budget{Log: &fakeLog{spent: 10_000_000}, CeilingFor: coach.FlatCeiling(10_000_000)}
	ok, _ = at.Allows(context.Background(), 1, august)
	require.False(t, ok)
}

// Zero is "no ceiling in this process", a supported choice — the provider's own
// spend limit is the real backstop and someone may reasonably prefer only that.
func TestBudgetWithNoCeilingAlwaysAllows(t *testing.T) {
	b := coach.Budget{Log: &fakeLog{spent: 500_000_000}}
	ok, spent := b.Allows(context.Background(), 1, august)
	require.True(t, ok)
	require.Equal(t, int64(500_000_000), spent)
}

// The documented exception to how this product handles a database it cannot
// reach. Everywhere else a failed read costs a feature; here it could cost
// money without limit, so it costs the feature instead.
func TestBudgetFailsClosedWhenTheSpendCannotBeRead(t *testing.T) {
	b := coach.Budget{
		Log:        &fakeLog{err: errors.New("no database")},
		CeilingFor: coach.FlatCeiling(10_000_000),
	}
	ok, _ := b.Allows(context.Background(), 1, august)
	require.False(t, ok)
}

func TestBudgetWithNoLogRefuses(t *testing.T) {
	var b coach.Budget
	ok, _ := b.Allows(context.Background(), 1, august)
	require.False(t, ok)
}

// Pricing happens at write time, so a later change to the price table cannot
// rewrite what last month cost.
func TestRecordPricesTheAnswerItself(t *testing.T) {
	log := &fakeLog{}
	b := coach.Budget{Log: log, CeilingFor: coach.FlatCeiling(10_000_000)}

	require.NoError(t, b.Record(context.Background(), 1, coach.Answer{
		Kind:      "sheet",
		Model:     "gpt-5.6-luna",
		InTokens:  2_000,
		OutTokens: 200,
		Used:      true,
		At:        august,
	}))

	require.Len(t, log.recorded, 1)
	require.Equal(t, int64(640), log.recorded[0].CostMicros)
}

// A reply the guard threw away was still paid for. A budget that counted only
// the good ones would be wrong in the direction that costs money.
func TestRecordCountsARejectedReply(t *testing.T) {
	log := &fakeLog{}
	b := coach.Budget{Log: log}

	require.NoError(t, b.Record(context.Background(), 1, coach.Answer{
		Model:     "gpt-5.6-luna",
		InTokens:  2_000,
		OutTokens: 200,
		Used:      false,
	}))

	require.Len(t, log.recorded, 1)
	require.False(t, log.recorded[0].Used)
	require.Equal(t, int64(640), log.recorded[0].CostMicros)
}

// The only accruing number this product puts on a screen. It is money rather
// than a score, bounded by a ceiling that was set on purpose, and a fact about
// a machine rather than about the person reading it.
func TestSpentReportsTheMonthAndTheCeiling(t *testing.T) {
	b := coach.Budget{Log: &fakeLog{spent: 2_610_000}, CeilingFor: coach.FlatCeiling(10_000_000)}

	spent, ceiling, ok := b.Spent(context.Background(), 1, august)
	require.True(t, ok)
	require.Equal(t, int64(2_610_000), spent)
	require.Equal(t, int64(10_000_000), ceiling)
}

func TestSpentFailsQuietRatherThanClosed(t *testing.T) {
	b := coach.Budget{Log: &fakeLog{err: errors.New("no database")}, CeilingFor: coach.FlatCeiling(10_000_000)}

	_, _, ok := b.Spent(context.Background(), 1, august)
	require.False(t, ok)

	// And the same failure on the gate still refuses the call.
	allowed, _ := b.Allows(context.Background(), 1, august)
	require.False(t, allowed)
}

func TestSpentWithNoLogSaysNothing(t *testing.T) {
	var b coach.Budget
	_, _, ok := b.Spent(context.Background(), 1, august)
	require.False(t, ok)
}

func TestEurosNeverReadsLowerThanWhatWasSpent(t *testing.T) {
	require.Equal(t, "€0.00", coach.Euros(0))
	require.Equal(t, "€0.01", coach.Euros(1))
	require.Equal(t, "€0.01", coach.Euros(640))
	require.Equal(t, "€0.01", coach.Euros(10_000))
	require.Equal(t, "€0.02", coach.Euros(10_001))
	require.Equal(t, "€2.61", coach.Euros(2_610_000))
	require.Equal(t, "€10.00", coach.Euros(10_000_000))
}

// The owner's ceiling and everybody else's are two numbers.
//
// Budget carried one CeilingMicros for the whole process and applied it to
// whoever asked, so two demo accounts would have been two monthly ceilings and
// the second one would have been yours to pay.
func TestTheOwnerAndAGuestHaveDifferentCeilings(t *testing.T) {
	b := coach.Budget{
		Log: &fakeLog{spent: 500_000},
		CeilingFor: func(personID int64) int64 {
			if personID == 1 {
				return 10_000_000
			}
			return 100_000
		},
	}

	ok, _ := b.Allows(context.Background(), 1, august)
	require.True(t, ok, "the owner was refused under their own ceiling")

	ok, _ = b.Allows(context.Background(), 2, august)
	require.False(t, ok, "a guest spent against the owner's ceiling")
}

// What a surface shows is the asker's ceiling too, or a guest reads the
// owner's allowance as their own.
func TestTheCeilingShownIsTheAskersOwn(t *testing.T) {
	b := coach.Budget{
		Log: &fakeLog{spent: 500_000},
		CeilingFor: func(personID int64) int64 {
			if personID == 1 {
				return 10_000_000
			}
			return 100_000
		},
	}

	_, ceiling, ok := b.Spent(context.Background(), 2, august)
	require.True(t, ok)
	require.Equal(t, int64(100_000), ceiling, "a guest was shown the owner's ceiling")
}

// No ceiling function at all is no ceiling, which is the state a build with no
// budget configured is in.
func TestNoCeilingFunctionIsNoCeiling(t *testing.T) {
	b := coach.Budget{Log: &fakeLog{spent: 999_999_999}}
	ok, _ := b.Allows(context.Background(), 1, august)
	require.True(t, ok)
}
