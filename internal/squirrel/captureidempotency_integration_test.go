//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestInsertItemReturningIDIsIdempotentOnCaptureKey(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	first, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: "ask the garage about the rattle",
		Payload: []byte(`{}`), ReceivedAt: time.Now(), CaptureKey: "retry-1",
	})
	require.NoError(t, err)

	second, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: "ask the garage about the rattle",
		Payload: []byte(`{}`), ReceivedAt: time.Now(), CaptureKey: "retry-1",
	})
	require.NoError(t, err)

	require.Equal(t, first, second, "a retried capture key produced a different row")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1, "a retried capture key wrote a second row")
}

func TestInsertItemReturningIDWithNoKeyIsNeverDeduped(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	first, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: "a duplicate on purpose",
		Payload: []byte(`{}`), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)

	second, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: "screen", PersonID: &p, RawText: "a duplicate on purpose",
		Payload: []byte(`{}`), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)

	require.NotEqual(t, first, second)

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
}
