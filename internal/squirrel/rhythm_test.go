package squirrel_test

import (
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func chore(id int64, name string, every, since int, everDone bool) squirrel.Chore {
	return squirrel.Chore{ID: id, Name: name, EveryDays: every, SinceDays: since, EverDone: everDone, Active: true}
}

var sundayMorning = time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC)

func TestARhythmIsDailyWeeklyOrSeldomAndNothingElse(t *testing.T) {
	for _, one := range []struct {
		every int
		want  squirrel.Rhythm
	}{{1, squirrel.Daily}, {2, squirrel.Daily}, {3, squirrel.Weekly}, {7, squirrel.Weekly},
		{14, squirrel.Weekly}, {15, squirrel.Seldom}, {30, squirrel.Seldom}, {365, squirrel.Seldom}} {
		require.Equal(t, one.want, squirrel.RhythmOf(one.every), "every %d days", one.every)
	}
}

func TestSquirrelWillNotClaimAUsualTimeFromTooLittleEvidence(t *testing.T) {
	two := []time.Time{sundayMorning, sundayMorning.AddDate(0, 0, -7)}
	require.False(t, squirrel.UsuallyFrom(two).Known(),
		"it told you when you usually do a thing you have done twice")

	three := append(two, sundayMorning.AddDate(0, 0, -14))
	got := squirrel.UsuallyFrom(three)
	require.True(t, got.Known())
	require.Equal(t, time.Sunday, got.Weekday)
	require.Equal(t, squirrel.Morning, got.Part)
	require.Equal(t, "you usually do this Sunday morning", got.Words())
}

func TestAThingYouDoEveryDayHasNoUsualDay(t *testing.T) {
	every := []time.Time{}
	for back := 0; back < 12; back++ {
		every = append(every, sundayMorning.AddDate(0, 0, -back))
	}

	got := squirrel.UsuallyFrom(every)
	require.False(t, got.OnADay,
		"it named a weekday for a thing you do on all of them, which is the loudest way to be wrong")
	require.Equal(t, squirrel.Morning, got.Part, "the hour is still yours, even when the day is not")
	require.Equal(t, "you usually do this in the morning", got.Words())
}

func TestNoUsualDayWhenTwoDaysShareIt(t *testing.T) {
	split := []time.Time{}
	for week := 0; week < 3; week++ {
		split = append(split,
			sundayMorning.AddDate(0, 0, -7*week),
			sundayMorning.AddDate(0, 0, -7*week-1))
	}

	got := squirrel.UsuallyFrom(split)
	require.False(t, got.OnADay,
		"Saturday and Sunday held three each and it picked one, which is a tie-break wearing a claim's clothes")
}

func TestAUsualDayIsClaimedOnAMajorityAndNotAPlurality(t *testing.T) {
	// Four Sundays, three Tuesdays, two Fridays. Sunday leads and is nowhere
	// near half, so there is no day you usually do this.
	var when []time.Time
	for i := 0; i < 4; i++ {
		when = append(when, sundayMorning.AddDate(0, 0, -7*i))
	}
	for i := 0; i < 3; i++ {
		when = append(when, sundayMorning.AddDate(0, 0, -7*i+2))
	}
	for i := 0; i < 2; i++ {
		when = append(when, sundayMorning.AddDate(0, 0, -7*i+5))
	}

	require.False(t, squirrel.UsuallyFrom(when).OnADay,
		"four out of nine is the commonest day, and it is not a day you usually do this")
}

func TestAThingWithNoUsualDayIsNeverRankedAsIfTodayWereIt(t *testing.T) {
	daily := squirrel.Usually{Part: squirrel.Morning}
	racks := squirrel.RacksOf(
		[]squirrel.Chore{chore(1, "water the plants", 1, 0, true)},
		map[int64]squirrel.Usually{1: daily}, sundayMorning, false)

	rack := rackFor(t, racks, squirrel.Daily)
	require.Len(t, rack.Waiting, 1)
	require.NotContains(t, rack.Waiting[0].Because, "the day you usually do it",
		"it has no usual day, so today cannot be it")
}

func TestTheOrderIsDueAndNowFirstThenDueThenTheDayYouUsuallyDoIt(t *testing.T) {
	usually := map[int64]squirrel.Usually{
		1: {Weekday: time.Sunday, Part: squirrel.Morning, OnADay: true},
		2: {Weekday: time.Sunday, Part: squirrel.Evening, OnADay: true},
		4: {Weekday: time.Sunday, Part: squirrel.Morning, OnADay: true},
	}
	chores := []squirrel.Chore{
		chore(3, "due, no usual time", 7, 9, true),
		chore(2, "due, usual time later today", 7, 8, true),
		chore(4, "not due, but today is the day", 7, 3, true),
		chore(1, "due, and now is when", 7, 7, true),
		chore(5, "not due", 7, 1, true),
	}

	racks := squirrel.RacksOf(chores, usually, sundayMorning, false)
	weekly := rackFor(t, racks, squirrel.Weekly)

	var order []string
	for _, s := range weekly.Waiting {
		order = append(order, s.Chore.Name)
	}
	require.Equal(t, []string{
		"due, and now is when",
		"due, usual time later today",
		"due, no usual time",
		"not due, but today is the day",
		"not due",
	}, order)
}

func TestSquirrelSaysWhyAThingIsWhereItIs(t *testing.T) {
	usually := map[int64]squirrel.Usually{1: {Weekday: time.Sunday, Part: squirrel.Morning, OnADay: true}}
	racks := squirrel.RacksOf([]squirrel.Chore{chore(1, "bins out", 7, 7, true)}, usually, sundayMorning, false)

	weekly := rackFor(t, racks, squirrel.Weekly)
	require.Len(t, weekly.Waiting, 1)
	require.Equal(t, "it comes back today, and this is when you usually do it", weekly.Waiting[0].Because,
		"the order cannot be read, so it is a ranking rather than a reason")
}

