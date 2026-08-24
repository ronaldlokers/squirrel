//go:build integration

package boot_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/boot"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// testPresenceSecret is the token every test that wants the arrival route
// mounted configures PRESENCE_SECRET as, and sends back as X-Squirrel-Token.
const testPresenceSecret = "presence-test-secret"

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

	requests = func() []campfireRequest {
		mu.Lock()
		defer mu.Unlock()
		out := make([]campfireRequest, len(got))
		copy(out, got)
		return out
	}

	// Tracked so campfireStubSawText can reach a stub's traffic even for a
	// test (bootWithStore's callers, mainly) that never sees this func
	// itself — envFor stands the stub up internally and only ever hands its
	// URL down into the environment. Tests in this package never run in
	// parallel (make test-integration passes -p 1 deliberately, see the
	// Makefile), so the single most-recently-created stub is unambiguous for
	// whichever test is currently running.
	lastCampfireStubMu.Lock()
	lastCampfireStubRequests = requests
	lastCampfireStubMu.Unlock()

	return srv.URL, requests
}

var (
	lastCampfireStubMu       sync.Mutex
	lastCampfireStubRequests func() []campfireRequest
)

// campfireStubSawText reports whether any request the most recently created
// campfire stub received so far carries text somewhere in its body — JSON or
// plain text, matching however sendMessage happened to have serialized it.
// It exists because tests built on bootWithStore never see the stub's
// requests func directly: envFor stands the stub up itself and only threads
// its URL through CAMPFIRE_BASE_URL.
func campfireStubSawText(t *testing.T, text string) bool {
	t.Helper()
	lastCampfireStubMu.Lock()
	requests := lastCampfireStubRequests
	lastCampfireStubMu.Unlock()
	if requests == nil {
		return false
	}
	for _, r := range requests() {
		if strings.Contains(string(r.body), text) {
			return true
		}
	}
	return false
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
//
// PRESENCE_SECRET is set here too, so every test built on this helper gets a
// live arrival route at presenceURL(s) without asking for it — a nudge test
// is the only reason bootWithStore needed this, but its one other caller,
// TestBootAppliesATap, still boots exactly the same server it did before
// this task, just with one extra mounted route it never touches.
//
// EVENING_AT is pinned to 23:59 for the same reason: Scheduler.Run calls
// Once synchronously before its first tick, and Once makes its own "last
// attempt at today's nudge" whenever the wall clock is already past
// EVENING_AT — true for any test run late enough in the day, and always true
// once EVENING_AT is pinned earlier than that for some other reason. Left
// unpinned, that startup Once can win the race against seedOverdueChore and
// claim the day's nudge slot for the same chore a nudge test seeds, sending
// it independently of whatever the arrival webhook does. Confirmed by trial:
// with the presence route wired to a no-op OnArrive, TestBootNudgesOnArrival
// still passed 1 run in 5 — campfireStubSawText found "vacuum" from that
// startup send, not from the arrival. Pinning EVENING_AT past any wall-clock
// time a test could run at keeps Once's own local.Before(threshold) guard
// skipping it for the whole test, so a nudge landing in the stub can only be
// the one the test itself triggered.
//
// PRESENCE_DELAY is pinned short: config.PresenceDelay now defaults to two
// minutes in production (the "you have a coat on" delay PresenceOptions'
// doc comment describes), which would blow TestBootNudgesOnArrival's 15s
// Eventually budget many times over. One second keeps the arrival-to-nudge
// gap real without making the test slow.
// daytimeZone is a real timezone in which it is currently the middle of the
// day.
//
// Nudges have quiet hours now, read in the scheduler's own location — so an
// end-to-end test that pings a real socket and expects a nudge back would
// otherwise be a test about what time the suite runs at. It cannot pass a
// fixed clock in, because the whole point of these tests is that they go
// through the real path; what it can do is say which part of the world the
// person is notionally in, which is a setting the product already has.
//
// The Etc/GMT names are guaranteed to exist in tzdata and their signs are
// inverted by POSIX convention — Etc/GMT+5 is UTC-5. Which one is picked does
// not matter; that one of them is mid-morning always does.
func daytimeZone(t *testing.T) string {
	t.Helper()
	for offset := -11; offset <= 12; offset++ {
		name := fmt.Sprintf("Etc/GMT%+d", -offset)
		loc, err := time.LoadLocation(name)
		if err != nil {
			continue
		}
		if h := time.Now().In(loc).Hour(); h >= 9 && h <= 16 {
			return name
		}
	}
	t.Fatal("no timezone is currently mid-morning, which cannot happen")
	return ""
}

func bootWithStore(t *testing.T) (*boot.Squirrel, *squirrel.Store) {
	t.Helper()
	store := withStore(t)
	s := boots(t, envFor(t, map[string]string{
		"CAMPFIRE_CONVERSATION_ID": "9",
		"PRESENCE_SECRET":          testPresenceSecret,
		"EVENING_AT":               "23:59",
		"PRESENCE_DELAY":           "1s",
		"DIGEST_TZ":                daytimeZone(t),
	}))
	return s, store
}

// bootWithoutPresence boots exactly like bootWithStore, minus PRESENCE_SECRET
// — so MountPresence's own refusal to mount with an empty secret leaves
// presenceURL(s) unrouted, a plain 404 like any other path nothing answers.
func bootWithoutPresence(t *testing.T) *boot.Squirrel {
	t.Helper()
	withStore(t)
	return boots(t, envFor(t, map[string]string{"CAMPFIRE_CONVERSATION_ID": "9"}))
}

// presenceURL is where the arrival webhook is mounted — config.go's
// PresencePath default, which boot.go never overrides.
func presenceURL(s *boot.Squirrel) string {
	return fmt.Sprintf("http://127.0.0.1:%d/hooks/home", s.Port())
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

// screenURL is the screen. It was the deck at /pile; the deck came out on
// 24 August 2026 and the conversation at the root is the whole of it.
//
// Old comment, kept for the reader who greps for it: the deck kept its own
// URL so that an installed app's start_url survived the move.
func screenURL(s *boot.Squirrel) string {
	return fmt.Sprintf("http://127.0.0.1:%d/", s.Port())
}
