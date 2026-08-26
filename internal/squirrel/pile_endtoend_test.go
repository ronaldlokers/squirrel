//go:build integration

package squirrel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The whole of 5a in one path: a thought arrives the way a thought actually
// arrives — through the spool and the drain, not by calling Apply directly —
// and then it is listed, searched, cleared, and gone.
//
// This project has twice shipped an end-to-end test that passed against
// unwired code: once because the assertion never touched the wired path, once
// because a startup send independently produced the state being asserted. So
// this one goes through Drain, and the mutation that proves it is removing the
// command dispatch entirely, not breaking a helper.
func TestThePileEndToEnd(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	// Seeded with an identity, not just a handle: the drain resolves a person
	// from (transport, sender id), and without that the capture still lands but
	// personID is nil and Apply returns before replying to anything. That is
	// correct behaviour — a chore belongs to a person — and it is also exactly
	// how an end-to-end test comes to assert nothing at all.
	p, err := store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "s1"}})
	require.NoError(t, err)

	sp, err := squirrel.OpenSpool(t.TempDir())
	require.NoError(t, err)

	chat, got := chatRecorder("m-1", "m-2", "m-3", "m-4", "m-5", "m-6", "m-7")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: store, Interval: time.Second, Applier: applier,
	})

	// Everything the room sends goes in as a capture would: onto the spool
	// first, then through the drain.
	// Each message needs its own external id. InsertItem is idempotent on
	// (transport, external_id), so saying "!notes" twice with the same id is a
	// redelivery: the row is not inserted again and Apply never runs, which is
	// the guard working correctly and a trap for a test that reuses text.
	nth := 0
	say := func(text string) {
		t.Helper()
		nth++
		_, err := sp.Write(squirrel.Capture{
			Transport:      "campfire",
			ExternalID:     squirrel.Ptr(fmt.Sprintf("e-%d", nth)),
			ConversationID: squirrel.Ptr("9"),
			SenderID:       squirrel.Ptr("s1"),
			Text:           text,
			ReceivedAt:     time.Now(),
			Payload:        []byte(`{}`),
		})
		require.NoError(t, err)
		require.Equal(t, squirrel.DrainResult{Inserted: 1}, drain.Once(ctx))
	}

	lastReply := func() string {
		t.Helper()
		require.NotEmpty(t, *got)
		return (*got)[len(*got)-1].message.Text
	}

	say("the boiler makes a noise on tuesdays")
	say("buy milk")

	// Listing shows both, newest first, and nothing the drain stored on our
	// behalf that was not a thought.
	say("!notes")
	require.Contains(t, lastReply(), "1. buy milk")
	require.Contains(t, lastReply(), "2. the boiler makes a noise")
	require.NotContains(t, lastReply(), "!notes", "a command is not a note")

	// Search reaches a note the pile is not currently showing first.
	say("!find boiler")
	require.Contains(t, lastReply(), "boiler makes a noise")
	require.NotContains(t, lastReply(), "buy milk")

	// A number typed against that search resolves to the note the search
	// printed, not to whatever line 1 meant before it.
	say("keep 1")
	require.Contains(t, lastReply(), "boiler")

	// And the pile is what is left.
	say("!notes")
	require.Contains(t, lastReply(), "1. buy milk")
	require.NotContains(t, lastReply(), "boiler",
		"kept has left the pile, and searching is how it comes back")

	say("done 1")
	say("!notes")
	require.Contains(t, lastReply(), "Nothing in the pile.")

	// Nothing was lost on the way: both thoughts are still stored, and the
	// evening message still reports them, because that window asks about when
	// they arrived rather than what became of them.
	captures, err := store.CapturesSince(ctx, p, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Contains(t, captures, "buy milk")
	require.Contains(t, captures, "the boiler makes a noise on tuesdays")
}

// A note promoted to a chore, end to end, through the same path.
func TestPromotionEndToEnd(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p, err := store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "s1"}})
	require.NoError(t, err)

	sp, err := squirrel.OpenSpool(t.TempDir())
	require.NoError(t, err)

	chat, got := chatRecorder("m-1", "m-2", "m-3", "m-4")
	applier := squirrel.NewApplier(store, nil, chat, nil)
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Spool: sp, Store: store, Interval: time.Second, Applier: applier,
	})

	// Each message needs its own external id. InsertItem is idempotent on
	// (transport, external_id), so saying "!notes" twice with the same id is a
	// redelivery: the row is not inserted again and Apply never runs, which is
	// the guard working correctly and a trap for a test that reuses text.
	nth := 0
	say := func(text string) {
		t.Helper()
		nth++
		_, err := sp.Write(squirrel.Capture{
			Transport:      "campfire",
			ExternalID:     squirrel.Ptr(fmt.Sprintf("e-%d", nth)),
			ConversationID: squirrel.Ptr("9"),
			SenderID:       squirrel.Ptr("s1"),
			Text:           text,
			ReceivedAt:     time.Now(),
			Payload:        []byte(`{}`),
		})
		require.NoError(t, err)
		require.Equal(t, squirrel.DrainResult{Inserted: 1}, drain.Once(ctx))
	}

	say("water the plants")
	say("!notes")
	say("!chore 1 every 2 weeks")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, "water the plants", chores[0].Name)
	require.Equal(t, 14, chores[0].EveryDays)

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, items, "the note left the pile by becoming a chore")

	require.NotEmpty(t, *got)
	require.Contains(t, (*got)[len(*got)-1].message.Text, "water the plants")
}
