//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The two floors under the nudge. Both only ever quieten, and neither can be
// lifted by anything — including the model, which is asked afterwards or not
// at all.

func TestNothingIsRaisedInTheNight(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")

	for _, hour := range []int{22, 23, 0, 3, 5} {
		chat, sent := chatRecorder("1")
		s := schedulerWithChat(t, store, p, chat)

		require.NoError(t, s.Nudge(ctx, today(t, hour, 0, 0), squirrel.NudgeFromArrival))
		require.Empty(t, *sent, "a nudge went out at %02d:00", hour)
	}
}

func TestTheDayStartsAtSix(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	chat, sent := chatRecorder("1")

	require.NoError(t, schedulerWithChat(t, store, p, chat).
		Nudge(ctx, today(t, 6, 0, 0), squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
}

// The night costs nothing, not even the day's nudge. A quiet hour that spent
// the slot would mean a ping at 03:00 silenced the whole of the next day.
func TestTheNightDoesNotSpendTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")

	night, nothing := chatRecorder("1")
	require.NoError(t, schedulerWithChat(t, store, p, night).
		Nudge(ctx, today(t, 3, 0, 0), squirrel.NudgeFromArrival))
	require.Empty(t, *nothing)

	morning, sent := chatRecorder("2")
	require.NoError(t, schedulerWithChat(t, store, p, morning).
		Nudge(ctx, today(t, 10, 0, 0), squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
}

// A chore raised on a wiped day is the product asking for something on the day
// you have least to give. The chore's own clock keeps running, so nothing is
// lost by waiting.
func TestNoNudgeOnALowDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	now := today(t, 10, 0, 0)
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "test", now))

	chat, sent := chatRecorder("1")
	require.NoError(t, schedulerWithChat(t, store, p, chat).
		Nudge(ctx, now, squirrel.NudgeFromArrival))
	require.Empty(t, *sent)
}

// Flat but functional is not a low day. CapacityOf draws that line and this
// path reads the same answer as the picker rather than a second opinion.
func TestAFlatDayStillNudges(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	now := today(t, 10, 0, 0)
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodLow, "test", now))

	chat, sent := chatRecorder("1")
	require.NoError(t, schedulerWithChat(t, store, p, chat).
		Nudge(ctx, now, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
}

// A reading from this morning is not now. The same six-hour staleness the
// picker already honours, and it comes from reading CapacityOf rather than the
// mood row.
func TestAStaleLowReadingDoesNotSilenceTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	now := today(t, 16, 0, 0)
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "test", now.Add(-8*time.Hour)))

	chat, sent := chatRecorder("1")
	require.NoError(t, schedulerWithChat(t, store, p, chat).
		Nudge(ctx, now, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
}

// Quiet hours are about what arrives unasked. The evening message is a
// once-a-day thing at an hour that was chosen — setting that hour to 22:30
// must not quietly cost it its chore line.
func TestTheEveningMessageKeepsItsChoreLateAtNight(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	chat, sent := chatRecorder("1")
	s := squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, Chat: chat, PersonID: p, ConversationID: "9",
		At: 22*time.Hour + 30*time.Minute, Location: amsterdam(t),
		OnError: func(error) {},
	})

	require.NoError(t, s.Once(ctx, today(t, 22, 35, 0)))
	require.Len(t, *sent, 1)
	require.Contains(t, (*sent)[0].message.Text, "vacuum")
}

// But a low day still costs it that line, because that gate is about how much
// you have to give rather than about the hour.
func TestTheEveningMessageDropsItsChoreOnALowDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	now := today(t, 19, 5, 0)
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "test", now))

	chat, sent := chatRecorder("1")
	s := squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, Chat: chat, PersonID: p, ConversationID: "9",
		At: 19 * time.Hour, Location: amsterdam(t),
		OnError: func(error) {},
	})

	require.NoError(t, s.Once(ctx, now))
	for _, m := range *sent {
		require.NotContains(t, m.message.Text, "vacuum")
	}
}

// The model is asked after both floors, never before. A held-back hour or a
// low day costs nothing at all, which is the arithmetic the design rests on.
func TestTheModelIsNotAskedBelowEitherFloor(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")

	night := &heldBack{allow: true}
	chat, _ := chatRecorder("1")
	require.NoError(t, schedulerHolding(t, store, p, chat, night).
		Nudge(ctx, today(t, 3, 0, 0), squirrel.NudgeFromArrival))
	require.Empty(t, night.asked, "the model was asked in the middle of the night")

	now := today(t, 10, 0, 0)
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "test", now))

	low := &heldBack{allow: true}
	quiet, _ := chatRecorder("2")
	require.NoError(t, schedulerHolding(t, store, p, quiet, low).
		Nudge(ctx, now, squirrel.NudgeFromArrival))
	require.Empty(t, low.asked, "the model was asked on a low day")
}
