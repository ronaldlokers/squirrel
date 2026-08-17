//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration0005And0006AddTheColumns(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	for _, c := range []struct{ table, column string }{
		{"events", "retracted_at"},
		{"prompts", "external_message_id"},
	} {
		var n int
		require.NoError(t, store.Pool().QueryRow(ctx, `
			select count(*) from information_schema.columns
			 where table_schema = 'public' and table_name = $1 and column_name = $2`,
			c.table, c.column).Scan(&n))
		require.Equal(t, 1, n, "%s.%s", c.table, c.column)
	}
}

func TestPromptMessageIDIsUniqueWhenPresent(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.Pool().Exec(ctx, `
		insert into prompts (person_id, conversation_id, kind, sent_at, external_message_id)
		values ($1, '9', 'digest', now(), 'm-1')`, p)
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx, `
		insert into prompts (person_id, conversation_id, kind, sent_at, external_message_id)
		values ($1, '9', 'query', now(), 'm-1')`, p)
	require.Error(t, err, "one Campfire message is one prompt")

	// Null is not a value: two prompts that never got a message id coexist.
	for range 2 {
		_, err = store.Pool().Exec(ctx, `
			insert into prompts (person_id, conversation_id, kind, sent_at)
			values ($1, '9', 'query', now())`, p)
		require.NoError(t, err)
	}
}
