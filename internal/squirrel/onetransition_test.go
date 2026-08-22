//go:build integration

package squirrel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Completing a task means the same thing wherever the tap came from.
//
// It meant the same thing because two places said it in the same words: the
// chat tap wrote the state and recorded the answer by hand, and `Did` — which
// exists to know what completing an offer means — wrote exactly that pair a
// few files away. Two descriptions of one transition, correct today, with
// nothing keeping the second correct tomorrow.
//
// This is inside the package because the half that is easy to forget is the
// one with no reader: `offers` is deliberately write-only from outside, since
// a function that could count answers is a sentence about the person. So the
// test asks the table directly rather than inventing an API nobody should
// have.
func TestCompletingAnOfferedTaskRecordsBothHalves(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)

	var personID int64
	require.NoError(t, store.pool.QueryRow(ctx, `
		insert into people (handle) values ('both-halves')
		on conflict (handle) do update set handle = excluded.handle
		returning id`).Scan(&personID))

	var itemID int64
	require.NoError(t, store.pool.QueryRow(ctx, `
		insert into items (transport, raw_text, received_at, person_id, kind, payload)
		values ('campfire', 'ring the vet', now(), $1, 'task', '{}') returning id`,
		personID).Scan(&itemID))

	require.NoError(t, store.Did(ctx, personID,
		Offer{Kind: OfferTask, RefID: itemID}, time.Now()))

	var state string
	require.NoError(t, store.pool.QueryRow(ctx,
		`select state from items where id = $1`, itemID).Scan(&state))
	require.Equal(t, string(ItemDone), state, "the state half did not happen")

	var answers int
	require.NoError(t, store.pool.QueryRow(ctx, `
		select count(*) from offers
		where person_id = $1 and kind = 'task' and ref_id = $2 and answer = 'did'`,
		personID, itemID).Scan(&answers))
	require.Equal(t, 1, answers, "the answer half did not happen")
}

// And the chat tap goes through it, rather than saying the same thing again in
// its own words.
//
// This is the wiring rather than the function: the existing tap tests check
// the state moved, which stays true whether or not the answer is recorded, so
// none of them notices the two halves coming apart. This one does.
func TestATappedTaskFromChatGoesThroughDid(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)

	var personID int64
	require.NoError(t, store.pool.QueryRow(ctx, `
		insert into people (handle) values ('tapped-task')
		on conflict (handle) do update set handle = excluded.handle
		returning id`).Scan(&personID))
	_, err := store.pool.Exec(ctx, `delete from offers where person_id = $1`, personID)
	require.NoError(t, err)

	var itemID int64
	require.NoError(t, store.pool.QueryRow(ctx, `
		insert into items (transport, raw_text, received_at, person_id, kind, payload)
		values ('campfire', 'send the deposit form', now(), $1, 'task', '{}') returning id`,
		personID).Scan(&itemID))

	prompt, err := store.RecordPromptLines(ctx, personID, "9", "now", time.Now(), nil,
		[]LineRef{{ItemID: &itemID}})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, prompt, "1", time.Now()))

	a := NewApplier(store, func(context.Context, string, string) error { return nil }, Chat{}, nil)
	require.NoError(t, a.Apply(ctx, Item{
		Transport: "campfire", ConversationID: Ptr("9"), PersonID: &personID,
		RawText: "!action 1 done:1 true", Payload: []byte(`{"type":"action"}`),
		ReceivedAt: time.Now(),
	}, &personID))

	var state string
	require.NoError(t, store.pool.QueryRow(ctx,
		`select state from items where id = $1`, itemID).Scan(&state))
	require.Equal(t, string(ItemDone), state, "the tap did not complete the task")

	var answers int
	require.NoError(t, store.pool.QueryRow(ctx, `
		select count(*) from offers
		where person_id = $1 and kind = 'task' and ref_id = $2 and answer = 'did'`,
		personID, itemID).Scan(&answers))
	require.Equal(t, 1, answers,
		"the tap moved the state without recording the answer — the two halves came apart")
}
