//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// End to end: a photograph posted to the room is kept, and the room is told
// what was and was not kept about it.
//
// The unit tests either side of this one prove the matcher and the wording.
// This is the wiring — that a plain capture, which normally answers with
// nothing at all because the boost is the receipt, answers with a sentence
// for this one shape. Without the branch in replyFor, nothing is sent and
// this fails; that is what it is for.
func TestAPhotographFromTheRoomIsKeptAndSaidSo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	item := itemOf("IMG_5991.jpeg")
	item.PersonID = squirrel.Ptr(p)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	require.Len(t, *got, 1, "the room was told nothing about a note it cannot read")
	require.Contains(t, (*got)[0].text, "Kept")
	require.Contains(t, (*got)[0].text, "IMG_5991.jpeg")
	require.Contains(t, (*got)[0].text, "the camera on the pile")
}

// And an ordinary thought still answers with nothing, because the boost is the
// receipt and a second acknowledgement would be noise on every note.
func TestAnOrdinaryCaptureStillSaysNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	item := itemOf("the boiler makes a noise")
	item.PersonID = squirrel.Ptr(p)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	require.Empty(t, *got, "an ordinary capture answered with words")
}
