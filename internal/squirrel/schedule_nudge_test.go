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

// today anchors a test's clock to today's real calendar date in Amsterdam, at
// a fixed wall-clock time, rather than to some fixed 2026 date. backdateChore
// pushes a chore's created_at relative to the database's real now(), so a
// Go-level "now" pinned to a fixed calendar date drifts further from that
// real now with every day the suite runs after it was written — far enough,
// eventually, to change whether a chore backdated by a fixed duration reads
// as due at that pinned instant. Anchoring to the real calendar date keeps
// the two in step indefinitely.
func today(t *testing.T, hour, min, sec int) time.Time {
	t.Helper()
	loc := amsterdam(t)
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), hour, min, sec, 0, loc)
}

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

// TestSecondNudgeInADaySendsNothing above proves nothing about the budget
// itself: with a single chore, DueChores' own per-chore tolerance gate
// already suppresses the second Nudge call — a nulled sent_for_date in
// nudgeFor (which would let the unique index refuse nothing at all) leaves
// that test just as green. A second due chore is what actually exercises the
// index: DueChores still returns it as an option for the second trigger,
// so only the once-a-day claim itself can be what refuses it. The clock is
// pinned to two fixed instants an hour apart, both mid-morning, rather than
// time.Now() and time.Now().Add(time.Hour) — the latter straddles local
// midnight into two different calendar dates whenever the suite happens to
// run after 23:00, which would make the second Nudge a legitimately new day
// rather than a refused second one.
func TestSecondNudgeInADayIsRefusedEvenWithASecondChoreStillDue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	vacuum, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, vacuum.ID, 20*24*time.Hour)

	mop, err := store.UpsertChore(ctx, p, "mop", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, mop.ID, 20*24*time.Hour)

	chat, sent := chatRecorder("1", "2")
	s := schedulerWithChat(t, store, p, chat)

	day := today(t, 9, 0, 0)
	require.NoError(t, s.Nudge(ctx, day, squirrel.NudgeFromMessage))
	require.NoError(t, s.Nudge(ctx, day.Add(time.Hour), squirrel.NudgeFromArrival),
		"a refused second nudge is the budget working, not a failure")
	require.Len(t, *sent, 1,
		"two chores still due must not defeat the once-a-day budget")
}

// A transient Campfire error at the moment of a nudge must not spend the
// day's slot: nudgeFor commits the dated row before the send is even
// attempted, so without cleanup the claim survives a failure the room never
// actually saw, and every later trigger — including the 19:00 fallback the
// spec relies on to catch exactly this — is refused by a message that was
// never delivered.
func TestFailedNudgeSendDoesNotSpendTheDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	var calls int
	chat := squirrel.Chat{
		Send: func(context.Context, string, squirrel.Message) (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("campfire unreachable")
			}
			return "m-2", nil
		},
	}
	s := schedulerWithChat(t, store, p, chat)

	day := today(t, 9, 0, 0)
	require.Error(t, s.Nudge(ctx, day, squirrel.NudgeFromMessage),
		"the transport failure must surface")

	require.NoError(t, s.Nudge(ctx, day.Add(time.Hour), squirrel.NudgeFromArrival),
		"a later trigger the same day must still be able to claim the slot the failed send never delivered")
	require.Equal(t, 2, calls,
		"the second trigger must attempt a real send, not be refused by the failed attempt's stale claim")
}

// The same defect as TestFailedNudgeSendDoesNotSpendTheDay, entered through
// once() rather than Nudge() directly: nudgeFor is tried first inside once()
// too, so a chore that rides along on a failed evening send commits the same
// claimed-but-undelivered nudge row. Without the matching cleanup on that
// path, the row survives and every later trigger the same day — including a
// direct Nudge() — is refused by a message the room never received.
func TestFailedOnceSendDoesNotSpendTheDaysNudge(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	var calls int
	chat := squirrel.Chat{
		Send: func(context.Context, string, squirrel.Message) (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("campfire unreachable")
			}
			return "m-2", nil
		},
	}
	s := schedulerWithChat(t, store, p, chat)

	evening := today(t, 19, 0, 1)
	require.Error(t, s.Once(ctx, evening), "the transport failure must surface")

	require.NoError(t, s.Nudge(ctx, evening.Add(time.Hour), squirrel.NudgeFromArrival),
		"a later trigger the same day must still be able to claim the slot the failed once() send never delivered")
	require.Equal(t, 2, calls,
		"the second trigger must attempt a real send, not be refused by the failed once()'s stale claim")
}

