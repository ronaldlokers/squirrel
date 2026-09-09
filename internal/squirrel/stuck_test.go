package squirrel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// No build tag: the ladder is the words and the shape, and neither needs a
// database. It should fail on a laptop rather than only in CI.

// The failure mode being designed against is the twelve-step productivity
// answer. Every branch produces one sentence and at most one control, and this
// is the test that says so.
func TestEveryAnswerIsOneSentenceAndNothingElse(t *testing.T) {
	for _, b := range squirrel.Blockers {
		u := squirrel.UnstuckFor(b)
		if u.Refuse {
			continue
		}
		require.NotEmpty(t, u.Line, string(b))
		require.NotContains(t, u.Line, "\n", "one sentence, never a plan")
		require.NotContains(t, u.Line, "1.", "never numbered")
		require.NotContains(t, u.Line, "•")
	}
}

// Making it smaller ends in something you can see from where you are standing.
// It offered a five-minute timer beside that until the body double went on
// 9 September 2026; the sentence was always the part that did the work.
func TestTooBigOffersTheSmallestPieceYouCanSee(t *testing.T) {
	u := squirrel.UnstuckFor(squirrel.BlockerBig)
	require.Contains(t, u.Line, "smallest")
}

// Not knowing how ends in a question whose answer is a thought — and thoughts
// go where thoughts go.
func TestNotKnowingHowAsksAndCaptures(t *testing.T) {
	u := squirrel.UnstuckFor(squirrel.BlockerHow)
	require.True(t, u.Ask)
}

// Boring was the one an alarm answered. Without the body double the sentence
// carries it alone, and what it says is the same thing: a short go, ended by
// you rather than by a bell.
func TestBoringAsksForAShortGoAndNoBell(t *testing.T) {
	u := squirrel.UnstuckFor(squirrel.BlockerBoring)
	require.Contains(t, u.Line, "short go")
	require.False(t, u.Ask)
	require.NotContains(t, u.Line, "minutes", "the sentence still promises an alarm")
}

func TestNotTodayRefusesAndSaysNothingElse(t *testing.T) {
	u := squirrel.UnstuckFor(squirrel.BlockerNotToday)
	require.True(t, u.Refuse)
	require.Empty(t, u.Line)
}

// This is read by someone who has just said they cannot make a decision, so
// the parsing is generous rather than strict about a wording the product chose
// for itself.
func TestBlockersAreReadGenerously(t *testing.T) {
	for _, said := range []string{"too big", "BIG", "it's too much", "huge"} {
		b, ok := squirrel.ParseBlocker(said)
		require.True(t, ok, said)
		require.Equal(t, squirrel.BlockerBig, b, said)
	}
	for _, said := range []string{"don't know how", "how", "I don't know"} {
		b, ok := squirrel.ParseBlocker(said)
		require.True(t, ok, said)
		require.Equal(t, squirrel.BlockerHow, b, said)
	}
	_, ok := squirrel.ParseBlocker("")
	require.False(t, ok, "nothing said is not an answer")
	_, ok = squirrel.ParseBlocker("purple")
	require.False(t, ok)
}

// Squirrel never asks why twice: the question is asked once, and the answer is
// an answer rather than another question.
func TestTheQuestionOffersTheFourAndAsksNothingElse(t *testing.T) {
	text := squirrel.StuckQuestion().Text
	for _, b := range squirrel.Blockers {
		require.Contains(t, text, squirrel.BlockerWords[b])
	}
	require.Equal(t, 1, strings.Count(text, "?"), "asked once")
}
