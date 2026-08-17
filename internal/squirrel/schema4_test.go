//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Two dated prompts of different kinds on one date must coexist — that is the
// whole reason this index changed.
func TestOneOfEachKindPerDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	today := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)

	for _, kind := range []string{"nudge", "evening"} {
		_, err := store.Pool().Exec(ctx, `
			insert into prompts (person_id, conversation_id, kind, sent_at, sent_for_date)
			values ($1, '9', $2, now(), $3)`, p, kind, today)
		require.NoError(t, err, kind)
	}
}

// And a second of the SAME kind on one date must still be refused — that is
// the guarantee the index existed for in the first place.
func TestStillOnlyOneNudgePerDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	today := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)

	_, err := store.Pool().Exec(ctx, `
		insert into prompts (person_id, conversation_id, kind, sent_at, sent_for_date)
		values ($1, '9', 'nudge', now(), $2)`, p, today)
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx, `
		insert into prompts (person_id, conversation_id, kind, sent_at, sent_for_date)
		values ($1, '9', 'nudge', now(), $2)`, p, today)
	require.Error(t, err, "one nudge a day is the budget")
}
