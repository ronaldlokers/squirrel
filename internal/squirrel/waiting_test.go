//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestWaitingCountsOpenNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "one thing")
	insertItem(t, store, p, "another thing")

	w, err := store.Waiting(ctx, p, time.Now())
	require.NoError(t, err)
	require.Equal(t, 2, w.Pile)
}

// A note that has been decided about is not waiting. Without this the pile
// number would only ever grow.
func TestWaitingIgnoresDecidedNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	id := insertItem(t, store, p, "one thing")
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, time.Now()))

	w, err := store.Waiting(ctx, p, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, w.Pile)
}

// A note pointing at an appointment still ahead has somewhere to be. The pile
// screen already excludes it; the number must agree, or the door disagrees with
// the room behind it.
func TestWaitingObeysTheSameRuleAsThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: time.Now().Add(3 * time.Hour),
	})
	require.NoError(t, err)
	id := insertItem(t, store, p, "the referral letter")
	ok, err := store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)
	require.True(t, ok)

	w, err := store.Waiting(ctx, p, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, w.Pile)
}

func TestWaitingCountsUndoneTasksOnly(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	open := insertItem(t, store, p, "ring the bank")
	_, err := store.SetItemKind(ctx, p, open, squirrel.ItemTask)
	require.NoError(t, err)

	done := insertItem(t, store, p, "post the form")
	_, err = store.SetItemKind(ctx, p, done, squirrel.ItemTask)
	require.NoError(t, err)
	require.NoError(t, store.SetItemState(ctx, done, squirrel.ItemDone, time.Now()))

	w, err := store.Waiting(ctx, p, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, w.Tasks)
}

// The agenda door is about today. A thing next week is not waiting for you.
func TestWaitingCountsOnlyTodaysFixedPoints(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := squirrel.StartOfDay(time.Now()).Add(9 * time.Hour)

	_, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: now.Add(4 * time.Hour),
	})
	require.NoError(t, err)
	_, err = store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "next week", Starts: now.Add(7 * 24 * time.Hour),
	})
	require.NoError(t, err)

	w, err := store.Waiting(ctx, p, now)
	require.NoError(t, err)
	require.Equal(t, 1, w.Agenda)
}

func TestWaitingIgnoresFixedPointsAlreadyPast(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := squirrel.StartOfDay(time.Now()).Add(14 * time.Hour)

	_, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "this morning", Starts: now.Add(-2 * time.Hour),
	})
	require.NoError(t, err)

	w, err := store.Waiting(ctx, p, now)
	require.NoError(t, err)
	require.Equal(t, 0, w.Agenda)
}

// Due, and not before. The second half is what stops a later author replacing
// the DueChores call with a count query and quietly redefining "due".
func TestWaitingCountsDueChores(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	_, err := store.UpsertChore(ctx, p, "water the plants", 24*time.Hour, time.Hour)
	require.NoError(t, err)

	w, err := store.Waiting(ctx, p, now.Add(48*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, w.Chores)

	w, err = store.Waiting(ctx, p, now)
	require.NoError(t, err)
	require.Equal(t, 0, w.Chores)
}

// One person's doors say nothing about another's.
func TestWaitingIsScopedToThePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	other, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)
	insertItem(t, store, other, "not yours")

	w, err := store.Waiting(ctx, p, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, w.Pile)
}
