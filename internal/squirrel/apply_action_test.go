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

// tapItem is what the transport produces for a tap.
func tapItem(p int64, messageID, value string, selected bool) squirrel.Item {
	return squirrel.Item{
		Transport:      "campfire",
		ConversationID: squirrel.Ptr("9"),
		PersonID:       squirrel.Ptr(p),
		RawText:        fmt.Sprintf("!action %s %s %t", messageID, value, selected),
		Payload:        []byte(`{"type":"action"}`),
		ReceivedAt:     time.Now(),
	}
}

func TestTapCompletesTheChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 15*24*time.Hour)

	id, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	// See the comment in TestApplyCompletesByPosition and TestUnTapRetracts:
	// the digest just marked sent above starts its own tolerance window, so
	// querying DueChores at time.Now() would read empty regardless of
	// whether the tap did anything. Checking "due" before the tap, at an
	// instant past that window, is what makes the "empty after" assertion
	// below mean the tap actually completed it rather than the tolerance
	// gate masking a no-op applyAction.
	past := time.Now().Add(oneWeek + time.Hour)
	due, err := store.DueChores(ctx, p, past)
	require.NoError(t, err)
	require.Len(t, due, 1, "backdated past its interval, it is due before the tap")

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", true), squirrel.Ptr(p)))

	due, err = store.DueChores(ctx, p, past)
	require.NoError(t, err)
	require.Empty(t, due, "completing it via a tap resets the clock")
	require.Empty(t, *got, "a tap posts no reply — the boost is the receipt")
}

// The payload has no event id, so a retried background job looks exactly like a
// second tap. The operation carries the guarantee instead.
func TestTwoIdenticalTapsCompleteOnce(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	item := tapItem(p, "1", "done:1", true)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	var live int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1 and retracted_at is null`,
		c.ID).Scan(&live))
	require.Equal(t, 1, live)
}

func TestUnTapRetracts(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 15*24*time.Hour)
	id, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", true), squirrel.Ptr(p)))
	require.NoError(t, a.Apply(ctx, tapItem(p, "1", "done:1", false), squirrel.Ptr(p)))

	// See the comment in TestApplyCompletesByPosition: the digest just marked
	// sent above starts its own tolerance window, which would otherwise
	// suppress the chore for oneWeek regardless of whether the retraction
	// reset the completion clock. Querying past that window is what makes
	// this assertion about the retraction rather than about the digest gate.
	due, err := store.DueChores(ctx, p, time.Now().Add(oneWeek+time.Hour))
	require.NoError(t, err)
	require.Len(t, due, 1, "the chore is overdue again")
}

func TestTapAgainstAnotherPersonsPromptDoesNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	a := owner(t, store)
	b, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)
	send, _ := recorder()

	theirs, err := store.UpsertChore(ctx, b, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, b, "9", "digest", time.Now(), nil, []squirrel.Chore{theirs})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	ap := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, ap.Apply(ctx, tapItem(a, "1", "done:1", true), squirrel.Ptr(a)))

	var n int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1`, theirs.ID).Scan(&n))
	require.Zero(t, n, "the prompt lookup is scoped by person")
}

func TestUndefineTapDeactivatesTheChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	c, err := store.UpsertChore(ctx, p, "i have a headache", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, p, "9", "define", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "9", time.Now()))

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "9", "undefine:1", true), squirrel.Ptr(p)))

	active, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, active)
}

// DefinedMessage uses selection_mode "single", so deselecting the correction
// button delivers "selected: false" — not a second tap on a different
// button. Before the fix, applyAction's "undefine" case deactivated the
// chore regardless of in.Selected, so an un-select (which Campfire can and
// does send, e.g. when the retained selection is cleared) silently dropped a
// chore nobody asked to drop. "selected: true" is the only undefine tap that
// should act; anything else is a no-op, same as an unselected "done".
func TestUndefineTapRequiresSelectedTrue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	c, err := store.UpsertChore(ctx, p, "i have a headache", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, p, "9", "define", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "9", time.Now()))

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, tapItem(p, "9", "undefine:1", false), squirrel.Ptr(p)))

	active, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, active, 1, "an unselected undefine tap must not deactivate the chore")
}

// A person can type text that is byte-identical to what the transport writes
// for a real tap — ParseAction cannot tell them apart on text alone. This is
// what item.Payload's "type" field is for: an ordinary message's payload
// never claims to be "action", so it must fall through to the normal matcher
// and complete nothing, no matter how action-shaped its text looks.
//
// Asserted against the events table directly rather than DueChores: the
// digest just marked sent below starts its own tolerance window that would
// mask an incorrectly-recorded completion for a week regardless, and the
// point here is specifically whether a completion event was recorded at all.
func TestTapShapedTextWithoutActionPayloadDoesNotComplete(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, _ := recorder()

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 15*24*time.Hour)

	id, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "1", time.Now()))

	item := squirrel.Item{
		Transport:      "campfire",
		ConversationID: squirrel.Ptr("9"),
		PersonID:       squirrel.Ptr(p),
		RawText:        "!action 1 done:1 true",
		Payload:        []byte(`{"type":"message"}`),
		ReceivedAt:     time.Now(),
	}

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	var n int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1`, c.ID).Scan(&n))
	require.Zero(t, n, "a lookalike message payload must not complete the chore")
}
