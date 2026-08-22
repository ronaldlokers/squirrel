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

	for _, what := range []Saying{SayingSlot, SayingOffer, SayingStop, SayingEnough} {
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

	for _, what := range []Saying{SayingSlot, SayingOffer, SayingStop, SayingEnough} {
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
func TestEveryDaySaysSomethingSomebodyWrote(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, what := range []Saying{SayingSlot, SayingOffer, SayingStop, SayingEnough} {
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
	} {
		require.Equal(t, first, Sayings(what)[0], "%s no longer leads with what shipped", what)
	}
}

// The rules every wording is held to, because the pools are the one place in
// this product where prose was written in bulk and the guard rails matter more
// than usual.
func TestNoWordingBreaksTheRules(t *testing.T) {
	for _, what := range []Saying{SayingSlot, SayingOffer, SayingStop, SayingEnough} {
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
