//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Three states for a thing you cannot act on. What matters as much as the
// transitions is the property that needed no code: a new state is invisible to
// every existing list, because every list names the state it wants.

func TestHoldingTakesItOutOfEverySurface(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	id := taskOf(t, store, p, "ring the vet")

	ok, err := store.HoldItem(ctx, p, id, squirrel.ItemWaiting, "the vet", now)
	require.NoError(t, err)
	require.True(t, ok)

	tasks, _, err := store.Tasks(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, tasks, "a held task is still in the tasks list")

	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, items, "a held thing is still in the pile")

	// And the picker cannot hand it over, which is the whole point.
	_, found, err := store.PickNow(ctx, p, now, true)
	require.NoError(t, err)
	require.False(t, found, "the picker offered something you cannot act on")
}

func TestWhatYouAreWaitingOnIsKept(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	id := taskOf(t, store, p, "ring the vet")
	_, err := store.HoldItem(ctx, p, id, squirrel.ItemWaiting, "the vet", now)
	require.NoError(t, err)

	held, _, err := store.HeldItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, held, 1)
	require.Equal(t, "ring the vet", held[0].Text)
	require.Equal(t, squirrel.ItemWaiting, held[0].State)
	require.Equal(t, "waiting on the vet", held[0].Words())
}

// Someday is not waiting on anything, and says so rather than saying nothing.
func TestSomedayNeedsNoReason(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	id := taskOf(t, store, p, "learn to solder")
	_, err := store.HoldItem(ctx, p, id, squirrel.ItemSomeday, "", now)
	require.NoError(t, err)

	held, _, err := store.HeldItems(ctx, p, 20)
	require.NoError(t, err)
	require.Equal(t, "someday", held[0].Words())
}

// All three read together, because the question is "what is not moving" and
// splitting it into three lists would make you ask it three times.
func TestAllThreeComeBackTogether(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i, tc := range []struct {
		text  string
		state squirrel.ItemState
	}{
		{"ring the vet", squirrel.ItemWaiting},
		{"fix the boiler", squirrel.ItemBlocked},
		{"learn to solder", squirrel.ItemSomeday},
	} {
		id := taskOf(t, store, p, tc.text)
		_, err := store.HoldItem(ctx, p, id, tc.state, "", now.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	held, more, err := store.HeldItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, held, 3)
	require.False(t, more)
}

// The way back is the transition that already exists, and it clears the
// reason: "waiting on the vet" is not true of something you are now doing.
func TestPickingItBackUpClearsTheReason(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	id := taskOf(t, store, p, "ring the vet")
	_, err := store.HoldItem(ctx, p, id, squirrel.ItemWaiting, "the vet", now)
	require.NoError(t, err)

	ok, err := store.Unhold(ctx, p, id, now)
	require.NoError(t, err)
	require.True(t, ok)

	tasks, _, err := store.Tasks(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, tasks, 1, "it did not come back")

	held, _, err := store.HeldItems(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, held)

	// Held again with no reason: the old one must not survive.
	_, err = store.HoldItem(ctx, p, id, squirrel.ItemSomeday, "", now)
	require.NoError(t, err)
	held, _, err = store.HeldItems(ctx, p, 20)
	require.NoError(t, err)
	require.Equal(t, "someday", held[0].Words())
}

// Only something open. Holding what is already done, dropped or kept would be
// a transition nobody asked for over a fact somebody established.
func TestOnlyAnOpenThingCanBeSetAside(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	id := taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, now))

	ok, err := store.HoldItem(ctx, p, id, squirrel.ItemWaiting, "the vet", now)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestHoldingSomethingThatIsNotYoursDoesNothing(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	mine := owner(t, store)
	now := time.Now()

	id := taskOf(t, store, mine, "ring the vet")

	ok, err := store.HoldItem(ctx, mine+1000, id, squirrel.ItemWaiting, "the vet", now)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = store.Unhold(ctx, mine+1000, id, now)
	require.NoError(t, err)
	require.False(t, ok)
}

// The vocabulary is closed. A state that is not one of the three is refused
// here as well as by the column's own constraint — two doors into the same
// table, and both of them shut.
func TestOnlyTheThreeAreWaysToSetSomethingAside(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	id := taskOf(t, store, p, "ring the vet")

	_, err := store.HoldItem(ctx, p, id, squirrel.ItemDropped, "", time.Now())
	require.Error(t, err)

	require.True(t, squirrel.IsHeld(squirrel.ItemWaiting))
	require.False(t, squirrel.IsHeld(squirrel.ItemOpen))
	require.False(t, squirrel.IsHeld(squirrel.ItemKept))
}

// Generous about the wording, like the ladder's own four, because this arrives
// from someone who has just found out they cannot do the thing.
func TestTheThreeAreRecognisedTheWayYouWouldSayThem(t *testing.T) {
	for said, want := range map[string]squirrel.ItemState{
		"waiting":    squirrel.ItemWaiting,
		"waiting on": squirrel.ItemWaiting,
		"wait":       squirrel.ItemWaiting,
		"blocked":    squirrel.ItemBlocked,
		"blocked on": squirrel.ItemBlocked,
		"someday":    squirrel.ItemSomeday,
		"some day":   squirrel.ItemSomeday,
		"maybe":      squirrel.ItemSomeday,
		"one day":    squirrel.ItemSomeday,
	} {
		got, ok := squirrel.ParseHeld(said)
		require.True(t, ok, said)
		require.Equal(t, want, got, said)
	}

	for _, said := range []string{"", "  ", "done", "later", "nope"} {
		_, ok := squirrel.ParseHeld(said)
		require.False(t, ok, said)
	}
}

// Nothing here counts them. A number beside a list of stalled work is a
// reproach, and setting something aside was meant to stop it being asked about.
func TestTheListCapsWithoutSayingHowMuchMore(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	for i, text := range []string{"one", "two", "three", "four", "five"} {
		id := taskOf(t, store, p, text)
		_, err := store.HoldItem(ctx, p, id, squirrel.ItemSomeday, "",
			now.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	held, more, err := store.HeldItems(ctx, p, 2)
	require.NoError(t, err)
	require.Len(t, held, 2)
	require.True(t, more, "the cap did not report that there is more")
}
