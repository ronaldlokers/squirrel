//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Deleting an appointment must never delete the owner's words. The note
// returns to the pile instead, which is what happens when it is over anyway.
func TestDeletingAFixedPointKeepsTheNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: time.Now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)
	id := taskOf(t, store, p, "the referral letter")

	_, err = store.Pool().Exec(ctx, `update items set moment_id = $1 where id = $2`, m.ID, id)
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx, `delete from moments where id = $1`, m.ID)
	require.NoError(t, err)

	var moment *int64
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select moment_id from items where id = $1`, id).Scan(&moment))
	require.Nil(t, moment, "the words survive the appointment")
}
