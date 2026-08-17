//go:build integration

package boot_test

import (
	"context"
	"encoding/json"
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

// TestBootSendsChoreConfirmationWithButtons is the wiring's own test.
// TestBootAppliesATap above proves the webhook-to-database pipeline works,
// but applyAction never touches Chat — a tap resolves purely against the
// store, so that test passes whether or not boot.go wires Chat in at all.
// Defining a chore is different: apply()'s reply goes through sendMessage,
// which picks between chat.Send and the phase-2 plain-text Sender depending
// on whether Chat carries a Send func. Wired, the confirmation reaches
// Campfire as JSON carrying DefinedMessage's "make it a note" (undefine:1)
// button; unwired, it reaches Campfire as text/plain with no actions at all.
// Asserting on the *shape* of the body — an actions array with at least one
// entry — rather than its exact bytes is what lets this test tell those two
// states apart without coupling to field ordering or the button's label.
func TestBootSendsChoreConfirmationWithButtons(t *testing.T) {
	withStore(t)
	stubURL, requests := campfireStub(t)
	s := boots(t, envFor(t, map[string]string{"CAMPFIRE_BASE_URL": stubURL}))

	body := strings.Replace(payload, `"plain": "buy milk"`, `"plain": "every 2 weeks: vacuum"`, 1)
	res, err := http.Post(
		"http://127.0.0.1:"+itoa(s.Port())+"/transports/campfire",
		"application/json", strings.NewReader(body))
	require.NoError(t, err)
	res.Body.Close()

	require.Eventually(t, func() bool {
		for _, r := range requests() {
			if r.method != http.MethodPost || !strings.HasPrefix(r.contentType, "application/json") {
				continue
			}
			var decoded struct {
				Actions []json.RawMessage `json:"actions"`
			}
			if err := json.Unmarshal(r.body, &decoded); err != nil {
				continue
			}
			if len(decoded.Actions) > 0 {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond,
		"the confirmation reached the stub as JSON carrying an actions array — "+
			"a Chat that had fallen back to plain text never would")
}
