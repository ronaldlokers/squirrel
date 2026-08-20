//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// `!stuck` is the ladder's chat half. Feature parity is a standing rule.

func TestStuckAsksOnceAndOffersTheFour(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	reply := triage(t, store, p, "!stuck")
	require.Contains(t, reply, "too big")
	require.Contains(t, reply, "boring")
}

// It acts on whatever the picker would hand you, rather than asking which
// thing you mean. Someone who has just said they cannot start is not the
// person to ask a disambiguating question.
func TestStuckAnswersAboutTheThingYouWouldBeHanded(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	reply := triage(t, store, p, "!stuck too big")

	require.Contains(t, reply, "smallest piece")
	require.Contains(t, reply, "ring the vet", "the timer names the thing")
}

// Not today turns that same thing down, and it is the same write "not now"
// makes.
func TestStuckNotTodayRefusesTheOffer(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	triage(t, store, p, "!stuck not today")

	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found)
}

// A low day is not a reason to refuse to help someone who has come asking.
func TestStuckWorksOnALowDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat", time.Now()))

	require.Contains(t, triage(t, store, p, "!stuck boring"), "ten minutes")
}
