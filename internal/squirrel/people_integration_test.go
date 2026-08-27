//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// An unknown sub becomes a person, which is a departure from ResolvePerson's
// rule and a deliberate one: Authentik's group binding is the gate now.
func TestAnUnknownSubBecomesAPerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	id, err := store.PersonForLogin(ctx, "sub-new", "someone")
	require.NoError(t, err)
	require.NotZero(t, id)

	again, err := store.PersonForLogin(ctx, "sub-new", "someone")
	require.NoError(t, err)
	require.Equal(t, id, again, "a second login made a second person")
}

// The trap from the design's section 4: a capture resolves its owner from a
// sender string through the spool, not from the session. A person with only an
// oidc identity spools notes belonging to nobody.
func TestALoginAlsoGetsAScreenIdentity(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	id, err := store.PersonForLogin(ctx, "sub-new", "someone")
	require.NoError(t, err)

	sub := "sub-new"
	byScreen, err := store.ResolvePerson(ctx, squirrel.ScreenTransport, &sub)
	require.NoError(t, err)
	require.NotNil(t, byScreen, "a capture typed by this person would belong to nobody")
	require.Equal(t, id, *byScreen)

	byOIDC, err := store.ResolvePerson(ctx, squirrel.OIDCTransport, &sub)
	require.NoError(t, err)
	require.NotNil(t, byOIDC)
	require.Equal(t, id, *byOIDC)
}

// A sub already seeded against the owner resolves to the owner rather than
// making a second person. This is what keeps your pile yours.
func TestASeededSubKeepsItsPerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	seeded, err := store.SeedOwner(ctx, "ronald", []squirrel.IdentitySeed{
		{Transport: squirrel.OIDCTransport, ExternalID: "sub-ronald"},
		{Transport: squirrel.ScreenTransport, ExternalID: "sub-ronald"},
	})
	require.NoError(t, err)

	got, err := store.PersonForLogin(ctx, "sub-ronald", "ronald")
	require.NoError(t, err)
	require.Equal(t, seeded, got, "logging in made a second person for the owner")
}

func TestTwoSubsAreTwoPeople(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	one, err := store.PersonForLogin(ctx, "sub-a", "a")
	require.NoError(t, err)
	two, err := store.PersonForLogin(ctx, "sub-b", "b")
	require.NoError(t, err)

	require.NotEqual(t, one, two)
}

// A handle collision does not merge two people. Authentik usernames are not
// unique forever and a person is its sub.
func TestTwoSubsWithOneHandleAreStillTwoPeople(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	one, err := store.PersonForLogin(ctx, "sub-a", "same")
	require.NoError(t, err)
	two, err := store.PersonForLogin(ctx, "sub-b", "same")
	require.NoError(t, err)

	require.NotEqual(t, one, two, "two accounts sharing a display name became one person")
}

// An account with no username still becomes somebody, rather than a person
// whose handle is the empty string.
func TestALoginWithNoUsernameStillBecomesSomebody(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()

	id, err := store.PersonForLogin(ctx, "sub-anon", "")
	require.NoError(t, err)
	require.NotZero(t, id)

	var handle string
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select handle from people where id = $1`, id).Scan(&handle))
	require.Contains(t, handle, "someone", "an account with no username got a nameless handle")
}
