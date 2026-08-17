//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

const (
	twoWeeks = 14 * 24 * time.Hour
	oneWeek  = 7 * 24 * time.Hour
	oneDay   = 24 * time.Hour
)

func owner(t *testing.T, store *squirrel.Store) int64 {
	t.Helper()
	id, err := store.SeedOwner(context.Background(), "ronald", nil)
	require.NoError(t, err)
	return id
}

func TestUpsertChoreCreatesAndUpdates(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	first, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.Equal(t, "vacuum", first.Name)

	second, err := store.UpsertChore(ctx, p, "Vacuum", 21*24*time.Hour, oneWeek)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "case-insensitive upsert")
	require.Equal(t, 21*24*time.Hour, second.Every)

	all, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, "vacuum", all[0].Name, "stored as first written")
}

// A new chore's first nudge is one interval away, which is what the
// confirmation message promises.
func TestNewChoreIsNotDueImmediately(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due)

	due, err = store.DueChores(ctx, p, time.Now().Add(twoWeeks+time.Hour))
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.Equal(t, 14, due[0].EveryDays)
}

// A `?` records a prompt_line for every active chore, due or not — that is
// what lets a bare number reply resolve later. Before the fix, baselineCTE's
// last_shown had no filter on prompts.kind, so that query prompt counted as a
// nudge and the tolerance gate hid an already-due chore until its tolerance
// window passed again. A "?" must never be able to suppress a real digest.
func TestQueryDoesNotSuppressTheNextDigest(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	vac, err := store.UpsertChore(ctx, p, "vacuum", oneWeek, oneDay)
	require.NoError(t, err)
	backdateChore(t, store, vac.ID, oneWeek+24*time.Hour)

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "backdated past its interval, it is due before the query")

	send, _ := recorder()
	require.NoError(t, squirrel.NewApplier(store, send, squirrel.Chat{}, nil).Apply(ctx, itemOf("?"), &p))

	due, err = store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "a query must not mark the chore as shown for the tolerance gate")
}

func TestDeactivateChoreHidesIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	require.NoError(t, store.DeactivateChore(ctx, c.ID))

	active, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, active)

	due, err := store.DueChores(ctx, p, time.Now().Add(twoWeeks+time.Hour))
	require.NoError(t, err)
	require.Empty(t, due)
}

// A tap is stored as an item whose raw_text is "!action <id> done:<n> <bool>"
// — the same items table a genuine capture lands in. Before the fix,
// CapturesSince had no case for that shape in matchFn, so it classified as
// IntentCapture and was echoed into the next digest's "Since yesterday" list,
// accumulating a line for every tap ever made. ParseAction is what the tap
// pipeline itself uses to recognise the shape, so filtering on it here is the
// one definition of what a tap looks like rather than a second one.
func TestCapturesSinceExcludesTaps(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	since := time.Now().Add(-time.Hour)

	_, err := store.InsertItem(ctx, squirrel.Item{
		Transport:      "campfire",
		ExternalID:     squirrel.Ptr("1"),
		ConversationID: squirrel.Ptr("9"),
		PersonID:       squirrel.Ptr(p),
		RawText:        "buy milk",
		Payload:        []byte(`{}`),
		ReceivedAt:     time.Now(),
	})
	require.NoError(t, err)

	_, err = store.InsertItem(ctx, squirrel.Item{
		Transport:      "campfire",
		ExternalID:     squirrel.Ptr("2"),
		ConversationID: squirrel.Ptr("9"),
		PersonID:       squirrel.Ptr(p),
		RawText:        "!action 451 done:1 true",
		Payload:        []byte(`{"type":"action"}`),
		ReceivedAt:     time.Now(),
	})
	require.NoError(t, err)

	texts, err := store.CapturesSince(ctx, p, since)
	require.NoError(t, err)
	require.Equal(t, []string{"buy milk"}, texts,
		"a tap must never be echoed into the digest's capture list")
}
