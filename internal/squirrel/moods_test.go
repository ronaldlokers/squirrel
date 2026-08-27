//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// How you have been, when you ask — and only when you ask.

func TestMoodsAreReadBackWhenAskedFor(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat", now.AddDate(0, 0, -1)))
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodGood, "chat", now))

	reply := triage(t, store, p, "!moods")
	require.Contains(t, reply, "today")
	require.Contains(t, reply, "yesterday")
	require.Contains(t, reply, squirrel.Words[squirrel.MoodGood])
	require.Contains(t, reply, squirrel.Words[squirrel.MoodWiped])
}

// The readings and their days, and nothing else. No average, no streak, no
// count — the interpretation is yours.
func TestMoodsSayNothingAboutWhatTheyMean(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	for i := 0; i < 4; i++ {
		require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat",
			now.AddDate(0, 0, -i)))
	}

	reply := triage(t, store, p, "!moods")
	for _, judgement := range []string{
		"average", "streak", "4 days", "in a row", "trend", "mostly", "worse", "better",
	} {
		require.NotContains(t, reply, judgement)
	}
}

// Two answers on one day is a day you checked in twice, not two facts about
// you, so they share a line.
func TestTwoReadingsInADayShareTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	// Midday, and not time.Now(): two readings two hours apart are only on the
	// same day if the first is not within two hours of midnight. This failed
	// once, at 00:22, and would have failed every night between midnight and
	// two. The date still has to be a real one — the reply says "today".
	y, m, d := time.Now().Date()
	now := time.Date(y, m, d, 12, 0, 0, 0, time.Local)

	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodLow, "chat", now.Add(-2*time.Hour)))
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodGood, "chat", now))

	reply := triage(t, store, p, "!moods")
	require.Equal(t, 1, countLines(reply), "one day, one line")
}

func countLines(s string) int {
	n := 1
	for _, r := range s {
		if r == '\n' {
			n++
		}
	}
	return n
}

func TestNoReadingsSaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	require.Contains(t, triage(t, store, p, "!moods"), "not said how you are")
}

func TestOlderThanAFortnightIsNotRead(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodFrazzled, "chat", now.AddDate(0, 0, -20)))
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodGood, "chat", now))

	readings, err := store.CheckinsSince(ctx, p, squirrel.MoodWindowStart(now))
	require.NoError(t, err)
	require.Len(t, readings, 1)
	require.Equal(t, squirrel.MoodGood, readings[0].Mood)
}

// Nothing reads them on its own. The rule is no longer enforced by the absence
// of a function, so it is enforced by there being exactly the callers that ask.
func TestNothingElseReadsTheReadingsBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	for i := 0; i < 3; i++ {
		require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat",
			now.AddDate(0, 0, -i)))
	}

	// The picker is handed one reading, derived, and no history.
	require.Equal(t, squirrel.CapacityLow, store.Capacity(ctx, p, now))

	// And the evening message says nothing about how you have been.
	evening := squirrel.EveningMessage(squirrel.Handled{}, []string{"a thought"}, nil, "")
	for _, mood := range squirrel.Moods {
		require.NotContains(t, evening.Text, squirrel.Words[mood])
	}
}
