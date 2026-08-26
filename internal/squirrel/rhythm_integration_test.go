//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestAWeekdayChoreIsDueOnItsDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "the bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 1))

	// The chore was created now; walk the next fortnight day by day.
	start := time.Now()
	var dueOn []time.Weekday
	for i := 1; i <= 14; i++ {
		at := start.AddDate(0, 0, i)
		due, err := store.DueChores(ctx, p, at)
		require.NoError(t, err)
		if len(due) > 0 {
			dueOn = append(dueOn, at.Weekday())
		}
	}
	require.NotEmpty(t, dueOn, "a weekly thursday chore never came due")
	for _, d := range dueOn {
		require.Equal(t, time.Thursday, d, "it came due on a %s", d)
	}
}

func TestAnAlternatingChoreSkipsAWeek(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "the bins", 14*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 2))

	start := time.Now()
	thursdays := 0
	due := 0
	for i := 1; i <= 28; i++ {
		at := start.AddDate(0, 0, i)
		if at.Weekday() != time.Thursday {
			continue
		}
		thursdays++
		got, err := store.DueChores(ctx, p, at)
		require.NoError(t, err)
		if len(got) > 0 {
			due++
		}
	}
	require.Equal(t, 4, thursdays)
	require.Equal(t, 2, due, "an alternating chore came due on %d of 4 thursdays", due)
}

// Doing it does not move the day. An interval measured from the last
// completion slides: do the bins a day late once and every reminder after it is
// a day late too. That is the failure this feature exists to prevent.
func TestDoingItLateDoesNotMoveTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "the bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 1))

	start := time.Now()
	firstThursday := next(start, time.Thursday)
	// Done on the friday, a day late.
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "screen", firstThursday.AddDate(0, 0, 1)))

	// The following thursday is still the day.
	nextThursday := firstThursday.AddDate(0, 0, 7)
	got, err := store.DueChores(ctx, p, nextThursday)
	require.NoError(t, err)
	require.Len(t, got, 1, "doing it late moved the day it comes back")

	// And the saturday after that late completion is not.
	got, err = store.DueChores(ctx, p, firstThursday.AddDate(0, 0, 2))
	require.NoError(t, err)
	require.Empty(t, got)
}

// Done today means not due again today, however many times the digest runs.
func TestAWeekdayChoreDoneTodayIsNotDueAgainToday(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "the bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 1))

	thursday := next(time.Now(), time.Thursday)
	got, err := store.DueChores(ctx, p, thursday)
	require.NoError(t, err)
	require.Len(t, got, 1)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "screen", thursday))

	got, err = store.DueChores(ctx, p, thursday.Add(2*time.Hour))
	require.NoError(t, err)
	require.Empty(t, got, "it came back the same day it was done")
}

// The old rhythm is untouched. Every chore that exists keeps its interval.
func TestAnIntervalChoreIsUnaffected(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "water the plants", 3*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, c.OnADay())

	got, err := store.DueChores(ctx, p, time.Now().Add(4*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1, "an ordinary interval chore stopped coming due")
	require.False(t, got[0].OnADay())
}

// The rhythm is readable back, so a screen can open the question on what is
// true rather than on a blank form.
func TestTheRhythmIsReadBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "the bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 2))

	all, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.True(t, all[0].OnADay())
	require.Equal(t, time.Thursday, all[0].Weekday)
	require.Equal(t, 2, all[0].Weeks)
	// The interval is written too, so everything that renders "how often"
	// keeps working without knowing about days.
	require.Equal(t, 14*24*time.Hour, all[0].Every)
}

// And it can be put back on an interval without being recreated.
func TestARhythmCanBeCleared(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "the bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 1))
	require.NoError(t, store.SetChoreRhythm(ctx, p, c.ID, 0, 0))

	all, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.False(t, all[0].OnADay())
}

// Somebody else's chore is not yours to reschedule.
func TestARhythmBelongsToOnePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-theirs", "theirs")
	require.NoError(t, err)

	c, err := store.UpsertChore(ctx, theirs, "their bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	require.NoError(t, store.SetChoreRhythm(ctx, mine, c.ID, time.Thursday, 2))

	all, err := store.ActiveChores(ctx, theirs)
	require.NoError(t, err)
	require.False(t, all[0].OnADay(), "somebody else rescheduled their chore")
}

func TestARhythmThisProductDoesNotHaveIsRefused(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	c, err := store.UpsertChore(ctx, p, "the bins", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	require.Error(t, store.SetChoreRhythm(ctx, p, c.ID, time.Thursday, 3))
	require.Error(t, store.SetChoreRhythm(ctx, p, c.ID, time.Weekday(9), 1))
}

func TestDayNamed(t *testing.T) {
	for _, said := range []string{"thursday", "Thursday", "thu", " THU "} {
		d, ok := squirrel.DayNamed(said)
		require.True(t, ok, said)
		require.Equal(t, time.Thursday, d, said)
	}
	for _, said := range []string{"", "th", "someday", "tomorrow"} {
		_, ok := squirrel.DayNamed(said)
		require.False(t, ok, said)
	}
}

// next is the coming day of the week, at midday so a test never straddles a
// boundary by an hour.
func next(from time.Time, want time.Weekday) time.Time {
	d := time.Date(from.Year(), from.Month(), from.Day(), 12, 0, 0, 0, from.Location())
	for i := 1; i <= 7; i++ {
		if got := d.AddDate(0, 0, i); got.Weekday() == want {
			return got
		}
	}
	return d
}
