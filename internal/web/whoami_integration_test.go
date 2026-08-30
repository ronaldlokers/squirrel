//go:build integration

package web

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// What the gate says about you, kept — and the half of it that must survive a
// provider that stops saying it.
func TestRememberingWhoYouAre(t *testing.T) {
	store := realStore(t)
	ctx := context.Background()
	personID, err := store.PersonForLogin(ctx, "sub-mine", "ronald")
	require.NoError(t, err)

	// Before the gate ever said anything, the handle stands in for a name.
	was, err := store.WhoIs(ctx, personID)
	require.NoError(t, err)
	require.Empty(t, was.Name, "the uniquified handle leaked onto the screen as a name")
	require.Contains(t, was.Handle, "ronald", "the handle is still the handle")
	require.False(t, was.HasFace)

	require.NoError(t, store.RememberPerson(ctx, personID, "Ronald Lokers", []byte("PNG"), "image/png"))
	was, err = store.WhoIs(ctx, personID)
	require.NoError(t, err)
	require.Equal(t, "Ronald Lokers", was.Name)
	require.True(t, was.HasFace)

	face, kind, found, err := store.PersonFace(ctx, personID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("PNG"), face)
	require.Equal(t, "image/png", kind)

	// A later sign-in where the provider sends neither must not erase what it
	// sent last time: losing your face because Authentik was reconfigured is
	// not a thing to store.
	require.NoError(t, store.RememberPerson(ctx, personID, "", nil, ""))
	was, err = store.WhoIs(ctx, personID)
	require.NoError(t, err)
	require.Equal(t, "Ronald Lokers", was.Name, "a silent login erased the name")
	require.True(t, was.HasFace, "a silent login erased the picture")

	// The handle is identity and is never repointed at a display name.
	require.Contains(t, was.Handle, "ronald")
	require.NotEqual(t, was.Name, was.Handle, "the handle became the name")
}
