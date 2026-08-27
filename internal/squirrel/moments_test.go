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

// The same sentence, on a day you chose.
//
// ParseMoment builds from today's date, so there is no way to say a date at
// all. Rather than widen the grammar — the "at" or "tomorrow" bar exists so a
// stray thought is never silently turned into something that interrupts you —
// the same parser is anchored somewhere else.
func TestMomentOnPutsItOnTheChosenDay(t *testing.T) {
	now := time.Date(2026, 8, 24, 9, 0, 0, 0, time.Local)
	day := time.Date(2026, 8, 27, 0, 0, 0, 0, time.Local)

	m, ok := squirrel.MomentOn(nil, day, "at 14:30 dentist", now)
	require.True(t, ok)
	require.Equal(t, 2026, m.Starts.Year())
	require.Equal(t, time.August, m.Starts.Month())
	require.Equal(t, 27, m.Starts.Day())
	require.Equal(t, 14, m.Starts.Hour())
	require.Equal(t, 30, m.Starts.Minute())
}

// An hour already gone today does not move a day that was chosen: choosing the
// 27th means the 27th, whatever o'clock it is now.
func TestAChosenDayIsNotRolledForward(t *testing.T) {
	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.Local)
	day := time.Date(2026, 8, 27, 0, 0, 0, 0, time.Local)

	m, ok := squirrel.MomentOn(nil, day, "at 09:00 dentist", now)
	require.True(t, ok)
	require.Equal(t, 27, m.Starts.Day())
}

// And ParseMoment is unchanged, in both directions — an hour still ahead is
// today, and one that has gone is tomorrow. Both halves, because a version
// that simply anchored a day later satisfies the first on its own.
func TestParseMomentStillRollsForwardAndOnlyThen(t *testing.T) {
	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.Local)

	gone, ok := squirrel.ParseMoment("at 09:00 dentist", now)
	require.True(t, ok)
	require.Equal(t, 25, gone.Starts.Day(), "an hour that has gone is tomorrow")

	ahead, ok := squirrel.ParseMoment("at 20:00 dentist", now)
	require.True(t, ok)
	require.Equal(t, 24, ahead.Starts.Day(), "an hour still ahead is today")
}

// The bar is unchanged. A chosen day does not make a bare time a fixed point:
// the picker composes a sentence that clears the bar, and anything that does
// not clear it is still a note.
func TestAChosenDayDoesNotLowerTheBar(t *testing.T) {
	now := time.Now()

	_, ok := squirrel.MomentOn(nil, now.AddDate(0, 0, 3), "14:30 dentist", now)
	require.False(t, ok)
}

// The clock a container happens to run on is not where the person is.
//
// This is the test issue #148 asked for, and it is the point of the fix: the
// fault was invisible because a confirmation restates your own time in the
// wrong zone, so a booking two hours late reads byte-for-byte like a correct
// one. TZ is forced to UTC here, which is what production had.
func TestAFixedPointIsBookedWhereThePersonIs(t *testing.T) {
	t.Setenv("TZ", "UTC")
	utc := time.FixedZone("UTC", 0)
	here, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	// 04:00 on a summer morning, on a process that thinks it is in UTC.
	now := time.Date(2026, 8, 24, 4, 0, 0, 0, utc)

	m, ok := squirrel.ParseMomentIn(here, "at 04:42 test", now)
	require.True(t, ok)

	// The instant is 04:42 where the person is, not where the process is.
	require.Equal(t, here.String(), m.Starts.Location().String())
	require.Equal(t, 4, m.Starts.Hour())
	require.Equal(t, 42, m.Starts.Minute())
	require.Equal(t, 2, m.Starts.UTC().Hour(),
		"04:42 in Amsterdam is 02:42 UTC; a process clock would have booked 04:42 UTC")
}

// And the day a refusal belongs to is the person's day, not the process's.
//
// "Not now means today, because tomorrow is a fresh question." On a UTC process
// in summer, today ended at 02:00 local — so after 02:00 it meant about an hour.
func TestTodayIsThePersonsDay(t *testing.T) {
	t.Setenv("TZ", "UTC")
	here, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	// 00:30 local on the 25th is 22:30 UTC on the 24th: the two disagree about
	// which day it is, which is the whole of the bug.
	now := time.Date(2026, 8, 24, 22, 30, 0, 0, time.UTC)

	require.Equal(t, 25, squirrel.StartOfDayIn(here, now).Day(),
		"the refusal window followed the process rather than the person")
}
