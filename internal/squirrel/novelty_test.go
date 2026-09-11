package squirrel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The stamp leans differently on different days, and never far.
//
// Two things at once, because either alone is satisfiable by a bug: a constant
// is inside the band and never varies, and a raw hash varies and lands
// anywhere. A stamp at 40 degrees is not slapped on, it is broken.
func TestTheStampLeansWithinAFewDegrees(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	seen := map[int]bool{}
	for d := 0; d < 400; d++ {
		got := Tilt(start.AddDate(0, 0, d))
		require.GreaterOrEqual(t, got, TiltFrom, "the stamp leaned too far")
		require.LessOrEqual(t, got, TiltTo, "the stamp barely leaned")
		require.Negative(t, got, "the stamp leaned the other way")
		seen[got] = true
	}
	require.Greater(t, len(seen), 4,
		"the stamp used %d of its %d angles in a year", len(seen), TiltTo-TiltFrom+1)
}

// The vertical and the alpha are not here to be tested because they are not
// variables: .35 is a measured contrast result, and this is the shape of the
// change that cannot touch it.
func TestTheLightMovesAcrossTheField(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	seen := map[int]bool{}
	for d := 0; d < 400; d++ {
		got := Light(start.AddDate(0, 0, d))
		require.GreaterOrEqual(t, got, LightFrom)
		require.LessOrEqual(t, got, LightTo)
		seen[got] = true
	}
	require.Greater(t, len(seen), 4,
		"the light stood in %d places in a year", len(seen))
}

// Both hold still for a day: a phone and a desktop are one product, and a
// reload is not a slot machine.
func TestTheDayHoldsTheAngleAndTheLight(t *testing.T) {
	morning := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	night := time.Date(2026, 8, 22, 23, 59, 0, 0, time.UTC)

	require.Equal(t, Tilt(morning), Tilt(night))
	require.Equal(t, Light(morning), Light(night))
}

// pick is one implementation shared by the stamp, the light and anything added
// later. Without the salt every caller over the same band would return the
// same number on the same day — the stamp and the light happen not to show it
// because their bands are different widths, which is luck rather than design
// and would stop being true the moment somebody added a third.
func TestTwoThingsChosenFromOneDayDisagree(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	same := 0
	for d := 0; d < 200; d++ {
		on := start.AddDate(0, 0, d)
		if pick("one", on, 0, 19) == pick("other", on, 0, 19) {
			same++
		}
	}
	// A twentieth of 200 is 10 by chance. Twice that is generous and still
	// nowhere near the 200 an unsalted pick would score.
	require.Less(t, same, 20,
		"two different things picked the same number on %d days of 200", same)
}
