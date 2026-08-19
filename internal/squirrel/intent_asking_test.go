package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

const aDay = 24 * time.Hour

// "Every other tuesday" is two facts, not one: a fortnightly rhythm, and a
// preference for which day to raise it on. Reading it as a rhythm alone throws
// away the half the person cared enough to type.
func TestEveryOtherTuesdayIsARhythmAndAPreference(t *testing.T) {
	name, every, asking, ok := squirrel.ParseEveryAsking("every other tuesday bins out")

	require.True(t, ok)
	require.Equal(t, "bins out", name)
	require.Equal(t, 14*aDay, every)
	require.Equal(t, squirrel.OnlyOn(time.Tuesday), asking.Days)
	require.Equal(t, squirrel.AnyPart, asking.Part, "the day was said, the hour was not")
}

func TestTheWiderVocabulary(t *testing.T) {
	for _, c := range []struct {
		in    string
		every time.Duration
		days  squirrel.Days
	}{
		{"every tuesday bins out", 7 * aDay, squirrel.OnlyOn(time.Tuesday)},
		{"every other week vacuum", 14 * aDay, squirrel.AnyDay},
		{"every fortnight vacuum", 14 * aDay, squirrel.AnyDay},
		{"every quarter the boiler", 90 * aDay, squirrel.AnyDay},
		{"every year the alarms", 365 * aDay, squirrel.AnyDay},
		{"every weekday the tablets", aDay, squirrel.Weekdays},
		{"every 3 days water plants", 3 * aDay, squirrel.AnyDay},
		{"every fri put the bins out", 7 * aDay, squirrel.OnlyOn(time.Friday)},
	} {
		name, every, asking, ok := squirrel.ParseEveryAsking(c.in)
		require.True(t, ok, c.in)
		require.NotEmpty(t, name, c.in)
		require.Equal(t, c.every, every, c.in)
		require.Equal(t, c.days, asking.Days, c.in)
	}
}

// "Every other 3 weeks" is not a thing anyone means, and guessing which half
// they meant is worse than saying no — a rejected definition is captured as a
// note, which loses nothing.
func TestOtherDoesNotCombineWithANumber(t *testing.T) {
	_, _, _, ok := squirrel.ParseEveryAsking("every other 3 weeks vacuum")
	require.False(t, ok)
}

// A preference is permission, not a deadline: an empty one never closes a
// window, and missing a window never makes a chore late.
func TestAnEmptyPreferenceIsAlwaysOpen(t *testing.T) {
	var a squirrel.Asking
	for h := 0; h < 24; h++ {
		at := time.Date(2026, 8, 20, h, 0, 0, 0, time.Local)
		require.True(t, a.Open(at), at.String())
	}
}

func TestTheWindowIsTheDayAndThePart(t *testing.T) {
	a := squirrel.Asking{Days: squirrel.OnlyOn(time.Tuesday), Part: squirrel.Evening}

	tuesEve := time.Date(2026, 8, 18, 19, 0, 0, 0, time.Local)
	require.Equal(t, time.Tuesday, tuesEve.Weekday())
	require.True(t, a.Open(tuesEve))

	require.False(t, a.Open(tuesEve.Add(-8*time.Hour)), "tuesday morning is not the evening")
	require.False(t, a.Open(tuesEve.Add(24*time.Hour)), "wednesday evening is not tuesday")
}

// Nothing asks in the small hours, in any part.
func TestNothingAsksAtThreeInTheMorning(t *testing.T) {
	for _, p := range []squirrel.DayPart{squirrel.Morning, squirrel.Afternoon, squirrel.Evening} {
		a := squirrel.Asking{Part: p}
		require.False(t, a.Open(time.Date(2026, 8, 20, 3, 0, 0, 0, time.Local)), string(p))
	}
}

// Said the way a person says it, and silent when there is nothing to say —
// "any time, any day" is noise on a chore with no preference at all.
func TestAPreferenceSaysItselfOrNothing(t *testing.T) {
	require.Equal(t, "", squirrel.Asking{}.Words())
	require.Equal(t, "evenings", squirrel.Asking{Part: squirrel.Evening}.Words())
	require.Equal(t, "weekdays", squirrel.Asking{Days: squirrel.Weekdays}.Words())
	require.Equal(t, "tuesdays, evenings",
		squirrel.Asking{Days: squirrel.OnlyOn(time.Tuesday), Part: squirrel.Evening}.Words())
}