func TestEveryRowThatIsDueOrNeverStartedCanSayWhyItIsThere(t *testing.T) {
	chores := []squirrel.Chore{
		chore(1, "bins out", 7, 7, true),
		chore(3, "descale the kettle", 90, 0, false),
		chore(4, "wipe the sills", 30, 40, true),
	}
	racks := squirrel.RacksOf(chores, map[int64]squirrel.Usually{}, sundayMorning, false)

	rows := 0
	for _, rack := range racks {
		for _, s := range rack.Waiting {
			rows++
			require.NotEmpty(t, s.Because,
				"%q sits somewhere in the %s rack and cannot say why", s.Chore.Name, rack.Rhythm)
		}
	}
	require.Equal(t, len(chores), rows, "a chore went missing between the store and the racks")
}

// A thing whose turn is simply not today says nothing at all, and above all
// does not say how long it has been waiting. The rack has refused to print
// that since it was built.
func TestAThingWhoseTurnHasNotComeSaysNothingAboutHowLongItHasWaited(t *testing.T) {
	racks := squirrel.RacksOf(
		[]squirrel.Chore{chore(2, "water the plants", 7, 2, true)},
		map[int64]squirrel.Usually{}, sundayMorning, false)

	weekly := rackFor(t, racks, squirrel.Weekly)
	require.Len(t, weekly.Waiting, 1)
	require.Empty(t, weekly.Waiting[0].Because,
		"it told you how long a thing has been sitting there, which is a sentence about you")
}

func TestAChoreNeverDoneIsNotTreatedAsOverdue(t *testing.T) {
	chores := []squirrel.Chore{
		chore(1, "never done, long past its interval", 7, 400, false),
		chore(2, "done before, due today", 7, 7, true),
	}
	racks := squirrel.RacksOf(chores, nil, sundayMorning, false)
	weekly := rackFor(t, racks, squirrel.Weekly)

	require.Equal(t, "done before, due today", weekly.Waiting[0].Chore.Name,
		"a chore nobody has ever done was ranked as the most overdue thing you have")
}

func TestOnAWipedDayTheRacksShowOnlyWhatComesBackToday(t *testing.T) {
	chores := []squirrel.Chore{
		chore(1, "due", 7, 7, true),
		chore(2, "not due", 7, 2, true),
		chore(3, "not due either", 7, 1, true),
	}

	racks := squirrel.RacksOf(chores, nil, sundayMorning, true)
	weekly := rackFor(t, racks, squirrel.Weekly)

	require.Len(t, weekly.Waiting, 1, "the board asked as much of you on a wiped day as on any other")
	require.Equal(t, "due", weekly.Waiting[0].Chore.Name)
	require.Equal(t, 2, weekly.Resting, "what is resting is not counted, so the rack cannot say it is quiet")
}

func TestNothingIsCountedThatYouDidNotDo(t *testing.T) {
	chores := []squirrel.Chore{chore(1, "due", 7, 30, true), chore(2, "also due", 7, 40, true)}

	racks := squirrel.RacksOf(chores, nil, sundayMorning, false)
	weekly := rackFor(t, racks, squirrel.Weekly)

	require.Zero(t, weekly.Resting,
		"a resting count appeared on an ordinary day, where it can only read as a backlog")
	for _, s := range weekly.Waiting {
		require.NotContains(t, s.Because, "late")
		require.NotContains(t, s.Because, "overdue")
		require.NotContains(t, s.Because, "behind")
	}
}

func rackFor(t *testing.T, racks []squirrel.Rack, want squirrel.Rhythm) squirrel.Rack {
	t.Helper()
	for _, r := range racks {
		if r.Rhythm == want {
			return r
		}
	}
	t.Fatalf("no %s rack", want)
	return squirrel.Rack{}
}

func TestNowHoldsOnlyWhatHasSomethingToSayForItselfToday(t *testing.T) {
	usually := map[int64]squirrel.Usually{
		1: {Weekday: time.Sunday, OnADay: true, Part: squirrel.Morning},
		4: {Weekday: time.Sunday, OnADay: true, Part: squirrel.Morning},
	}
	chores := []squirrel.Chore{
		chore(1, "bins out", 7, 7, true),
		chore(2, "water the plants", 1, 1, true),
		chore(3, "wipe the sills", 30, 4, true),
		chore(4, "change the bed", 7, 2, true),
		chore(5, "descale the kettle", 90, 0, false),
	}

	now := squirrel.Now(squirrel.RacksOf(chores, usually, sundayMorning, false))

	names := []string{}
	for _, s := range now {
		names = append(names, s.Chore.Name)
	}
	require.Equal(t, []string{"bins out", "water the plants", "change the bed"}, names,
		"the phone is showing you your whole life again, which is the thing that makes it invisible")
}

func TestNowCrossesTheRacksRatherThanReadingThemInTurn(t *testing.T) {
	usually := map[int64]squirrel.Usually{
		1: {Weekday: time.Sunday, OnADay: true, Part: squirrel.Morning},
	}
	chores := []squirrel.Chore{
		chore(2, "water the plants", 1, 3, true),
		chore(1, "service the boiler", 90, 90, true),
	}

	now := squirrel.Now(squirrel.RacksOf(chores, usually, sundayMorning, false))

	require.Len(t, now, 2)
	require.Equal(t, "service the boiler", now[0].Chore.Name,
		"a seldom thing whose day this is came second to a daily one, so the order is the rack rather than the rank")
}
