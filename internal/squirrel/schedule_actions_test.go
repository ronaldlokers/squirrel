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

// sentMessage is one Send call. updates collects the message ids that were
// closed (via Chat.Update) after this Send and before the next one, so a
// digest's own entry shows what it, specifically, went on to disable.
// updateMessages is the Message actually handed to each of those Update
// calls, same order as updates, for asserting on what the close carried
// rather than just that it happened.
type sentMessage struct {
	conversationID string
	message        squirrel.Message
	updates        []string
	updateMessages []squirrel.Message
}

// chatRecorder builds a Chat whose Send hands back the given ids in order —
// "" once they run out — and records every Message sent. Update appends the
// message id it was asked to close, and the Message it was asked to close
// with, onto the most recent Send's entry.
func chatRecorder(ids ...string) (squirrel.Chat, *[]sentMessage) {
	sent := &[]sentMessage{}
	next := 0
	return squirrel.Chat{
		Send: func(_ context.Context, conversationID string, m squirrel.Message) (string, error) {
			*sent = append(*sent, sentMessage{conversationID: conversationID, message: m})
			id := ""
			if next < len(ids) {
				id = ids[next]
				next++
			}
			return id, nil
		},
		Update: func(_ context.Context, _, messageID string, m squirrel.Message) error {
			last := len(*sent) - 1
			(*sent)[last].updates = append((*sent)[last].updates, messageID)
			(*sent)[last].updateMessages = append((*sent)[last].updateMessages, m)
			return nil
		},
	}, sent
}

// chatRecorderFailingUpdate behaves like chatRecorder, except closing a
// previous prompt always fails — for proving that never blocks the new one
// from going out.
func chatRecorderFailingUpdate(ids ...string) (squirrel.Chat, *[]sentMessage) {
	chat, sent := chatRecorder(ids...)
	chat.Update = func(context.Context, string, string, squirrel.Message) error {
		return errors.New("update failed")
	}
	return chat, sent
}

func schedulerWithChat(t *testing.T, store *squirrel.Store, p int64, chat squirrel.Chat) *squirrel.Scheduler {
	t.Helper()
	return squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, Chat: chat, PersonID: p, ConversationID: "9",
		At: 8 * time.Hour, Location: amsterdam(t),
	})
}

func TestDigestRecordsItsMessageID(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	chat, sent := chatRecorder("m-1")
	s := schedulerWithChat(t, store, p, chat)
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))))

	require.Len(t, *sent, 1)
	require.NotEmpty(t, (*sent)[0].message.Actions, "the digest carries buttons")

	var id string
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select external_message_id from prompts where person_id = $1 and kind = 'digest'`,
		p).Scan(&id))
	require.Equal(t, "m-1", id)
}

func TestANewDigestDisablesTheOneBefore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// Tolerance is oneDay rather than the oneWeek phase 2 tests elsewhere use:
	// DueChores gates a chore's reappearance on last_shown + tolerance, and the
	// first Once() call below is the chore's last_shown. A oneWeek tolerance
	// would suppress it from the very next morning's digest regardless of how
	// overdue it still is, which would make the second Once() call send an
	// empty digest for the wrong reason and never reach the code under test.
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneDay)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	chat, sent := chatRecorder("m-1", "m-2")
	s := schedulerWithChat(t, store, p, chat)
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))))
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 16, 8, 0, 1, 0, amsterdam(t))))

	require.Len(t, *sent, 2)
	require.Len(t, (*sent)[0].updates, 0)
	require.Equal(t, []string{"m-1"}, (*sent)[1].updates,
		"the second digest closes the first one's buttons")
}

// The update must carry the previous prompt's own action values — not a
// synthetic button — and no body override, so Campfire's per-user retained
// selection on the old message survives and its text is left untouched.
func TestClosePreviousRebuildsItsOwnActions(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// See the comment in TestANewDigestDisablesTheOneBefore for why the
	// tolerance is short.
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneDay)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	chat, sent := chatRecorder("m-1", "m-2")
	s := schedulerWithChat(t, store, p, chat)
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))))
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 16, 8, 0, 1, 0, amsterdam(t))))

	require.Len(t, (*sent)[1].updateMessages, 1)
	closed := (*sent)[1].updateMessages[0]
	require.Empty(t, closed.Text, "no body override — whatever text the first digest had stays untouched")
	require.Equal(t, []squirrel.Action{{Label: "vacuum", Value: "done:1", Emoji: "✅"}}, closed.Actions,
		"the same action values the first digest was actually sent with, not a synthetic button")
}

// Never let closing the past block speaking in the present.
func TestAFailedDisableStillSendsTheDigest(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// See the comment in TestANewDigestDisablesTheOneBefore: tolerance must be
	// short enough that the chore is still due the very next morning.
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneDay)
	require.NoError(t, err)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack",
		time.Date(2026, 7, 1, 9, 0, 0, 0, amsterdam(t))))

	chat, sent := chatRecorderFailingUpdate("m-1", "m-2")
	s := schedulerWithChat(t, store, p, chat)
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 15, 8, 0, 1, 0, amsterdam(t))))
	require.NoError(t, s.Once(ctx, time.Date(2026, 8, 16, 8, 0, 1, 0, amsterdam(t))),
		"the update failed and the digest still went out")
	require.Len(t, *sent, 2)
}
