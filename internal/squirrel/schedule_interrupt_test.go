//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The veto. What these tests are about is the direction it can fail in, and
// the fact that it cannot cause an interruption — only prevent one.

type heldBack struct {
	asked []string
	say   string
	allow bool
}

func (h *heldBack) fn(_ context.Context, _ int64, about string, _ time.Time) (string, bool) {
	h.asked = append(h.asked, about)
	return h.say, h.allow
}

func schedulerHolding(t *testing.T, store *squirrel.Store, personID int64,
	chat squirrel.Chat, hold *heldBack) *squirrel.Scheduler {

	t.Helper()
	opts := squirrel.SchedulerOptions{
		Store: store, Chat: chat, PersonID: personID, ConversationID: "9",
		At: 19 * time.Hour, Location: amsterdam(t),
		OnError: func(error) {},
	}
	if hold != nil {
		opts.Interrupt = hold.fn
	}
	return squirrel.NewScheduler(opts)
}

func aDueChore(t *testing.T, store *squirrel.Store, personID int64, name string) {
	t.Helper()
	c, err := store.UpsertChore(context.Background(), personID, name, twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)
}

func TestAHeldBackNudgeSaysNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	chat, sent := chatRecorder("1")
	hold := &heldBack{allow: false}

	require.NoError(t, schedulerHolding(t, store, p, chat, hold).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))

	require.Empty(t, *sent)
	require.Equal(t, []string{"vacuum"}, hold.asked)
}

// The ordering that matters. RecordPrompt spends the day's one nudge in a
// unique index; a decision to stay quiet taken after that would spend the day
// on a message nobody received, and every later trigger — including the
// evening fallback — would be refused by it.
func TestBeingHeldBackDoesNotSpendTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")

	quiet, nothing := chatRecorder("1")
	require.NoError(t, schedulerHolding(t, store, p, quiet, &heldBack{allow: false}).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))
	require.Empty(t, *nothing)

	// A later trigger the same day still works, because nothing was claimed.
	chat, sent := chatRecorder("2")
	require.NoError(t, schedulerHolding(t, store, p, chat, &heldBack{allow: true}).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
	require.Contains(t, (*sent)[0].message.Text, "vacuum")
}

// It may change the words. It may not change the buttons: what may be said is
// a matter of words, and what may be done is not.
func TestSuppliedWordingReplacesTheTextAndNothingElse(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	chat, sent := chatRecorder("1")
	hold := &heldBack{allow: true, say: "The hallway is getting bad."}

	require.NoError(t, schedulerHolding(t, store, p, chat, hold).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))

	require.Len(t, *sent, 1)
	require.Equal(t, "The hallway is getting bad.", (*sent)[0].message.Text)
	// Done and "not today", exactly as without a coach.
	require.Len(t, (*sent)[0].message.Actions, 2)
}

// Empty wording means the fixed message stands, which is what it was before
// any of this existed.
func TestAgreeingWithoutWordingLeavesTheNudgeAsItWas(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	chat, sent := chatRecorder("1")

	require.NoError(t, schedulerHolding(t, store, p, chat, &heldBack{allow: true}).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))

	require.Len(t, *sent, 1)
	require.Contains(t, (*sent)[0].message.Text, "vacuum")
}

// With nothing wired, the nudge path is exactly what it was.
func TestWithNoInterrupterTheNudgeIsUnchanged(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")
	chat, sent := chatRecorder("1")

	require.NoError(t, schedulerHolding(t, store, p, chat, nil).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
}

// It is never asked about something the rules did not choose. Nothing due
// means no call at all — which is the arithmetic the whole design rests on:
// 1,435 of 1,440 ticks a day end before this line.
func TestItIsNeverAskedWhenNothingIsDue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	chat, sent := chatRecorder("1")
	hold := &heldBack{allow: true}

	require.NoError(t, schedulerHolding(t, store, p, chat, hold).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))

	require.Empty(t, *sent)
	require.Empty(t, hold.asked, "it was asked about a nudge the rules never made")
}

// And never about one the day's budget already refused.
func TestItIsNotAskedTwiceInADay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	aDueChore(t, store, p, "vacuum")

	first, _ := chatRecorder("1")
	require.NoError(t, schedulerHolding(t, store, p, first, &heldBack{allow: true}).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))

	second, sent := chatRecorder("2")
	hold := &heldBack{allow: true}
	require.NoError(t, schedulerHolding(t, store, p, second, hold).
		Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))

	require.Empty(t, *sent)
	// And it was not even asked. A chore that has already been raised today
	// stops being due, so the second trigger ends before the model is reached
	// rather than at the unique index — which means the day's second trigger
	// costs nothing at all.
	require.Empty(t, hold.asked, "the day's second trigger reached the model")
}
