//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The picker is the one function whose wrongness is most visible, so the rules
// are exercised in order and the gate is exercised in both directions.

func TestPickNowOffersNothingWhenThereIsNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	_, found, err := store.PickNow(context.Background(), p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found, "nothing to hand you is a normal answer")
}

// Chores belong to the racks, which order and explain the whole of them. An
// offer naming one could only repeat the top of a rack already on the screen.
func TestThePickerLeavesChoresToTheRack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "bins out", oneDay, oneDay)
	require.NoError(t, err)
	backdate(t, store, "bins out", 3)

	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found, "the chore is no less due; the picker is no longer where it is answered")
}

// The gate is not what is doing this. A chore is left to the rack on a day
// with every bit of capacity in it, and when the gate is lifted outright.
func TestNoChoreIsOfferedEvenWithTheGateLifted(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "bins out", oneDay, oneDay)
	require.NoError(t, err)
	backdate(t, store, "bins out", 3)

	_, found, err := store.PickNow(ctx, p, time.Now(), true)
	require.NoError(t, err)
	require.False(t, found)
}

// Rule 4, and what a due chore no longer does to it: the thing you decided on
// is handed back whole rather than queued behind a rhythm.
func TestADueChoreDoesNotStandInFrontOfWhatYouDecided(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	_, err := store.UpsertChore(ctx, p, "bins out", oneDay, oneDay)
	require.NoError(t, err)
	backdate(t, store, "bins out", 3)

	o, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.OfferTask, o.Kind)
	require.Equal(t, "ring the vet", o.Text)
}

// The oldest decision, not the newest — every list here is newest-first
// because a list is read, and this is acted on.
func TestPickNowOffersTheOldestTask(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	taskOf(t, store, p, "book the car in")

	o, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.OfferTask, o.Kind)
	require.Equal(t, "ring the vet", o.Text)
}

func TestRefusingSuppressesForTheRestOfTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	taskOf(t, store, p, "book the car in")

	first, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NoError(t, store.Refuse(ctx, p, first.Kind, first.RefID, time.Now()))

	second, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.NotEqual(t, first.RefID, second.RefID, "a refusal is honoured")
	require.Equal(t, "book the car in", second.Text)
}

// Every transition reverses, and a refusal is a transition.
func TestARefusalCanBeTakenBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)

	require.NoError(t, store.Refuse(ctx, p, o.Kind, o.RefID, time.Now()))
	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found)

	require.NoError(t, store.UnrefuseToday(ctx, p, o.Kind, o.RefID, time.Now()))
	again, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, o.RefID, again.RefID)
}

// The gate. A fresh capacity word drops everything Squirrel would raise on its
// own initiative, and nothing else.
func TestALowDayOffersNothingOfItsOwnInitiative(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "screen", time.Now()))

	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found)
}

func TestShowMeAnywayLiftsTheGate(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "screen", time.Now()))

	o, found, err := store.PickNow(ctx, p, time.Now(), true)
	require.NoError(t, err)
	require.True(t, found, "saying you are wiped must not be a wall")
	require.Equal(t, "ring the vet", o.Text)
}

// Never a count, on the newest surface, in the shape the deck's own test uses.
func TestAnOfferCarriesNoTotal(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for _, text := range []string{"one", "two", "three", "four"} {
		taskOf(t, store, p, text)
	}

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	m := squirrel.NowMessage(o)
	require.NotContains(t, m.Text, "4")
	require.NotContains(t, m.Text, "3 more")
}

// taskOf decides something outright, the way the tasks screen's own slot does.
func taskOf(t *testing.T, store *squirrel.Store, personID int64, text string) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &personID, RawText: text,
		Payload: []byte(squirrel.ScreenCapture), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)
	_, err = store.SetItemKind(ctx, personID, id, squirrel.ItemTask)
	require.NoError(t, err)
	return id
}