// A cancelled context must not defeat the same cleanup: a rollout tearing
// down the scheduler loop mid-send cancels the context the send is running
// on, so the send fails with context.Canceled — and if the cleanup delete
// reused that same context, it would fail for the identical reason, leaving
// the claimed row in place and the day spent. Observed in production as
// "deleting undelivered nudge prompt 1: deleting prompt: context canceled".
// The cleanup must run on a context detached from that cancellation.
func TestNudgeCleanupSurvivesTheSendsCancelledContext(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	sendCtx, cancel := context.WithCancel(context.Background())
	var calls int
	chat := squirrel.Chat{
		Send: func(sc context.Context, _ string, _ squirrel.Message) (string, error) {
			calls++
			if calls == 1 {
				// The send fails because its own context was just cancelled —
				// not a plain transport error.
				cancel()
				return "", sc.Err()
			}
			return "m-2", nil
		},
	}
	s := schedulerWithChat(t, store, p, chat)

	day := today(t, 9, 0, 0)
	require.Error(t, s.Nudge(sendCtx, day, squirrel.NudgeFromMessage),
		"the cancellation must surface as the send's failure")

	require.NoError(t, s.Nudge(ctx, day.Add(time.Hour), squirrel.NudgeFromArrival),
		"a later trigger on a fresh context must still be able to claim the slot — "+
			"the cleanup must not itself be defeated by the send's now-cancelled context")
	require.Equal(t, 2, calls,
		"the second trigger must attempt a real send, not be refused by a claim "+
			"the cancelled-context cleanup failed to release")
}

