//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// `!now` is the picker's chat half. Parity relaxed in one direction on 22
// August, and this stays: reading the pile back is on the floor chat keeps, so it
// has to be able to do what the screen's offer does — and its number has to
// resolve, or the buttons and the typed form mean different things.

func TestNowNamesOneThingAndOffersTwoButtons(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")

	chat, got := chatRecorder("451")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(context.Background(), itemOf("!now"), &p))
	require.Len(t, *got, 1)

	m := (*got)[0].message
	require.Contains(t, m.Text, "ring the vet")
	require.Len(t, m.Actions, 2, "one to do it, one to turn it down")
	require.Equal(t, "done:1", m.Actions[0].Value)
	require.Equal(t, "later:1", m.Actions[1].Value)
}

// The line it prints is a line you can answer by number, like every other
// numbered surface in this room.
func TestDoneOneAfterNowArchivesTheTask(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	id := taskOf(t, store, p, "ring the vet")
	triage(t, store, p, "!now")

	triage(t, store, p, "done 1")
	require.Equal(t, "done", stateOf(t, store, id))
}

// A tap is a state assertion. The picker is the first surface that can put a
// ✅ over something you decided to do, so that button has to mean what the
// tasks screen's own means — including in reverse.
func TestTappingDoneOnATaskOfferArchivesAndUntapsBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	id := taskOf(t, store, p, "ring the vet")
	chat, _ := chatRecorder("451")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	require.NoError(t, applier.Apply(ctx, itemOf("!now"), &p))

	require.NoError(t, applier.Apply(ctx, tapItem(p, "451", "done:1", true), &p))
	require.Equal(t, "done", stateOf(t, store, id))

	require.NoError(t, applier.Apply(ctx, tapItem(p, "451", "done:1", false), &p))
	require.Equal(t, "open", stateOf(t, store, id),
		"untapping returns it to the tasks, still decided")
}

// "Not now" costs one press, changes nothing about the thing, and is honoured
// by the next ask.
func TestTappingLaterRefusesForTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	id := taskOf(t, store, p, "ring the vet")
	chat, _ := chatRecorder("451")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	require.NoError(t, applier.Apply(ctx, itemOf("!now"), &p))

	require.NoError(t, applier.Apply(ctx, tapItem(p, "451", "later:1", true), &p))
	require.Equal(t, "open", stateOf(t, store, id), "refusing is not doing")

	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found)
}

// A chore turned down in the picker is exactly as due as it was. The picker's
// memory and the nudge's budget are two things, and one press must not spend
// both.
func TestRefusingAChoreLeavesItDue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "bins out", oneDay, oneDay)
	require.NoError(t, err)
	backdate(t, store, "bins out", 3)

	chat, _ := chatRecorder("451")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	require.NoError(t, applier.Apply(ctx, itemOf("!now"), &p))
	require.NoError(t, applier.Apply(ctx, tapItem(p, "451", "later:1", true), &p))

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "the clock kept running")
}

func TestNowSaysSoWhenThereIsNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	reply := triage(t, store, p, "!now")
	require.Contains(t, reply, "Nothing")
}

// A low day says nothing of its own initiative, and says how to ask anyway.
func TestNowOnALowDayOffersTheWayThrough(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat", time.Now()))

	reply := triage(t, store, p, "!now")
	require.Contains(t, reply, "anyway")

	require.Contains(t, triage(t, store, p, "!now anyway"), "ring the vet")
}
