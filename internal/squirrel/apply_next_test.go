//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The moment straight after a completion is the cheapest moment to start the
// next thing, and the product used to walk away from it.

func TestFinishingAChoreHandsYouOneMoreThing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	_, err := store.UpsertChore(ctx, p, "bins out", oneDay, oneDay)
	require.NoError(t, err)
	backdate(t, store, "bins out", 3)

	triage(t, store, p, "!now")
	reply := triage(t, store, p, "done 1")

	require.Contains(t, reply, "bins out", "what you just did")
	require.Contains(t, reply, "ring the vet", "and one more thing")
}

// Once, and never a queue.
func TestTheHandOffIsOneThing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	for _, text := range []string{"ring the vet", "book the car in", "post the form"} {
		taskOf(t, store, p, text)
	}

	chat, got := chatRecorder("451", "452")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	require.NoError(t, applier.Apply(context.Background(), itemOf("!now"), &p))
	require.NoError(t, applier.Apply(context.Background(), itemOf("done 1"), &p))

	m := (*got)[1].message
	require.Contains(t, m.Text, "book the car in")
	require.NotContains(t, m.Text, "post the form", "one, not a queue")
	require.Len(t, m.Actions, 2)
}

// The hand-off is answerable, and its number resolves against it.
func TestTheHandOffCanBeDoneStraightAway(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	second := taskOf(t, store, p, "book the car in")

	triage(t, store, p, "!now")
	// The oldest is offered first, so this completes "ring the vet" and the
	// hand-off names "book the car in".
	triage(t, store, p, "done 1")
	triage(t, store, p, "done 1")

	require.Equal(t, "done", stateOf(t, store, second))
}

func TestIgnoringTheHandOffChangesNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	id := taskOf(t, store, p, "book the car in")

	triage(t, store, p, "!now")
	triage(t, store, p, "done 1")

	require.Equal(t, "open", stateOf(t, store, id))
}

// On a low day, finishing one thing is not evidence that you have more in you.
func TestNoHandOffOnALowDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	taskOf(t, store, p, "book the car in")
	triage(t, store, p, "!now anyway")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat", time.Now()))

	reply := triage(t, store, p, "done 1")
	require.NotContains(t, reply, "book the car in")
}

// Already on something. Being handed a second thing mid-timer would read as a
// suggestion to abandon the first.
func TestNoHandOffWhileATimerIsRunning(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	taskOf(t, store, p, "book the car in")
	triage(t, store, p, "!now")
	_, err := store.StartTimer(ctx, p, "the kitchen", 10*time.Minute, time.Now())
	require.NoError(t, err)

	reply := triage(t, store, p, "done 1")
	require.NotContains(t, reply, "book the car in")
	require.NotContains(t, reply, "the kitchen")
}

func TestTriagingANoteHandsYouNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	pileOf(t, store, p, "the tyre is flat")

	reply := triage(t, store, p, "done 1")
	require.NotContains(t, reply, "ring the vet")
}
