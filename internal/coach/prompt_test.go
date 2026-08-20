package coach_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

func TestSystemAsksForOneThingAndNoPlan(t *testing.T) {
	s := coach.System(coach.Now{Capacity: "ok"}, "sheet")
	require.Contains(t, s, "Never produce a plan")
	require.Contains(t, s, "Two sentences at most")
	// The line that matters most: a model allowed to decline produces silence,
	// and silence is the deterministic answer taking over.
	require.Contains(t, s, "say nothing rather than something generic")
}

func TestContextIsOneShortLine(t *testing.T) {
	free := 25
	line := coach.Context(coach.Now{
		Clock: "10:42", PartOfDay: "morning", Capacity: "ok", FreeUntil: &free,
	})
	require.Equal(t,
		"It is 10:42, in the morning, capacity is ok, the next fixed thing is in 25 minutes.",
		line)
}

// Nil FreeUntil says nothing at all rather than "nothing is coming". Those are
// different — one means the day is open, the other means nothing was ever
// typed in — and the model must not read the second as the first.
func TestContextSaysNothingAboutAnEmptyDiary(t *testing.T) {
	line := coach.Context(coach.Now{Clock: "10:42", PartOfDay: "morning", Capacity: "ok"})
	require.Equal(t, "It is 10:42, in the morning, capacity is ok.", line)
	require.NotContains(t, line, "fixed")
}

func TestContextOfNothingIsNothing(t *testing.T) {
	require.Empty(t, coach.Context(coach.Now{}))
}

// The model is told a signal, never a diagnosis: "low", never "wiped", and
// never a history. Derived before it ever reaches this package.
func TestContextCarriesNoMoodWord(t *testing.T) {
	line := coach.Context(coach.Now{Clock: "10:42", Capacity: "low"})
	require.Contains(t, line, "capacity is low")
	require.NotContains(t, line, "wiped")
	require.NotContains(t, line, "frazzled")
}

// The overwhelm turn's addition, and the line in it that does the work: every
// instinct a chat model has says to acknowledge what it was told, and
// acknowledging a list is reading the list out.
func TestSystemAddsTheOrderingRuleForAnOverwhelmTurn(t *testing.T) {
	s := coach.System(coach.Now{Capacity: "ok"}, coach.KindOverwhelm)
	require.Contains(t, s, "do not reflect it back")
	require.Contains(t, s, "Choose ONE")
	require.Contains(t, s, "Do not list the rest")

	require.NotContains(t, coach.System(coach.Now{Capacity: "ok"}, "sheet"), "Choose ONE")
}

// They compose, and the turn where both matter most is the one where both
// apply: several things at once, on a bad day.
func TestSystemComposesOverwhelmAndTheLowVoice(t *testing.T) {
	s := coach.System(coach.Now{Capacity: "low"}, coach.KindOverwhelm)
	require.Contains(t, s, "Choose ONE")
	require.Contains(t, s, "drop warmth and character")
}
