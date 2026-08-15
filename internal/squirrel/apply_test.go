//go:build integration

package squirrel_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type sent struct {
	conversationID string
	text           string
}

func recorder() (squirrel.Sender, *[]sent) {
	got := &[]sent{}
	return func(_ context.Context, conversationID, text string) error {
		*got = append(*got, sent{conversationID, text})
		return nil
	}, got
}

func itemOf(text string) squirrel.Item {
	return squirrel.Item{
		Transport:      "campfire",
		ConversationID: squirrel.Ptr("9"),
		RawText:        text,
		Payload:        []byte(`{}`),
		ReceivedAt:     time.Now(),
	}
}

func TestApplyDefinesAChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	require.NoError(t, squirrel.NewApplier(store, send, nil).
		Apply(ctx, itemOf("every 2 weeks: vacuum"), &p))

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, "vacuum", chores[0].Name)

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "every 14 days")
	require.Equal(t, "9", (*got)[0].conversationID)
}

func TestApplyCompletesByPosition(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()
	applier := squirrel.NewApplier(store, send, nil)

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)

	require.NoError(t, applier.Apply(ctx, itemOf("done 1"), &p))

	// Completing resets the baseline to now, so the chore is not due again
	// until a fresh interval has elapsed. Checking at now+interval+1h (the
	// offset TestNewChoreIsNotDueImmediately uses to prove a chore IS due)
	// would cross that threshold and always come back due — it is the
	// baseline reset itself that this test is verifying, so it must check
	// before the next interval, not after it.
	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due)
	require.Contains(t, strings.ToLower((*got)[len(*got)-1].text), "vacuum")
}

// A bare `done` with exactly one line outstanding needs no number.
func TestApplyBareDoneWithOneOutstanding(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)

	require.NoError(t, squirrel.NewApplier(store, send, nil).Apply(ctx, itemOf("done"), &p))

	// See the comment in TestApplyCompletesByPosition: check before the next
	// interval elapses, not a point that is guaranteed to be past it.
	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due)
}

// With several outstanding it lists rather than guessing.
func TestApplyBareDoneWithSeveralOutstandingLists(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	bins, err := store.UpsertChore(ctx, p, "bin day", oneWeek, oneDay)
	require.NoError(t, err)
	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)

	require.NoError(t, squirrel.NewApplier(store, send, nil).Apply(ctx, itemOf("done"), &p))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "bin day")
	require.Contains(t, (*got)[0].text, "vacuum")
}

func TestApplyStopDeactivates(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)

	require.NoError(t, squirrel.NewApplier(store, send, nil).Apply(ctx, itemOf("stop 1"), &p))

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores)
}

func TestApplyNvmRemovesTheChoreJustDefined(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()
	applier := squirrel.NewApplier(store, send, nil)

	require.NoError(t, applier.Apply(ctx, itemOf("every 2 weeks I forget to call my mother"), &p))
	require.NoError(t, applier.Apply(ctx, itemOf("nvm"), &p))

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores)
}

// The undo window is bounded, so `nvm` long after the fact is not a
// destructive surprise against a chore you meant to keep.
func TestApplyNvmDoesNotReachPastTheWindow(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx,
		`update chores set created_at = now() - interval '30 minutes' where id = $1`, c.ID)
	require.NoError(t, err)

	require.NoError(t, squirrel.NewApplier(store, send, nil).Apply(ctx, itemOf("nvm"), &p))

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1, "a chore created half an hour ago is not undone")
	require.Contains(t, (*got)[0].text, "Nothing to undo")
}

// A capture says nothing. The squirrel already went out in the HTTP response.
func TestApplySaysNothingForACapture(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	require.NoError(t, squirrel.NewApplier(store, send, nil).Apply(ctx, itemOf("buy milk"), &p))
	require.Empty(t, *got)
}

// An unknown person cannot own chores, so nothing is applied and nothing sent.
func TestApplyIgnoresAnUnknownPerson(t *testing.T) {
	store := withStore(t)
	send, got := recorder()

	require.NoError(t, squirrel.NewApplier(store, send, nil).
		Apply(context.Background(), itemOf("every 2 weeks: vacuum"), nil))
	require.Empty(t, *got)
}
