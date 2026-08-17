//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestNudgeSendsOneChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	chat, sent := chatRecorder("1")
	s := schedulerWithChat(t, store, p, chat)

	require.NoError(t, s.Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
	require.Contains(t, (*sent)[0].message.Text, "vacuum")
	require.Len(t, (*sent)[0].message.Actions, 1)
}

// The budget is the design. Two triggers on one day produce one nudge, and the
// index is what enforces it — not anything held in memory.
func TestSecondNudgeInADaySendsNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	chat, sent := chatRecorder("1", "2")
	s := schedulerWithChat(t, store, p, chat)
	now := time.Now()

	require.NoError(t, s.Nudge(ctx, now, squirrel.NudgeFromMessage))
	require.NoError(t, s.Nudge(ctx, now.Add(time.Hour), squirrel.NudgeFromArrival),
		"a refused second nudge is the budget working, not a failure")
	require.Len(t, *sent, 1)
}

func TestNudgeWithNothingDueSendsNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	chat, sent := chatRecorder("1")
	s := schedulerWithChat(t, store, p, chat)

	require.NoError(t, s.Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))
	require.Empty(t, *sent)
}

// The evening message runs whether or not a nudge already fired — it is the
// floor for captures, not an alternative to them.
func TestEveningRunsAfterANudge(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)
	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport: "campfire", PersonID: squirrel.Ptr(p),
		RawText: "buy milk", Payload: []byte(`{}`), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)

	chat, sent := chatRecorder("1", "2")
	s := schedulerWithChat(t, store, p, chat)
	now := time.Date(2026, 8, 17, 19, 0, 1, 0, amsterdam(t))

	require.NoError(t, s.Nudge(ctx, now, squirrel.NudgeFromMessage))
	require.NoError(t, s.Once(ctx, now))
	require.Len(t, *sent, 2, "the nudge and the evening message are different prompts")
	require.Contains(t, (*sent)[1].message.Text, "buy milk")
	require.Empty(t, (*sent)[1].message.Actions, "the nudge already went; no second button")
}

// On a quiet day the fallback nudge joins the evening message rather than
// arriving as a second notification a second apart.
func TestEveningCarriesTheNudgeOnAQuietDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	chat, sent := chatRecorder("1")
	s := schedulerWithChat(t, store, p, chat)

	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 17, 19, 0, 1, 0, amsterdam(t))))
	require.Len(t, *sent, 1, "one message, not two")
	require.Contains(t, (*sent)[0].message.Text, "vacuum")
	require.Len(t, (*sent)[0].message.Actions, 1)
}
