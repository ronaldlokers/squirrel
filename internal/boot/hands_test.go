//go:build integration

package boot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Every write here calls a function the screen's own buttons already call.
// What is being tested is that it is the same write, and that it is only ever
// yours.

func handsFor(t *testing.T, store *squirrel.Store, at time.Time) *hands {
	t.Helper()
	h := handsOver(store)
	h.now = func() time.Time { return at }
	return h
}

func TestCompletingATaskIsTheSameWriteTheCardMakes(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	id := taskFor(t, store, p, "ring the vet")

	what, err := handsFor(t, store, now).Complete(ctx, p, id)
	require.NoError(t, err)
	require.Equal(t, "ring the vet", what)

	it, found, err := store.ItemByID(ctx, p, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.ItemDone, it.State)
}

// Only ever yours. A model asking about an id that is not yours gets nothing,
// and the store's own scoping is what guarantees it.
func TestCompletingSomethingThatIsNotYoursDoesNothing(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	mine := factsOwner(t, store)

	id := taskFor(t, store, mine, "ring the vet")

	_, err := handsFor(t, store, time.Now()).Complete(ctx, mine+1000, id)
	require.ErrorIs(t, err, errNotYours)

	it, _, err := store.ItemByID(ctx, mine, id)
	require.NoError(t, err)
	require.Equal(t, squirrel.ItemOpen, it.State)
}

// The same row `!did` and the chores screen's button write, so a retraction
// works on it exactly as it works on either of those.
func TestCompletingAChoreRecordsAnOrdinaryCompletion(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	overdueChore(t, store, p, "put the bins out")
	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1)

	what, err := handsFor(t, store, now).CompleteChore(ctx, p, chores[0].ID)
	require.NoError(t, err)
	require.Equal(t, "put the bins out", what)

	done, err := store.CompletedToday(ctx, p, squirrel.StartOfDay(now))
	require.NoError(t, err)
	require.Equal(t, []string{"put the bins out"}, done)
}

func TestActingOnAChoreThatIsNotYoursDoesNothing(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	h := handsFor(t, store, time.Now())

	_, err := h.CompleteChore(ctx, p, 999)
	require.ErrorIs(t, err, errNotYours)

	_, err = h.SnoozeChore(ctx, p, 999, 4)
	require.ErrorIs(t, err, errNotYours)
}

func TestStartingATimerStartsTheOneInTheLid(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	require.NoError(t, handsFor(t, store, now).StartTimer(ctx, p, "the kitchen", 15))

	timer, found, err := store.CurrentTimer(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the kitchen", timer.Label)
}

// A snooze is bounded in code, at both ends. A model asked not to silence
// something for a year is a model that can, and one asked for zero hours is a
// model that can hand back a deadline already in the past.
func TestSnoozingIsBounded(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	overdueChore(t, store, p, "put the bins out")
	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)

	_, err = handsFor(t, store, now).SnoozeChore(ctx, p, chores[0].ID, 24*365)
	require.NoError(t, err)

	// Two weeks out, not a year: it is quiet now, and back inside the ceiling.
	due, err := store.DueChores(ctx, p, now.Add(15*24*time.Hour))
	require.NoError(t, err)
	require.NotEmpty(t, due, "a chore was silenced past the ceiling")
}

// And the floor. Zero hours, or a negative number, is a snooze that expires
// before it is written — the chore comes straight back, so the press did
// nothing and there is nothing on the screen to say why.
func TestSnoozingForNoTimeIsStillASnooze(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	for _, hours := range []int{0, -5} {
		overdueChore(t, store, p, "put the bins out")
		chores, err := store.ActiveChores(ctx, p)
		require.NoError(t, err)
		require.NotEmpty(t, chores)

		_, err = handsFor(t, store, now).SnoozeChore(ctx, p, chores[0].ID, hours)
		require.NoError(t, err, "%d hours", hours)

		due, err := store.DueChores(ctx, p, now)
		require.NoError(t, err)
		require.Empty(t, due, "%d hours left the chore due immediately", hours)
	}
}

// Refusing takes the vocabulary the picker itself uses, checked rather than
// trusted: a kind that was never offered is a value nobody pressed.
func TestRefusingOnlyTakesTheKindsThePickerOffers(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	id := taskFor(t, store, p, "ring the vet")
	h := handsFor(t, store, now)

	require.ErrorIs(t, h.Refuse(ctx, p, "checkin", id), errNotYours)
	require.NoError(t, h.Refuse(ctx, p, "task", id))

	suppressed, err := store.Suppressed(ctx, p, squirrel.StartOfDay(now))
	require.NoError(t, err)
	require.True(t, suppressed[squirrel.SuppressionKey(squirrel.OfferTask, id)])
}

// A task the coach recorded is an ordinary task. Nothing marks it as a
// machine's doing, because it is a thing you said out loud and it was written
// down where you would have written it.
func TestATaskTheCoachRecordedIsAnOrdinaryTask(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	require.NoError(t, handsFor(t, store, now).CreateTask(ctx, p, "book the MOT"))

	tasks, _, err := store.Tasks(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "book the MOT", tasks[0].RawText)
	require.Equal(t, squirrel.ItemTask, tasks[0].Kind)
	require.Equal(t, squirrel.ItemOpen, tasks[0].State)
}
