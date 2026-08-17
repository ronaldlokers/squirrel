package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func overdue(id int64, name string, sinceDays, everyDays int) squirrel.Chore {
	return squirrel.Chore{
		ID: id, Name: name, SinceDays: sinceDays, EveryDays: everyDays,
		Every: time.Duration(everyDays) * 24 * time.Hour,
	}
}

func TestPickChoreOnNothingDue(t *testing.T) {
	_, ok := squirrel.PickChore(nil, 0.5)
	require.False(t, ok)
}

func TestPickChoreOnOneDue(t *testing.T) {
	c := overdue(1, "vacuum", 20, 14)
	got, ok := squirrel.PickChore([]squirrel.Chore{c}, 0.99)
	require.True(t, ok)
	require.Equal(t, c.ID, got.ID)
}

// Weight is the overdue ratio — days since, over the interval. A chore three
// intervals late is three times as likely as one just past due, and never
// certain. The draw walks the cumulative weights, so a low draw lands in the
// first bucket and a high draw in the last.
func TestPickChoreIsWeightedByHowOverdue(t *testing.T) {
	// ratios 1.0 and 3.0, so weights 1 and 3 out of 4.
	due := []squirrel.Chore{
		overdue(1, "just due", 14, 14),
		overdue(2, "very late", 42, 14),
	}

	first, ok := squirrel.PickChore(due, 0.0)
	require.True(t, ok)
	require.Equal(t, int64(1), first.ID, "the bottom of the range is the first bucket")

	boundary, ok := squirrel.PickChore(due, 0.24)
	require.True(t, ok)
	require.Equal(t, int64(1), boundary.ID, "still inside the 1/4 the first chore owns")

	second, ok := squirrel.PickChore(due, 0.26)
	require.True(t, ok)
	require.Equal(t, int64(2), second.ID, "past 1/4, the late one owns the rest")

	last, ok := squirrel.PickChore(due, 0.999)
	require.True(t, ok)
	require.Equal(t, int64(2), last.ID)
}

// The most overdue chore is the one you have been avoiding longest, and by
// definition the one nudging has already failed to shift. It must not be
// certain to win, or this is "most overdue" with extra steps.
func TestPickChoreCanNameSomethingOtherThanTheWorst(t *testing.T) {
	due := []squirrel.Chore{
		overdue(1, "mild", 15, 14),
		overdue(2, "dreadful", 140, 14),
	}
	got, ok := squirrel.PickChore(due, 0.0)
	require.True(t, ok)
	require.Equal(t, int64(1), got.ID, "the mild one is reachable even against a 10x ratio")
}

// A chore that is due but not yet past its interval still has to be pickable,
// or a zero weight would make it invisible forever.
func TestPickChoreGivesEveryDueChoreSomeChance(t *testing.T) {
	due := []squirrel.Chore{overdue(1, "barely", 0, 14), overdue(2, "late", 28, 14)}

	got, ok := squirrel.PickChore(due, 0.0)
	require.True(t, ok)
	require.Equal(t, int64(1), got.ID, "a zero-days-overdue chore is still reachable")
}

func TestNudgeMessageCarriesOneButton(t *testing.T) {
	m := squirrel.NudgeMessage(overdue(1, "bin day", 19, 14), squirrel.NudgeFromArrival)

	require.Contains(t, m.Text, "bin day")
	require.Contains(t, m.Text, "19 days")
	require.Contains(t, m.Text, "usually 14")
	require.Len(t, m.Actions, 1, "one chore, one button — never a list")
	require.Equal(t, "done:1", m.Actions[0].Value)
	require.Equal(t, "✅", m.Actions[0].Emoji)
	require.Equal(t, "single", m.SelectionMode)
}

// Riding back on a message the person sent, it acknowledges the piggyback.
// Varied phrasing is the same anti-habituation mechanism as varied timing.
func TestNudgeFramingVariesByReason(t *testing.T) {
	c := overdue(1, "bin day", 19, 14)

	piggyback := squirrel.NudgeMessage(c, squirrel.NudgeFromMessage)
	require.Contains(t, piggyback.Text, "While you're here")

	arrival := squirrel.NudgeMessage(c, squirrel.NudgeFromArrival)
	require.NotContains(t, arrival.Text, "While you're here")

	evening := squirrel.NudgeMessage(c, squirrel.NudgeFromEvening)
	require.NotEqual(t, arrival.Text, evening.Text, "each trigger reads differently")
}

// Facts only. The words that would make this a nag are the ones under test.
func TestNudgeNeverScolds(t *testing.T) {
	c := overdue(1, "vacuum", 200, 14)
	for _, why := range []squirrel.NudgeReason{
		squirrel.NudgeFromMessage, squirrel.NudgeFromArrival, squirrel.NudgeFromEvening,
	} {
		text := squirrel.NudgeMessage(c, why).Text
		for _, forbidden := range []string{
			"still", "again", "overdue", "finally", "haven't", "!", "should",
		} {
			require.NotContains(t, text, forbidden, "%s / %q", why, forbidden)
		}
	}
}
