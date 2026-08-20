package coach_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// The arithmetic worked by hand, because a budget that is quietly out by a
// factor of a hundred looks exactly like a budget that works.
//
// Luna is 20 cents per million in, 120 out. A million in and a million out is
// 140 cents, which is 1.4 euros, which is 1,400,000 micro-euros.
func TestCostIsInMicroEuros(t *testing.T) {
	require.Equal(t, int64(1_400_000), coach.Cost("gpt-5.6-luna", 1_000_000, 1_000_000))
	require.Equal(t, int64(14_000_000), coach.Cost("gpt-5.6-terra", 1_000_000, 1_000_000))
}

// The reason the unit is micro-euros rather than cents: a realistic turn must
// not round to nothing. Two thousand in and two hundred out on Luna is 640
// micro-euros — under a tenth of a cent, and a number a month of calls can
// still be summed from.
func TestCostOfARealisticTurnIsNotZero(t *testing.T) {
	require.Equal(t, int64(640), coach.Cost("gpt-5.6-luna", 2_000, 200))
}

// A month at the estimate stays well under the ceiling. Three hundred turns of
// that size is about twenty cents; the ceiling is ten euros.
func TestAMonthOfTurnsFitsTheCeiling(t *testing.T) {
	month := coach.Cost("gpt-5.6-luna", 2_000, 200) * 300
	require.Less(t, month, int64(10_000_000))
}

// Stated rather than assumed: an unknown model costs nothing, which means the
// ceiling stops protecting anything. Boot warns about it for that reason.
func TestUnknownModelCostsNothingAndIsNotKnown(t *testing.T) {
	require.Equal(t, int64(0), coach.Cost("gpt-4o", 1_000_000, 1_000_000))
	require.False(t, coach.KnownModel("gpt-4o"))
	require.True(t, coach.KnownModel("gpt-5.6-luna"))
	require.True(t, coach.KnownModel("gpt-5.6-terra"))
	require.True(t, coach.KnownModel("gpt-5.6-sol"))
}
