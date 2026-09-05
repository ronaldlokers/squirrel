//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestNotThisOneIsAFreshQuestionTomorrow(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NoError(t, store.NotThisOne(ctx, p, o.Kind, o.RefID, time.Now()))

	_, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.False(t, found, "a wrong pick was offered again the same day")

	tomorrow := time.Now().Add(48 * time.Hour)
	again, found, err := store.PickNow(ctx, p, tomorrow, false)
	require.NoError(t, err)
	require.True(t, found, "a pick that was wrong once was hidden forever")
	require.Equal(t, o.RefID, again.RefID)
}

func TestNotThisOneIsWrittenDownAsItsOwnAnswer(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NoError(t, store.NotThisOne(ctx, p, o.Kind, o.RefID, time.Now()))

	var answer string
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select answer from offers where person_id = $1 order by answered_at desc limit 1`,
		p).Scan(&answer))
	require.Equal(t, "wrong", answer, "a wrong pick was recorded as a deferral")
}

func TestRefusingExpiresPastMidnightToo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NoError(t, store.Refuse(ctx, p, o.Kind, o.RefID, time.Now()))

	tomorrow := time.Now().Add(48 * time.Hour)
	again, found, err := store.PickNow(ctx, p, tomorrow, false)
	require.NoError(t, err)
	require.True(t, found, "a same-day deferral was still suppressed the next day")
	require.Equal(t, o.RefID, again.RefID)
}

func TestNotThisOneMovesOnToTheNextThing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	taskOf(t, store, p, "book the car in")

	first, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NoError(t, store.NotThisOne(ctx, p, first.Kind, first.RefID, time.Now()))

	second, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	require.NotEqual(t, first.RefID, second.RefID)
	require.Equal(t, "book the car in", second.Text)
}

func TestNotThisOneRecordsNoCountAnywhereAnOfferReads(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	taskOf(t, store, p, "book the car in")

	o, _, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.NoError(t, store.NotThisOne(ctx, p, o.Kind, o.RefID, time.Now()))

	next, found, err := store.PickNow(ctx, p, time.Now(), false)
	require.NoError(t, err)
	require.True(t, found)
	m := squirrel.NowMessage(next)
	require.NotContains(t, m.Text, "1")
	require.NotContains(t, m.Text, "wrong")
}
