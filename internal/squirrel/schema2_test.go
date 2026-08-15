//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMigration0002CreatesTheTables(t *testing.T) {
	store := withStore(t)

	for _, name := range []string{"chores", "events", "prompts", "prompt_lines"} {
		var exists bool
		require.NoError(t, store.Pool().QueryRow(context.Background(),
			`select exists (select 1 from information_schema.tables
			   where table_schema = 'public' and table_name = $1)`, name,
		).Scan(&exists))
		require.True(t, exists, name)
	}
}

// "Vacuum" must collide with "vacuum" or upsert-by-name is not safe.
func TestChoreNameUniquenessIsCaseInsensitive(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	personID, err := store.SeedOwner(ctx, "ronald", nil)
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx,
		`insert into chores (person_id, name, interval_seconds, tolerance_seconds)
		 values ($1, 'vacuum', 1209600, 604800)`, personID)
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx,
		`insert into chores (person_id, name, interval_seconds, tolerance_seconds)
		 values ($1, 'Vacuum', 1209600, 604800)`, personID)
	require.Error(t, err)
}

// Only one digest per person per day can exist, whatever the process does.
func TestOneDigestPerDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	personID, err := store.SeedOwner(ctx, "ronald", nil)
	require.NoError(t, err)
	day := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	_, err = store.Pool().Exec(ctx,
		`insert into prompts (person_id, conversation_id, kind, sent_at, sent_for_date)
		 values ($1, '9', 'digest', now(), $2)`, personID, day)
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx,
		`insert into prompts (person_id, conversation_id, kind, sent_at, sent_for_date)
		 values ($1, '9', 'digest', now(), $2)`, personID, day)
	require.Error(t, err)
}

// On-demand queries have no date and must not collide with each other.
func TestManyQueryPromptsCoexist(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	personID, err := store.SeedOwner(ctx, "ronald", nil)
	require.NoError(t, err)

	for range 3 {
		_, err = store.Pool().Exec(ctx,
			`insert into prompts (person_id, conversation_id, kind, sent_at)
			 values ($1, '9', 'query', now())`, personID)
		require.NoError(t, err)
	}

	var n int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from prompts where kind = 'query'`).Scan(&n))
	require.Equal(t, 3, n)
}