// Nudge is the only place that ever opens a numbered surface without once()
// right behind it — once() skips closePrevious on a day it claims nothing
// new (see its own comment) — so if Nudge did not close the previous surface
// itself, nothing would: a day-1 button would still be live on day 10,
// growing by one every day a nudge fires. That is not just a stray button:
// RetractCompletion is bounded only below, by e.occurred_at >= p.sent_at, so
// a button left live that long would retract every completion of that chore
// since day 1, not only the one it was originally sent for.
func TestSecondDaysNudgeClosesTheFirsts(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// Tolerance is oneDay, not oneWeek: DueChores gates reappearance on
	// last_shown + tolerance, and day 1's nudge is the chore's last_shown. A
	// longer tolerance would suppress it from day 2's nudge regardless of how
	// overdue it still is, and the test would never reach the code under
	// test.
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneDay)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	chat, sent := chatRecorder("1", "2")
	s := schedulerWithChat(t, store, p, chat)

	day1 := today(t, 10, 0, 0)
	require.NoError(t, s.Nudge(ctx, day1, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)
	require.Empty(t, (*sent)[0].updates, "nothing to close on the first nudge")

	day2 := day1.Add(24 * time.Hour)
	require.NoError(t, s.Nudge(ctx, day2, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 2)
	require.Equal(t, []string{"1"}, (*sent)[1].updates,
		"the second day's nudge must close the first day's still-live button")
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

// This is the case that motivated the whole change: today, an arrival that
// legitimately produces no message is indistinguishable in the logs from an
// arrival that never happened at all. "presence: ping accepted" only
// promises a nudge attempt follows, not that it finds anything — this is
// what closes that gap.
func TestNudgeWithNothingDueLogsIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	chat, sent := chatRecorder("1")
	s := schedulerWithChat(t, store, p, chat)

	logs := captureLogs(t)
	require.NoError(t, s.Nudge(ctx, time.Now(), squirrel.NudgeFromArrival))
	require.Empty(t, *sent)

	require.Contains(t, logs.String(), "nudge: nothing due")
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
	now := today(t, 19, 0, 1)

	require.NoError(t, s.Nudge(ctx, now, squirrel.NudgeFromMessage))
	require.NoError(t, s.Once(ctx, now))
	require.Len(t, *sent, 2, "the nudge and the evening message are different prompts")
	require.Contains(t, (*sent)[1].message.Text, "buy milk")
	require.Empty(t, (*sent)[1].message.Actions, "the nudge already went; no second button")
}

// The evening message carries no button of its own once a nudge has already
// claimed the day, so it must not disable the nudge's still-live one — and
// the chore that nudge named must still resolve, both by tap (through the
// nudge row's own external_message_id) and by the typed/bare "done" paths
// (through OutstandingLines, which only ever looks at numberedKinds). Before
// this was fixed, 'evening' being numbered with zero lines of its own made
// it win latestPrompt on exactly this day, so OutstandingLines found nothing
// even though the chore was genuinely still due and its button still live.
func TestNudgeStaysResolvableAfterTheEveningMessage(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)
	// A capture is what gives the evening message something to say — the
	// chore itself is no longer due by 19:00, since the nudge above already
	// shown it moments earlier is well inside its week-long tolerance.
	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport: "campfire", PersonID: squirrel.Ptr(p),
		RawText: "buy milk", Payload: []byte(`{}`), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)

	chat, sent := chatRecorder("1", "2")
	s := schedulerWithChat(t, store, p, chat)
	now := today(t, 19, 0, 1)

	require.NoError(t, s.Nudge(ctx, now, squirrel.NudgeFromMessage))
	require.NoError(t, s.Once(ctx, now))
	require.Len(t, *sent, 2)
	require.Empty(t, (*sent)[1].updates,
		"a message with no button of its own must not close the nudge's still-live one")

	outstanding, err := store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Len(t, outstanding, 1, "the nudged chore must still resolve for a bare done")
	require.Equal(t, "vacuum", outstanding[0].Name)
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

	day1Evening := today(t, 19, 0, 1)
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

	// Derived from day1Evening, one calendar day later, rather than a second
	// independent call to today(): that guarantees day2Morning and
	// day2Evening land on the day right after day1Evening's regardless of
	// when each line happens to run, the same way TestSecondDaysNudgeClosesTheFirsts
	// derives day2 from day1 by adding a day rather than pinning it separately.
	day2 := day1Evening.AddDate(0, 0, 1)
	day2Morning := time.Date(day2.Year(), day2.Month(), day2.Day(), 10, 0, 0, 0, amsterdam(t))
	require.NoError(t, s.Nudge(ctx, day2Morning, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 2, "the intervening nudge sends its own message")

	day2Evening := time.Date(day2.Year(), day2.Month(), day2.Day(), 19, 0, 1, 0, amsterdam(t))
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

	require.NoError(t, s.Once(ctx, today(t, 19, 0, 1)))
	require.Len(t, *sent, 1, "one message, not two")
	require.Contains(t, (*sent)[0].message.Text, "vacuum")
	require.Len(t, (*sent)[0].message.Actions, 1)
}

// PickChore's overdue weighting exists to stop the same most-overdue chore
// from being nudged every single day — but that only holds if a chore that
// was just nudged stops counting as due until its tolerance passes.
// DueChores' last_shown must recognise a nudge, not just a digest, as having
// shown a chore.
func TestNudgedChoreIsNotOfferedAgainWithinItsTolerance(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 20*24*time.Hour)

	chat, sent := chatRecorder("1", "2")
	s := schedulerWithChat(t, store, p, chat)
	now := time.Now()

	require.NoError(t, s.Nudge(ctx, now, squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1)

	require.NoError(t, s.Nudge(ctx, now.Add(24*time.Hour), squirrel.NudgeFromArrival))
	require.Len(t, *sent, 1, "still inside the week tolerance since yesterday's nudge")
}
