//go:build integration

package squirrel_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Whether anything would be sent to, and the way off.
//
// The screen needs both to say what the state is rather than to offer a control
// that cannot report one — which is what the old one-shot button was.
func TestNotifyingSaysWhetherThereIsAnythingToSendTo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	on, err := store.Notifying(ctx, p)
	require.NoError(t, err)
	require.False(t, on, "a person who has told nothing is being notified")

	require.NoError(t, store.SaveSubscription(ctx, p, squirrel.Subscription{
		Endpoint: "https://push.example/abc", P256dh: "key", Auth: "auth",
	}))

	on, err = store.Notifying(ctx, p)
	require.NoError(t, err)
	require.True(t, on, "a browser was told and it does not count")
}

func TestStopNotifyingRetiresEveryBrowser(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for _, at := range []string{"https://push.example/one", "https://push.example/two"} {
		require.NoError(t, store.SaveSubscription(ctx, p, squirrel.Subscription{
			Endpoint: at, P256dh: "key", Auth: "auth",
		}))
	}
	live, err := store.LiveSubscriptions(ctx, p)
	require.NoError(t, err)
	require.Len(t, live, 2)

	require.NoError(t, store.StopNotifying(ctx, p, time.Now()))

	live, err = store.LiveSubscriptions(ctx, p)
	require.NoError(t, err)
	require.Empty(t, live, "a browser is still being sent to after it was turned off")

	on, err := store.Notifying(ctx, p)
	require.NoError(t, err)
	require.False(t, on)
}

// Turning off is per person, and this product has one — but the query says so,
// and one that did not would silence somebody else's browsers.
func TestStopNotifyingLeavesAnotherPersonAlone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.SeedOwner(ctx, "somebody-else", nil)
	require.NoError(t, err)
	require.NotEqual(t, mine, theirs)

	for _, p := range []int64{mine, theirs} {
		require.NoError(t, store.SaveSubscription(ctx, p, squirrel.Subscription{
			Endpoint: "https://push.example/of-" + strconv.FormatInt(p, 10),
			P256dh:   "key", Auth: "auth",
		}))
	}

	require.NoError(t, store.StopNotifying(ctx, mine, time.Now()))

	on, err := store.Notifying(ctx, theirs)
	require.NoError(t, err)
	require.True(t, on, "turning mine off turned theirs off too")
}
