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
	b := coach.Budget{Log: &fakeLog{spent: 9_000_000}, CeilingMicros: 10_000_000}
	ok, spent := b.Allows(context.Background(), 1, august)
	require.True(t, ok)
	require.Equal(t, int64(9_000_000), spent)

	// Exactly at the ceiling is over it. A ceiling that lets one more call
	// through at the boundary is a ceiling that is off by one call, every
	// month, in the direction that costs money.
	at := coach.Budget{Log: &fakeLog{spent: 10_000_000}, CeilingMicros: 10_000_000}
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
		Log:           &fakeLog{err: errors.New("no database")},
		CeilingMicros: 10_000_000,
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
	b := coach.Budget{Log: log, CeilingMicros: 10_000_000}

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
