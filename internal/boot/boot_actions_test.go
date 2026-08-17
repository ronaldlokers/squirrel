//go:build integration

package boot_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// End to end over a real socket: a tap arrives, is acknowledged, and lands as a
// completion once the drain has run.
func TestBootAppliesATap(t *testing.T) {
	s, store := bootWithStore(t)
	ctx := context.Background()

	p := ownerOf(t, store)
	c := seedOverdueChore(t, store, p, "vacuum")
	// The message id is "1", not a placeholder like "m-1" — ParseAction's
	// pattern only matches a numeric id, and the webhook below reports
	// message.id 1, so the two must agree for the tap to resolve at all.
	promptID := seedSentPrompt(t, store, p, "digest", "1", c)
	_ = promptID

	res, err := http.Post(webhookURL(s), "application/json", strings.NewReader(`{
		"type": "action",
		"room": { "id": 9 },
		"user": { "id": 1 },
		"message": { "id": 1 },
		"action": { "value": "done:1", "selected": true }
	}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Empty(t, res.Header.Get("Content-Type"), "a tap posts nothing back")
	res.Body.Close()

	require.Eventually(t, func() bool {
		due, err := store.DueChores(ctx, p, time.Now())
		return err == nil && len(due) == 0
	}, 10*time.Second, 100*time.Millisecond, "the tap completed the chore")
}
