//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// You are awake, holding your phone, and already in the conversation. It is
// the best moment available and phase 3 wasted it by only ever answering.
func TestACaptureCarriesANudgeBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	var called []squirrel.NudgeReason
	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetNudger(func(_ context.Context, _ time.Time, why squirrel.NudgeReason) error {
		called = append(called, why)
		return nil
	})

	require.NoError(t, a.Apply(ctx, squirrel.Item{
		Transport: "campfire", ExternalID: squirrel.Ptr("77"),
		ConversationID: squirrel.Ptr("9"), PersonID: squirrel.Ptr(p),
		RawText: "buy milk", Payload: []byte(`{}`), ReceivedAt: time.Now(),
	}, squirrel.Ptr(p)))

	require.Equal(t, []squirrel.NudgeReason{squirrel.NudgeFromMessage}, called)
}

// A tap is not you opening the conversation, and nudging back at one would
// turn a single completion into a chain.
func TestATapDoesNotCarryANudge(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	var called int
	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetNudger(func(context.Context, time.Time, squirrel.NudgeReason) error {
		called++
		return nil
	})

	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", true), squirrel.Ptr(p)))
	require.Zero(t, called)
}

// `?` opens its own numbered surface — the list itself — and the piggyback
// nudge must not ride in right behind it: Nudge's own closePrevious would
// disable the list's buttons a beat after they were printed and become the
// newest numbered prompt, so `done 3` would answer "I don't have a line 3"
// until the budget is spent and it stops recurring. The spec's trigger table
// says "any inbound capture", not "any inbound message" — a command reply is
// not a capture.
func TestQueryDoesNotCarryANudge(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	var called int
	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetNudger(func(context.Context, time.Time, squirrel.NudgeReason) error {
		called++
		return nil
	})

	require.NoError(t, a.Apply(ctx, itemOf("?"), &p))
	require.Zero(t, called, "a command reply must not carry a nudge back")
	require.Len(t, *got, 1, "exactly one message — the list itself")

	line, ok, err := store.ChoreAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok, "the list's own numbering must still resolve")
	require.Equal(t, c.ID, line.ID)
}

// Fail-open: a nudge that cannot be sent must never turn a stored capture into
// an error.
func TestAFailedNudgeIsNotAnError(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetNudger(func(context.Context, time.Time, squirrel.NudgeReason) error {
		return context.DeadlineExceeded
	})

	require.NoError(t, a.Apply(ctx, squirrel.Item{
		Transport: "campfire", ExternalID: squirrel.Ptr("77"),
		ConversationID: squirrel.Ptr("9"), PersonID: squirrel.Ptr(p),
		RawText: "buy milk", Payload: []byte(`{}`), ReceivedAt: time.Now(),
	}, squirrel.Ptr(p)))
}
