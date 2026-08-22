package coach

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Principle 5 lets the coach evaluate and compare, and the cost recorded when
// it was opened is that it can say something that lands badly on a bad day.
//
// The answer to that is not another instruction. An instruction nobody can
// check is a wish; what changes behaviour is the sentences themselves, handed
// back as things that did not work here.
func TestWhatDidNotLandIsShownToTheModel(t *testing.T) {
	said := System(Now{
		Capacity: "ok",
		LandedBadly: []string{
			"you have done this three times this week",
			"that is not much for a Tuesday",
		},
	}, "sheet")

	require.Contains(t, said, "did not land well")
	require.Contains(t, said, "you have done this three times this week")
	require.Contains(t, said, "that is not much for a Tuesday")
}

// Never a count, here least of all. How often is a fact about the person, and
// rule 2 forbids one on any surface — including the one the person never sees,
// because a model told "four times" will reason about the number.
func TestTheModelIsNeverToldHowOften(t *testing.T) {
	said := System(Now{LandedBadly: []string{"a", "b", "c"}}, "sheet")

	lower := strings.ToLower(said)
	for _, counting := range []string{
		"three", "3 ", "times", "often", "again and again", "repeatedly",
		"history", "pattern of", "usually",
	} {
		require.NotContains(t, lower, counting,
			"the prompt counts what did not land, which is a sentence about the person")
	}
}

// Nothing to say means nothing said. "Nothing has landed badly" would invite
// the model to congratulate itself, which is the shape this product refuses
// everywhere else.
func TestNothingToShowAddsNoSentence(t *testing.T) {
	with := System(Now{Capacity: "ok"}, "sheet")
	require.NotContains(t, strings.ToLower(with), "land")
	require.Equal(t, System(Now{Capacity: "ok"}, "sheet"), with)
}

// And it composes with the voices rather than replacing them: a low day with
// something that did not land gets the plainer voice *and* the examples, which
// is exactly the turn where both matter most.
func TestItComposesWithTheOtherVoices(t *testing.T) {
	said := System(Now{Capacity: "low", LandedBadly: []string{"buck up"}}, KindOverwhelm)

	require.Contains(t, said, "buck up")
	require.Contains(t, said, System(Now{}, KindOverwhelm)[:200],
		"the overwhelm voice went missing")
}
