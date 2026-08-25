package boot

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The owner's monthly ceiling and a guest's are two numbers, and telling them
// apart is boot's job because boot is the only place that knows who the owner
// is.
//
// One process-wide ceiling would have made every demo account another monthly
// allowance, all of them yours to pay.
func TestAGuestDoesNotSpendTheOwnersAllowance(t *testing.T) {
	cfg := squirrel.CoachConfig{BudgetMicros: 10_000_000, GuestBudgetMicros: 1_000_000}
	b := budgetFor(cfg, nil, func() int64 { return 7 })

	require.Equal(t, int64(10_000_000), b.CeilingFor(7))
	require.Equal(t, int64(1_000_000), b.CeilingFor(8), "a guest was given the owner's ceiling")
}

// Before Postgres answers, the owner is zero. A person id of zero must not
// match it and collect the owner's allowance.
func TestNobodyIsNotTheOwner(t *testing.T) {
	cfg := squirrel.CoachConfig{BudgetMicros: 10_000_000, GuestBudgetMicros: 1_000_000}
	b := budgetFor(cfg, nil, func() int64 { return 0 })

	require.Equal(t, int64(1_000_000), b.CeilingFor(0),
		"a request with nobody on it was given the owner's ceiling")
}
