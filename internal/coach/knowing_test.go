package coach

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Coming to know somebody.

// Without the last line a model handed a list of observations will demonstrate
// that it has them — "I know forms are hard for you" — which is being told
// what a machine has concluded about you, mid-sentence, when you asked about
// the bins. Nothing can enforce it, which is exactly why it has to be said.
func TestTheModelIsToldNotToSayItBack(t *testing.T) {
	said := knowsYou([]string{"Phone calls get done; forms get put off."})

	require.Contains(t, said, "Phone calls get done")
	require.Contains(t, said, "Never say any of this back to them")
	require.Contains(t, said, "never mention that you know it")
}

// Knowing nothing adds no sentence at all, rather than "you know nothing about
// this person" — which would be an instruction to guess.
func TestKnowingNothingSaysNothing(t *testing.T) {
	require.Empty(t, knowsYou(nil))
	require.Empty(t, knowsYou([]string{}))
}

// It reaches the system prompt, which is the only place it can shape anything.
func TestWhatIsKnownReachesThePrompt(t *testing.T) {
	said := System(Now{Knowing: []string{"Forms get put off."}}, "chat")

	require.Contains(t, said, "Forms get put off.")
	require.Contains(t, said, "You are Buddy", "the preamble went")
}

// The preamble refuses the three things this must never produce: a count, an
// absolute, and a judgement about the person rather than about how they work.
func TestThePreambleRefusesTheThreeThings(t *testing.T) {
	for _, refused := range []string{
		"Never count anything", `"always"`, `"never"`, `"every time"`,
		"never what they are like", "no diagnosis",
	} {
		require.Contains(t, knowingPreamble, refused,
			"the preamble stopped refusing %q", refused)
	}
}

// A model that numbered them after being told not to has ignored the shape of
// what was asked, and the numbered ones go.
func TestANumberedObservationIsDropped(t *testing.T) {
	calls := noticedCall("noticed", `{"observations":["1. Forms get put off.","Phone calls get done."]}`)

	got := knowingIn(calls)

	require.Len(t, got, 1, "the numbered one survived")
	require.Equal(t, "Phone calls get done.", got[0])
}

// One bad observation does not take the others with it, which is the opposite
// of what stepsIn does. Half a breakdown is worse than none, because the half
// you keep is the half that fitted; half a set of observations is half a set
// of observations.
func TestOneBadObservationDoesNotTakeTheOthers(t *testing.T) {
	calls := noticedCall("noticed", `{"observations":["   ","Forms get put off.","Phone calls get done."]}`)

	require.Len(t, knowingIn(calls), 2)
}

// Six at most. A model asked for twenty things it has noticed will produce
// twenty, and the last fourteen will be invented.
func TestNoMoreThanSixAreKept(t *testing.T) {
	var many []string
	for range 20 {
		many = append(many, "a thing about how they work")
	}
	calls := noticedCall("noticed", `{"observations":["a","b","c","d","e","f","g","h","i"]}`)

	require.Len(t, knowingIn(calls), mostKnown)
	require.NotEmpty(t, many)
}

func TestAnotherToolIsNotAnObservation(t *testing.T) {
	calls := noticedCall("steps", `{"steps":["open the letter"]}`)

	require.Nil(t, knowingIn(calls))
}

// And nothing sensible out of nonsense.
func TestUnreadableArgumentsAreNoObservations(t *testing.T) {
	calls := noticedCall("noticed", `{`)

	require.Nil(t, knowingIn(calls))
}

// noticedCall builds the tool call shape, which is an anonymous struct on
// toolCall and so cannot be written as a literal from here.
func noticedCall(name, args string) []toolCall {
	var c toolCall
	c.Function.Name = name
	c.Function.Arguments = args
	return []toolCall{c}
}
