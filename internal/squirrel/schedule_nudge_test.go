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

// A nudge carries a date too — it has to, to get its own slot in the per-day
// index — and it is usually the more recent dated prompt on any day one
// fires. LastDigestSentAt must still anchor the next evening message's
// capture window to the last evening message, not to that nudge: a capture
// made between the two would otherwise fall before the (wrongly late) anchor
// and never appear in any future message.
func TestEveningCaptureWindowSurvivesAnInterveningNudge(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	day1Evening := time.Date(2026, 8, 17, 19, 0, 1, 0, amsterdam(t))
	_, err := store.InsertItem(ctx, squirrel.Item{
		Transport: "campfire", PersonID: squirrel.Ptr(p),
		RawText: "before the first evening message", Payload: []byte(`{}`),
		ReceivedAt: day1Evening.Add(-time.Hour),
	})
	require.NoError(t, err)

	chat, sent := chatRecorder("e-1", "n-1", "e-2")
	s := schedulerWithChat(t, store, p, chat)

	require.NoError(t, s.Once(ctx, day1Evening))
	require.Len(t, *sent, 1)
	require.Contains(t, (*sent)[0].message.Text, "before the first evening message")

	// A chore only becomes due after the first evening message, so it is the
	// intervening nudge below that first claims it — the first evening
	// message above ran with nothing due at all.
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport: "campfire", PersonID: squirrel.Ptr(p),
		RawText: "after the first evening, before the nudge", Payload: []byte(`{}`),
		ReceivedAt: day1Evening.Add(2 * time.Hour),
	})
	require.NoError(t, err)

	day2Morning := time.Date(2026, 8, 18, 10, 0, 0, 0, amsterdam(t))
	require.NoError(t, s.Nudge(ctx, day2Morning, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 2, "the intervening nudge sends its own message")

	day2Evening := time.Date(2026, 8, 18, 19, 0, 1, 0, amsterdam(t))
	require.NoError(t, s.Once(ctx, day2Evening))
	require.Len(t, *sent, 3)
	require.Contains(t, (*sent)[2].message.Text, "after the first evening, before the nudge",
		"a capture made before an intervening nudge must still reach the next evening message")
	require.NotContains(t, (*sent)[2].message.Text, "before the first evening message",
		"but a capture the first evening message already carried must not repeat")
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
