//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func countPeople(t *testing.T, store *squirrel.Store) int {
	t.Helper()
	var n int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from people`).Scan(&n))
	return n
}

func countIdentities(t *testing.T, store *squirrel.Store) int {
	t.Helper()
	var n int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from identities`).Scan(&n))
	return n
}

func TestSeedOwnerCreatesPersonAndIdentities(t *testing.T) {
	store := withStore(t)

	id, err := store.SeedOwner(context.Background(), "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)
	require.Positive(t, id)
	require.Equal(t, 1, countPeople(t, store))
	require.Equal(t, 1, countIdentities(t, store))
}

// Seeding is declarative: every boot reconciles to configuration.
func TestSeedOwnerIsIdempotent(t *testing.T) {
	store := withStore(t)
	seeds := []squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}}

	first, err := store.SeedOwner(context.Background(), "ronald", seeds)
	require.NoError(t, err)
	second, err := store.SeedOwner(context.Background(), "ronald", seeds)
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.Equal(t, 1, countPeople(t, store))
	require.Equal(t, 1, countIdentities(t, store))
}

func TestSeedOwnerAttachesASecondTransport(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	_, err := store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)

	id, err := store.SeedOwner(ctx, "ronald", []squirrel.IdentitySeed{
		{Transport: "campfire", ExternalID: "1"},
		{Transport: "matrix", ExternalID: "@me:example"},
	})
	require.NoError(t, err)

	require.Equal(t, 1, countPeople(t, store))
	require.Equal(t, 2, countIdentities(t, store))

	resolved, err := store.ResolvePerson(ctx, "matrix", squirrel.Ptr("@me:example"))
	require.NoError(t, err)
	require.Equal(t, id, *resolved)
}

func TestResolvePersonFindsASeededIdentity(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	id, err := store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)

	got, err := store.ResolvePerson(ctx, "campfire", squirrel.Ptr("1"))
	require.NoError(t, err)
	require.Equal(t, id, *got)
}

func TestResolvePersonReturnsNilForUnknown(t *testing.T) {
	store := withStore(t)
	got, err := store.ResolvePerson(context.Background(), "campfire", squirrel.Ptr("999"))
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestResolvePersonReturnsNilForNilID(t *testing.T) {
	store := withStore(t)
	got, err := store.ResolvePerson(context.Background(), "campfire", nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestResolvePersonKeepsTransportsApart(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	_, err := store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: "1"}})
	require.NoError(t, err)

	got, err := store.ResolvePerson(ctx, "matrix", squirrel.Ptr("1"))
	require.NoError(t, err)
	require.Nil(t, got)
}

// The test that keeps the guard from being decorative: auto-creating a person
// on first sight would re-admit whoever the allowlist turned away.
func TestResolvePersonNeverCreatesAnyone(t *testing.T) {
	store := withStore(t)

	_, err := store.ResolvePerson(context.Background(), "campfire", squirrel.Ptr("999"))
	require.NoError(t, err)
	require.Zero(t, countPeople(t, store))
	require.Zero(t, countIdentities(t, store))
}
