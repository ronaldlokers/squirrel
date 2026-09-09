//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func recurring(t *testing.T, store *squirrel.Store, personID int64, label string, in time.Duration, weeks int) squirrel.Moment {
	t.Helper()
	m, err := store.CreateMoment(context.Background(), personID, squirrel.Moment{
		Label: label, Starts: time.Now().Add(in),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute, EveryWeeks: weeks,
	})
	require.NoError(t, err)
	return m
}

func TestClosingOneThatComesRoundArmsTheNextOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	m := recurring(t, store, p, "the physio", 2*time.Hour, 2)
	require.NoError(t, store.MomentDone(ctx, p, m.ID, now))

	got, err := store.Upcoming(ctx, p, now, 20)
	require.NoError(t, err)
	require.Len(t, got, 1, "it came round and nothing is ahead of you")
	require.NotEqual(t, m.ID, got[0].ID, "the row you kept was moved rather than replaced")
	require.Equal(t, "the physio", got[0].Label)
	require.Equal(t, 2, got[0].EveryWeeks, "the next one forgot that it comes round")
	require.WithinDuration(t, m.Starts.AddDate(0, 0, 14), got[0].Starts, time.Second,
		"the next one is not a fortnight after the one you kept")
	require.Equal(t, 15*time.Minute, got[0].Travel, "the next one forgot how far away it is")
}

func TestClosingOneThatHappensOnceArmsNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	m := aFixedPoint(t, store, p, "the dentist", 2*time.Hour)
	require.NoError(t, store.MomentDone(ctx, p, m.ID, now))

	got, err := store.Upcoming(ctx, p, now, 20)
	require.NoError(t, err)
	require.Empty(t, got, "a fixed point that happens once came round again")
}

func TestClosingTheSameOneTwiceArmsOneNextOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	m := recurring(t, store, p, "the physio", 2*time.Hour, 1)
	require.NoError(t, store.MomentDone(ctx, p, m.ID, now))
	require.NoError(t, store.MomentDone(ctx, p, m.ID, now))

	got, err := store.Upcoming(ctx, p, now, 20)
	require.NoError(t, err)
	require.Len(t, got, 1, "a second press put a second copy in the diary")
}

func TestOneThatComesRoundIsNotSomebodyElsesToClose(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	other, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)
	now := time.Now()

	m := recurring(t, store, p, "the physio", 2*time.Hour, 1)
	require.NoError(t, store.MomentDone(ctx, other, m.ID, now))

	got, err2 := store.Upcoming(ctx, p, now, 20)
	require.NoError(t, err2)
	require.Len(t, got, 1)
	require.Equal(t, m.ID, got[0].ID, "somebody else closed it, and it came round in your diary")
}
