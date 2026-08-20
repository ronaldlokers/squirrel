package coach_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

func TestSystemAsksForOneThingAndNoPlan(t *testing.T) {
	s := coach.System(coach.Now{Capacity: "ok"})
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
