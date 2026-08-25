package squirrel

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The same day says the same thing, twice and on both devices.
//
// Not random, and that is the design rather than a shortcut. A phone and a
// desktop are one product, so a line that differs between them is a bug rather
// than a variation — and a sentence that changes while you are reading it is
// worse than one that never changes.
func TestADaySaysOneThing(t *testing.T) {
	day := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	later := time.Date(2026, 8, 22, 23, 30, 0, 0, time.UTC)

	for _, what := range everySaying {
		require.Equal(t, Say(what, day), Say(what, later), "%s changed during the day", what)
		require.NotEmpty(t, Say(what, day))
	}
}

// And the days do not all say the same thing, or none of this is worth having.
//
// A fortnight is the window the check-in already uses and about twice the week
// PRODUCT.md says a surface takes to stop being seen.
func TestAFortnightIsNotOneSentence(t *testing.T) {
	start := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)

	for _, what := range everySaying {
		seen := map[string]bool{}
		for d := 0; d < 14; d++ {
			seen[Say(what, start.AddDate(0, 0, d))] = true
		}
		require.Greater(t, len(seen), 2,
			"%s said %d different things in a fortnight", what, len(seen))
	}
}

// Every wording is one of the wordings. A day that lands outside the pool
// would be a sentence nobody wrote.
// everySaying is the whole set, so a pool added later is held to the rules
// without anybody remembering to add it to four lists. The acknowledgements
// joined on 25 August 2026 and this is what stopped them arriving unguarded.
var everySaying = []Saying{
	SayingSlot, SayingOffer, SayingStop, SayingEnough,
	SayingDid, SayingKept, SayingDropped, SayingDecided,
	SayingHere, SayingLater, SayingHeard,
}

func TestEveryDaySaysSomethingSomebodyWrote(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, what := range everySaying {
		pool := map[string]bool{}
		for _, said := range Sayings(what) {
			pool[said] = true
		}
		for d := 0; d < 400; d++ {
			said := Say(what, start.AddDate(0, 0, d))
			require.True(t, pool[said], "%s said %q, which is not in its pool", what, said)
		}
	}
}

// What shipped is still what it says on the days it picks it. The pools are
// variations on the product's own words, not a replacement for them.
func TestTheOriginalWordingIsStillInEveryPool(t *testing.T) {
	for what, first := range map[Saying]string{
		SayingSlot:   "tell me a thing",
		SayingOffer:  "RIGHT NOW",
		SayingStop:   "stop whenever you like",
		SayingEnough: "that will do",
		// The acknowledgements' own originals, which are the words the
		// conversation shipped with.
		SayingDid:     "Good.",
		SayingKept:    "Kept.",
		SayingHere:    "This one.",
		SayingDecided: "On the list.",
	} {
		require.Equal(t, first, Sayings(what)[0], "%s no longer leads with what shipped", what)
	}
}

// The rules every wording is held to, because the pools are the one place in
// this product where prose was written in bulk and the guard rails matter more
// than usual.
func TestNoWordingBreaksTheRules(t *testing.T) {
	for _, what := range everySaying {
		for _, said := range Sayings(what) {
			require.NotEmpty(t, strings.TrimSpace(said))

			// Rule 2. Nothing accrues, in any wording, on any surface.
			for _, digit := range "0123456789" {
				require.NotContains(t, said, string(digit),
					"%q carries a number", said)
			}
			for _, banned := range []string{"left", "still", "remaining", "again", "streak", "!"} {
				require.NotContains(t, strings.ToLower(said), banned,
					"%q says something about what is outstanding, or demands", said)
			}

			// Short enough to be read on the way past rather than read.
			require.Less(t, len(said), 40, "%q is a paragraph", said)
		}
	}
}

// Stopping never sounds like failure, in any of its wordings. Principle 3 is
// the one this pool could most easily break by accident: a cheerful line about
// leaving is a line that says leaving needs cheering up.
func TestNoWordingMakesStoppingSoundLikeFailure(t *testing.T) {
	for _, what := range []Saying{SayingStop, SayingEnough} {
		for _, said := range Sayings(what) {
			for _, wrong := range []string{
				"quit", "give up", "fail", "sorry", "at least", "only", "just",
				"well done", "congrat", "great", "nice",
			} {
				require.NotContains(t, strings.ToLower(said), wrong,
					"%q makes stopping sound like something other than a normal ending", said)
			}
		}
	}
}

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

// The room's light moves across, and only across.
//
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

// Both hold still for a day, for the same reason the sentences do: a phone and
// a desktop are one product, and a reload is not a slot machine.
func TestTheDayHoldsTheAngleAndTheLight(t *testing.T) {
	morning := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	night := time.Date(2026, 8, 22, 23, 59, 0, 0, time.UTC)

	require.Equal(t, Tilt(morning), Tilt(night))
	require.Equal(t, Light(morning), Light(night))
}

// The salt is mixed in, so two things chosen from the same day disagree.
//
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
