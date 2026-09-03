//go:build integration

package squirrel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The daily read of the board.

// aNoticer records what it was shown and answers with what it is told to.
type aNoticer struct {
	shown   [][]squirrel.NoticeThing
	refused [][]string
	says    []squirrel.NoticeNote
	err     error
}

func (n *aNoticer) notice(_ context.Context, _ int64, things []squirrel.NoticeThing,
	refused []string) ([]squirrel.NoticeNote, error) {

	n.shown = append(n.shown, things)
	n.refused = append(n.refused, refused)
	return n.says, n.err
}

func noticing(t *testing.T, store *squirrel.Store, p int64, n *aNoticer) *squirrel.Scheduler {
	t.Helper()
	return squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, PersonID: p, ConversationID: "9",
		At: 8 * time.Hour, Location: time.UTC,
		Send:    func(context.Context, string, string) error { return nil },
		OnError: func(error) {},
		Notice:  n.notice,
	})
}

func aNote(t *testing.T, store *squirrel.Store, p int64, words string) int64 {
	t.Helper()
	id, err := store.InsertItemReturningID(context.Background(), squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: words,
		Payload: []byte(squirrel.ScreenCapture), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)
	return id
}

func TestTheDailyPassReadsTheBoardAndKeepsWhatItNoticed(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	first := aNote(t, store, p, "ring the garage about the MOT")
	aNote(t, store, p, "tyre pressure")

	n := &aNoticer{says: []squirrel.NoticeNote{
		{Kind: "note", RefID: first, Words: "Both of these are the same trip."},
	}}
	require.NoError(t, noticing(t, store, p, n).NoticeTick(ctx, time.Now()))

	require.Len(t, n.shown, 1, "the board was not read")
	require.Len(t, n.shown[0], 2, "the pass was shown something other than the board")

	lines, err := store.WhatWasNoticed(ctx, p)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "Both of these are the same trip.", lines[0].Words)
	require.Equal(t, first, lines[0].RefID)
}

func TestTheDailyPassRunsOnceADay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	first := aNote(t, store, p, "ring the garage")
	aNote(t, store, p, "tyre pressure")

	n := &aNoticer{says: []squirrel.NoticeNote{{Kind: "note", RefID: first, Words: "one trip"}}}
	s := noticing(t, store, p, n)
	now := time.Now()

	require.NoError(t, s.NoticeTick(ctx, now))
	require.NoError(t, s.NoticeTick(ctx, now.Add(time.Hour)))
	require.Len(t, n.shown, 1, "it read the board twice in one day")

	require.NoError(t, s.NoticeTick(ctx, now.Add(25*time.Hour)))
	require.Len(t, n.shown, 2, "it never reads the board again")
}

func TestOneThingOnTheBoardIsNotWorthReading(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	aNote(t, store, p, "the only thing here")

	n := &aNoticer{}
	require.NoError(t, noticing(t, store, p, n).NoticeTick(ctx, time.Now()))

	require.Empty(t, n.shown, "it spent a call to find what one row says about nothing")
}

func TestWhatWasRefusedIsShownToTheNextPass(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	first := aNote(t, store, p, "ring the garage")
	aNote(t, store, p, "tyre pressure")

	require.NoError(t, store.Notice(ctx, p, "note", first, "a line you did not want", time.Now()))
	lines, _ := store.WhatWasNoticed(ctx, p)
	_, err := store.NotUseful(ctx, p, lines[0].ID, time.Now())
	require.NoError(t, err)

	n := &aNoticer{}
	require.NoError(t, noticing(t, store, p, n).NoticeTick(ctx, time.Now().Add(48*time.Hour)))

	require.Len(t, n.refused, 1)
	require.Equal(t, []string{"a line you did not want"}, n.refused[0],
		"the pass was not told what had already been refused")
}

func TestAPassThatCouldNotRunLeavesTheClockAlone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	aNote(t, store, p, "ring the garage")
	aNote(t, store, p, "tyre pressure")

	n := &aNoticer{err: errors.New("unreachable")}
	s := noticing(t, store, p, n)

	require.NoError(t, s.NoticeTick(ctx, time.Now()))
	require.NoError(t, s.NoticeTick(ctx, time.Now().Add(time.Minute)))
	require.Len(t, n.shown, 2, "a failed pass spent the day")
}
