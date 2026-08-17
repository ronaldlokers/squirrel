//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestCompletingByTapEarnsAReaction(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()
	chat, boosts := boostRecorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, p, "9", "nudge", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	a := squirrel.NewApplier(store, send, chat, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", true), squirrel.Ptr(p)))

	require.Len(t, *boosts, 1, "the tap earns exactly one reaction")
	require.Equal(t, "1", (*boosts)[0].messageID, "on the message that named the chore")
	require.Contains(t, squirrel.Reactions, (*boosts)[0].content)
}

// Campfire's background job carries no event id in the tap payload, so a
// retried delivery is byte-identical to a second genuine tap — applyAction's
// CompletedSince guard is what tells them apart, and it must stop the second
// delivery before react is ever reached. Built with a live boostRecorder,
// not a bare Chat{}, so a spurious second call would actually be recorded
// rather than silently absorbed by react's nil-Boost guard.
func TestRedeliveredTapEarnsOnlyOneReaction(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()
	chat, boosts := boostRecorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, p, "9", "nudge", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	a := squirrel.NewApplier(store, send, chat, nil)
	item := tapItem(p, "1", "done:1", true)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	require.Len(t, *boosts, 1, "a redelivered tap must not earn a second reaction")
}

// Un-tapping is a correction, not an achievement.
func TestUnTappingEarnsNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()
	chat, boosts := boostRecorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, p, "9", "nudge", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	a := squirrel.NewApplier(store, send, chat, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", true), squirrel.Ptr(p)))
	before := len(*boosts)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", false), squirrel.Ptr(p)))
	require.Len(t, *boosts, before, "no reaction for taking it back")
}

// Nothing accrues, so nothing can be lost — that is the whole rule. Every
// reaction must be positive or neutral, and none may imply a tally.
func TestReactionsCannotPunish(t *testing.T) {
	require.NotEmpty(t, squirrel.Reactions)
	for _, r := range squirrel.Reactions {
		require.NotContains(t, []string{"😞", "😡", "👎", "⚠️", "❌"}, r)
	}
}
