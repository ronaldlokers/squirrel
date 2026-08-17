//go:build integration

package boot_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/boot"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// testDatabaseURL fails rather than skips, so an unset variable in CI is a
// failure and not a green run over nothing.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")
	return url
}

// campfireRequest is one HTTP call the stub received, kept for assertions
// about what the transport actually sent rather than just that it sent
// something. Content-Type is what tells apart a Chat.Send (JSON, carrying
// whatever buttons the message had) from a boost or a phase-2 plain-text
// Sender call (text/plain).
type campfireRequest struct {
	method      string
	path        string
	contentType string
	body        []byte
}

// campfireStub is somewhere harmless for the applier and scheduler to post:
// it accepts anything and answers 201, same as a real Campfire boost/message
// endpoint would on success. The returned requests func hands back everything
// received so far, so a test can inspect the shape of an outbound body —
// whether the confirmation went out through Chat as JSON carrying an actions
// array, or fell back to the plain-text Sender — rather than only knowing a
// request arrived.
func campfireStub(t *testing.T) (baseURL string, requests func() []campfireRequest) {
	t.Helper()

	var mu sync.Mutex
	var got []campfireRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, campfireRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			body:        body,
		})
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	return srv.URL, func() []campfireRequest {
		mu.Lock()
		defer mu.Unlock()
		out := make([]campfireRequest, len(got))
		copy(out, got)
		return out
	}
}

func withStore(t *testing.T) *squirrel.Store {
	t.Helper()
	ctx := context.Background()

	store, err := squirrel.OpenStore(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(store.Close)

	require.NoError(t, store.Migrate(ctx))
	truncateAll(t, store)
	return store
}

func truncateAll(t *testing.T, store *squirrel.Store) {
	t.Helper()
	_, err := store.Pool().Exec(context.Background(),
		`truncate table prompt_lines, prompts, events, items, chores, identities, people
		 restart identity cascade`)
	require.NoError(t, err)
}

// bootWithStore boots a real Squirrel over a real socket, and hands back the
// store used to seed and assert against it. The two are opened separately —
// withStore migrates and truncates before Boot ever touches Postgres — but
// both point at the same TEST_DATABASE_URL, so what one seeds the other's
// server can see once its own connectAndDrain reaches the database.
//
// The conversation id is overridden to "9" — the room seedSentPrompt records
// its prompt against, following internal/squirrel's own apply_action_test.go
// convention — so a tap arriving from that room is not silently dropped by
// the sink's Allow check before it ever reaches the applier.
func bootWithStore(t *testing.T) (*boot.Squirrel, *squirrel.Store) {
	t.Helper()
	store := withStore(t)
	s := boots(t, envFor(t, map[string]string{"CAMPFIRE_CONVERSATION_ID": "9"}))
	return s, store
}

// ownerOf seeds (or reconciles) the same "ronald" person boot.Boot's own
// SeedOwner reconciles to, and hands back its id. SeedOwner is idempotent on
// handle, so whichever of the two calls runs first is the one that creates
// the row — both return the same id either way.
func ownerOf(t *testing.T, store *squirrel.Store) int64 {
	t.Helper()
	id, err := store.SeedOwner(context.Background(), "ronald", nil)
	require.NoError(t, err)
	return id
}

// seedOverdueChore creates a chore and backdates its creation far enough past
// a two-week interval that it is due immediately, the same way
// internal/squirrel's own backdateChore does for the applier's tests.
//
// The tolerance is kept well under 2 hours on purpose. DueChores' digest gate
// masks a chore for last_shown+tolerance-2h once a digest has recorded it —
// seedSentPrompt below records one moments before the webhook fires — and
// with a multi-day tolerance that mask alone would read empty regardless of
// whether the tap ever landed, exactly the trap described in the task brief.
// A short tolerance keeps last_shown+tolerance-2h in the past, so the mask
// never engages and "due" reflects only the chore's own completion state.
func seedOverdueChore(t *testing.T, store *squirrel.Store, personID int64, name string) squirrel.Chore {
	t.Helper()
	ctx := context.Background()

	c, err := store.UpsertChore(ctx, personID, name, 14*24*time.Hour, 30*time.Minute)
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx,
		`update chores set created_at = now() - make_interval(secs => $2) where id = $1`,
		c.ID, int64((15 * 24 * time.Hour).Seconds()))
	require.NoError(t, err)

	return c
}

// seedSentPrompt records a prompt carrying the chore and marks it sent with
// messageID, exactly as the applier and scheduler do after a real Chat.Send.
// A numeric messageID is required — ParseAction's pattern never matches
// anything else, so a tap can never resolve back to a prompt seeded with a
// non-numeric id.
func seedSentPrompt(t *testing.T, store *squirrel.Store, personID int64, kind, messageID string, c squirrel.Chore) int64 {
	t.Helper()
	ctx := context.Background()

	id, err := store.RecordPrompt(ctx, personID, "9", kind, time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, messageID, time.Now()))
	return id
}

// webhookURL is where the Campfire transport mounts its inbound handler.
func webhookURL(s *boot.Squirrel) string {
	return fmt.Sprintf("http://127.0.0.1:%d/transports/campfire", s.Port())
}
