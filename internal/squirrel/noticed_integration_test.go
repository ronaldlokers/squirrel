//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestASecondLineAboutOneThingReplacesTheFirst(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	require.NoError(t, store.Notice(ctx, p, "note", 4, "the first thing said", at))
	require.NoError(t, store.Notice(ctx, p, "note", 4, "the second thing said", at.Add(time.Hour)))

	lines, err := store.WhatWasNoticed(ctx, p)
	require.NoError(t, err)
	require.Len(t, lines, 1, "two lines are stacked under one thing")
	require.Equal(t, "the second thing said", lines[0].Words)
}

func TestARefusedLineIsKeptAndNotDrawn(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	require.NoError(t, store.Notice(ctx, p, "note", 4, "not worth saying", at))
	lines, err := store.WhatWasNoticed(ctx, p)
	require.NoError(t, err)
	require.Len(t, lines, 1)

	refused, err := store.NotUseful(ctx, p, lines[0].ID, at)
	require.NoError(t, err)
	require.True(t, refused)

	lines, err = store.WhatWasNoticed(ctx, p)
	require.NoError(t, err)
	require.Empty(t, lines, "a refused line is still drawn")

	words, err := store.WhatWasRefused(ctx, p, 10)
	require.NoError(t, err)
	require.Equal(t, []string{"not worth saying"}, words,
		"the refusal was forgotten, so the next pass may write it again")
}

func TestNoticingAgainClearsTheRefusal(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	require.NoError(t, store.Notice(ctx, p, "note", 4, "not worth saying", at))
	lines, _ := store.WhatWasNoticed(ctx, p)
	_, err := store.NotUseful(ctx, p, lines[0].ID, at)
	require.NoError(t, err)

	require.NoError(t, store.Notice(ctx, p, "note", 4, "something else entirely", at.Add(time.Hour)))
	lines, err = store.WhatWasNoticed(ctx, p)
	require.NoError(t, err)
	require.Len(t, lines, 1, "a new line about the thing inherited the old one's refusal")
	require.Equal(t, "something else entirely", lines[0].Words)
}

func TestALineIsOnlyYours(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-someone-else", "someone-else")
	require.NoError(t, err)

	require.NoError(t, store.Notice(ctx, theirs, "note", 4, "theirs", time.Now()))
	require.NoError(t, store.Notice(ctx, mine, "note", 5, "mine", time.Now()))

	lines, err := store.WhatWasNoticed(ctx, mine)
	require.NoError(t, err)
	require.Len(t, lines, 1, "someone else's line is on this board")
	require.Equal(t, "mine", lines[0].Words)
}

func TestWhenItLastNoticedComesFromTheRows(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	at, err := store.NoticedAt(ctx, p)
	require.NoError(t, err)
	require.True(t, at.IsZero(), "it has noticed nothing and says otherwise")

	made := time.Now().Truncate(time.Second)
	require.NoError(t, store.Notice(ctx, p, "note", 4, "a line", made))

	at, err = store.NoticedAt(ctx, p)
	require.NoError(t, err)
	require.WithinDuration(t, made, at, time.Second)
}
