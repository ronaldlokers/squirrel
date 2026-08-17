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

type boostCall struct{ conversationID, messageID, content string }

func boostRecorder() (squirrel.Chat, *[]boostCall) {
	var calls []boostCall
	return squirrel.Chat{
		Boost: func(_ context.Context, conversationID, messageID, content string) error {
			calls = append(calls, boostCall{conversationID, messageID, content})
			return nil
		},
	}, &calls
}

func TestHandledCaptureGetsATick(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()
	chat, boosts := boostRecorder()

	a := squirrel.NewApplier(store, send, chat, nil)
	require.NoError(t, a.Apply(ctx, squirrel.Item{
		Transport:      "campfire",
		ExternalID:     squirrel.Ptr("77"),
		ConversationID: squirrel.Ptr("9"),
		PersonID:       squirrel.Ptr(p),
		RawText:        "buy milk",
		Payload:        []byte(`{}`),
		ReceivedAt:     time.Now(),
	}, squirrel.Ptr(p)))

	require.Len(t, *boosts, 1)
	require.Equal(t, "✅", (*boosts)[0].content)
	require.Equal(t, "77", (*boosts)[0].messageID)
	require.Equal(t, "9", (*boosts)[0].conversationID)
}

// A tap is not a message in the room, so there is nothing to boost.
func TestATapIsNotBoosted(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()
	chat, boosts := boostRecorder()

	a := squirrel.NewApplier(store, send, chat, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "m-1", "done:1", true), squirrel.Ptr(p)))
	require.Empty(t, *boosts)
}

// Fail-open, unchanged from phase 1: the receipt is cosmetic and the capture is
// already durable.
func TestAFailedTickIsNotAnError(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	chat := squirrel.Chat{Boost: func(context.Context, string, string, string) error {
		return errors.New("campfire unreachable")
	}}

	a := squirrel.NewApplier(store, send, chat, nil)
	require.NoError(t, a.Apply(ctx, squirrel.Item{
		Transport:      "campfire",
		ExternalID:     squirrel.Ptr("77"),
		ConversationID: squirrel.Ptr("9"),
		PersonID:       squirrel.Ptr(p),
		RawText:        "buy milk",
		Payload:        []byte(`{}`),
		ReceivedAt:     time.Now(),
	}, squirrel.Ptr(p)))
}
