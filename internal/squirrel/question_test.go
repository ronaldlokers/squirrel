package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The floor under the reading path: no model, no network, no cluster.
//
// Deliberately narrow. Match's own rule applies word for word — when in doubt
// the answer is always capture — because the two failures are not the same
// size. A question read as a thought is a note in the pile you could have had
// answered, and there is a chip on the answer to fix it. A thought read as a
// question is a thought dropped out of the pile, which is the one failure this
// product does not have.

func TestAQuestionMarkAtTheEndIsAQuestion(t *testing.T) {
	require.True(t, squirrel.LooksLikeAQuestion("what should I do about the tax thing?"))
	require.True(t, squirrel.LooksLikeAQuestion("  can it wait?  "))
}

// In the middle it is punctuation rather than a request: somebody thinking on
// the page.
func TestAQuestionMarkInTheMiddleIsThinkingAloud(t *testing.T) {
	require.False(t, squirrel.LooksLikeAQuestion("ring the vet? no, the dentist"))
	require.False(t, squirrel.LooksLikeAQuestion("bins? already done"))
}

// The openings that make a sentence a question without a mark. Every one of
// them is a thing you say to somebody rather than about something.
func TestSomeOpeningsAreQuestionsWithoutAMark(t *testing.T) {
	for _, asked := range []string{
		"what should I do about the boiler",
		"should i ring them today",
		"how do i even start this",
		"can you break this down",
		"any idea where the meter is",
	} {
		require.True(t, squirrel.LooksLikeAQuestion(asked), "%q read as a thought", asked)
	}
}

// And the ones that only look like them. "What a day" is not a question and
// neither is a note that happens to start with a word a question could.
func TestAThoughtThatStartsLikeAQuestionIsStillAThought(t *testing.T) {
	for _, thought := range []string{
		"what a day",
		"whatever happens, ring the vet first",
		"how the boiler sounds when it starts",
		"can opener is broken",
		"shallots for the soup",
	} {
		require.False(t, squirrel.LooksLikeAQuestion(thought), "%q read as a question", thought)
	}
}

// A command is never a question. The grammar has already claimed these and
// answering them with a model would be answering something the product knows
// how to do itself.
func TestACommandIsNeverAQuestion(t *testing.T) {
	for _, command := range []string{
		"!find boiler", "!at 14:30 dentist", "!notes", "!help",
	} {
		require.False(t, squirrel.LooksLikeAQuestion(command), "%q read as a question", command)
	}
}

// The escape hatch wins over everything, including this. A leading dot means
// "keep exactly this", and it must not be read for intent at all.
func TestTheEscapeHatchIsNeverAQuestion(t *testing.T) {
	require.False(t, squirrel.LooksLikeAQuestion(".what should I do?"))
}

// Nothing is not a question.
func TestNothingIsNotAQuestion(t *testing.T) {
	require.False(t, squirrel.LooksLikeAQuestion(""))
	require.False(t, squirrel.LooksLikeAQuestion("   "))
}

// The bias, stated as a test rather than as a comment: of the things somebody
// actually types into this box, the overwhelming majority are thoughts, and
// the rule keeps them.
func TestTheRuleKeepsAlmostEverything(t *testing.T) {
	typed := []string{
		"the boiler is making that noise again",
		"bins out thursday",
		"meter reading 48213",
		"ring the vet about the booster",
		"buy milk",
		"tax letter came",
		"ask mum about the weekend",
		"that podcast about sourdough",
		"what should I do about the tax thing?",
	}
	questions := 0
	for _, one := range typed {
		if squirrel.LooksLikeAQuestion(one) {
			questions++
		}
	}
	require.Equal(t, 1, questions,
		"the rule sent %d of %d captures abroad", questions, len(typed))
}
