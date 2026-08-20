package coach_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// The turn the project exists for, and the only routine one that escalates.
// Detection is rules, because a model asked "is this overwhelm?" before every
// turn doubles the calls to answer a question that costs nothing to answer
// badly.

func TestOverwhelmIsAPileOfThingsAtOnce(t *testing.T) {
	for _, said := range []string{
		"the tax thing, the vet, the bins and mum's birthday",
		"tax\nvet\nbins\nbirthday",
		"I need to ring the vet and put the bins out and do the tax",
		"bins, vet, tax",
		// A list written as lines with commas inside one of them is still one
		// list, and the line split is what settles it.
		"ring the vet, if they are open\nput the bins out\nthe tax thing",
	} {
		require.True(t, coach.Overwhelmed(said), "%q read as an ordinary question", said)
	}
}

func TestAnOrdinaryQuestionIsNotOverwhelm(t *testing.T) {
	for _, said := range []string{
		"what now",
		"I can't face the tax thing",
		// Two things is a sentence people write when they are fine. Three is
		// the bar, and it is set there on purpose.
		"the tax thing and the vet",
		"the bins, and then the vet",
		"",
		"   ",
	} {
		require.False(t, coach.Overwhelmed(said), "%q read as overwhelm", said)
	}
}

// "and" is a word, not three letters. Splitting on the letters would make
// "standing" a list.
func TestOverwhelmDoesNotSplitInsideWords(t *testing.T) {
	require.False(t, coach.Overwhelmed(
		"I have been standing in the hallway understanding nothing and it is bad"))
}

// Trailing punctuation leaves fragments, and a fragment is not a thing. Two
// real items with stray commas must not add up to three.
func TestOverwhelmIgnoresFragments(t *testing.T) {
	require.False(t, coach.Overwhelmed("the vet, the bins,,"))
	require.False(t, coach.Overwhelmed("vet\n\n\nbins"))
}

// A long single thought is not a list however long it runs. The failure being
// avoided is escalating on length, which would make every bad day expensive.
func TestALongSentenceIsNotOverwhelm(t *testing.T) {
	require.False(t, coach.Overwhelmed(
		"I have been trying to start this one thing all morning and I keep opening "+
			"the page and closing it again without reading any of it"))
}
