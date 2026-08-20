package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// No build tag: the parser and the arithmetic are the load-bearing halves and
// neither needs a database.

func at(hour, minute int) time.Time {
	return time.Date(2026, 8, 20, hour, minute, 0, 0, time.UTC)
}

func TestAFixedPointIsReadFromASentence(t *testing.T) {
	m, ok := squirrel.ParseMoment("at 14:30 dentist", at(9, 0))
	require.True(t, ok)
	require.Equal(t, "dentist", m.Label)
	require.Equal(t, at(14, 30), m.Starts)
	require.True(t, m.Guessed, "nobody said how far away it is")
}

func TestTravelTimeIsPeeledOffTheEnd(t *testing.T) {
	m, ok := squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", at(9, 0))
	require.True(t, ok)
	require.Equal(t, "dentist", m.Label)
	require.Equal(t, 20*time.Minute, m.Travel)
	require.False(t, m.Guessed)
	require.Equal(t, at(14, 10), m.LeaveAt())
	require.Equal(t, at(14, 0), m.WarnAt())
}

func TestAmAndPmAreRead(t *testing.T) {
	m, ok := squirrel.ParseMoment("at 2pm dentist", at(9, 0))
	require.True(t, ok)
	require.Equal(t, at(14, 0), m.Starts)

	m, ok = squirrel.ParseMoment("at 9am school run", at(6, 0))
	require.True(t, ok)
	require.Equal(t, at(9, 0), m.Starts)
}

// A time that has gone is tomorrow, which is what a person means when they
// type it.
func TestATimeAlreadyPassedIsTomorrow(t *testing.T) {
	m, ok := squirrel.ParseMoment("at 09:00 school run", at(14, 0))
	require.True(t, ok)
	require.Equal(t, at(9, 0).AddDate(0, 0, 1), m.Starts)
}

func TestTomorrowIsRead(t *testing.T) {
	m, ok := squirrel.ParseMoment("tomorrow 09:00 school run", at(6, 0))
	require.True(t, ok)
	require.Equal(t, at(9, 0).AddDate(0, 0, 1), m.Starts)
}

// The bar is the same one the chore definition sets: when in doubt, capture.
// Someone writing a thought down should never have to escape it.
func TestProseIsNotAFixedPoint(t *testing.T) {
	for _, said := range []string{
		"14:30 dentist",           // no "at", no "tomorrow"
		"at 5 dentist",            // a number, not a time
		"at the dentist tomorrow", // no time at all
		"the tyre is flat",        // a thought
		"at 25:00 nonsense",       // not a time
		"at 14:30",                // no label
		"ring the garage at 14:30 about the noise", // a thought that happens to contain one
	} {
		_, ok := squirrel.ParseMoment(said, at(9, 0))
		require.False(t, ok, "%q should stay a note", said)
	}
}

// Nothing here is ever late, and nothing says hurry.
func TestTheLeaveSentenceNeverPushes(t *testing.T) {
	m, _ := squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", at(9, 0))
	words := squirrel.LeaveWords(m)

	require.Contains(t, words, "14:30")
	require.Contains(t, words, "14:10")
	for _, banned := range []string{"late", "hurry", "overdue", "!"} {
		require.NotContains(t, words, banned)
	}
}

// A guess says it is a guess. A default that presents itself as a fact is how
// someone ends up late while trusting the machine.
func TestAGuessedTravelTimeSaysSo(t *testing.T) {
	m, _ := squirrel.ParseMoment("at 14:30 dentist", at(9, 0))
	require.Contains(t, squirrel.LeaveWords(m), "if it is")

	m, _ = squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", at(9, 0))
	require.NotContains(t, squirrel.LeaveWords(m), "if it is")
}

// The window opens one "get ready" before leaving and closes when it starts:
// once you are late there is nothing useful left to say.
func TestTheWindowOpensBeforeLeavingAndClosesAtTheStart(t *testing.T) {
	m, _ := squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", at(9, 0))

	require.False(t, m.Open(at(13, 59)))
	require.True(t, m.Open(at(14, 0)))
	require.True(t, m.Open(at(14, 29)))
	require.False(t, m.Open(at(14, 30)))
	require.False(t, m.Open(at(15, 0)))
}

// The matcher's own bar, from the other side: this is what makes a fixed point
// reachable without a command.
func TestMatchRecognisesADeliberateFixedPoint(t *testing.T) {
	in := squirrel.Match("at 14:30 dentist")
	require.Equal(t, squirrel.IntentMoment, in.Kind)
	require.Equal(t, "dentist", in.At.Label)

	require.Equal(t, squirrel.IntentCapture, squirrel.Match("14:30 dentist").Kind)
	require.Equal(t, squirrel.IntentCapture, squirrel.Match(".at 14:30 dentist").Kind,
		"the escape hatch still wins")
}
