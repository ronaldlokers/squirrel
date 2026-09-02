//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestWhatWasSaidComesBackNewestFirst(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	require.NoError(t, store.RecordSaid(ctx, p,
		squirrel.Push{Title: "the bins", Body: "they go out today", URL: "/"}, at.Add(-4*time.Hour)))
	require.NoError(t, store.RecordSaid(ctx, p,
		squirrel.Push{Title: "time to leave", Body: "the dentist is at 14:30", URL: "/"}, at))

	said, err := store.WhatWasSaid(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, said, 2)
	require.Equal(t, "time to leave", said[0].Title, "the oldest came back first")
	require.Equal(t, "the dentist is at 14:30", said[0].Body)
	require.Equal(t, "the bins", said[1].Title)
}

func TestWhatWasSaidIsOnlyYours(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-someone-else", "someone-else")
	require.NoError(t, err)
	at := time.Now()

	require.NoError(t, store.RecordSaid(ctx, theirs, squirrel.Push{Title: "their dentist"}, at))
	require.NoError(t, store.RecordSaid(ctx, mine, squirrel.Push{Title: "my bins"}, at))

	said, err := store.WhatWasSaid(ctx, mine, 20)
	require.NoError(t, err)
	require.Len(t, said, 1, "someone else's notification is in this list")
	require.Equal(t, "my bins", said[0].Title)
}

func TestWhatWasSaidStopsAtTheLimit(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	for i := 0; i < 5; i++ {
		require.NoError(t, store.RecordSaid(ctx, p,
			squirrel.Push{Title: "one"}, at.Add(time.Duration(i)*time.Minute)))
	}

	said, err := store.WhatWasSaid(ctx, p, 3)
	require.NoError(t, err)
	require.Len(t, said, 3)
}
