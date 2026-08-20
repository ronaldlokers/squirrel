//go:build integration

package squirrel_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A photograph, all the way from the volume to the pile.

func TestAWordlessPhotographIsStillInThePile(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	photos, err := squirrel.OpenPhotos(t.TempDir())
	require.NoError(t, err)
	name, err := photos.Keep(strings.NewReader("jpegbytes"), "image/jpeg")
	require.NoError(t, err)

	spool, err := squirrel.OpenSpool(t.TempDir())
	require.NoError(t, err)

	sender := "ronald"
	_, err = store.SeedOwner(ctx, "ronald",
		[]squirrel.IdentitySeed{{Transport: squirrel.ScreenTransport, ExternalID: sender}})
	require.NoError(t, err)

	_, err = spool.Write(squirrel.Capture{
		Transport: squirrel.ScreenTransport, SenderID: &sender,
		Text: "", Payload: []byte(squirrel.ScreenCapture), ReceivedAt: time.Now(),
		PhotoName: name, PhotoType: "image/jpeg",
	})
	require.NoError(t, err)

	squirrel.NewDrain(squirrel.DrainOptions{Store: store, Spool: spool}).Once(ctx)

	// The whole point of has_content: a note with no words and a photograph is
	// not a blank row, and every list that used to hide blank rows now knows
	// the difference.
	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, items, 1, "a wordless photograph vanished from the pile")
	require.Empty(t, items[0].RawText)
	require.Equal(t, name, items[0].PhotoName)
	require.Equal(t, "image/jpeg", items[0].PhotoType)
}

// A row with neither words nor a photograph is still hidden, which is what the
// filter was for in the first place.
func TestATrulyBlankRowIsStillHidden(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	_, err := store.InsertItem(ctx, squirrel.Item{
		PersonID: &p, RawText: "", ReceivedAt: time.Now(),
		Transport: "campfire", ExternalID: squirrel.Ptr("blank-1"),
		Payload: []byte(`{}`),
	})
	require.NoError(t, err)

	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, items)
}

// And it comes back by id, with everything the screen needs to serve it.
func TestAPhotographIsFoundByTheNotesID(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	id, err := store.InsertItemReturningID(ctx, squirrel.Item{
		PersonID: &p, RawText: "the tax letter", ReceivedAt: time.Now(),
		Transport: squirrel.ScreenTransport, Payload: []byte(squirrel.ScreenCapture),
		PhotoName: "abc123.jpg", PhotoType: "image/jpeg",
	})
	require.NoError(t, err)

	it, found, err := store.ItemByID(ctx, p, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "abc123.jpg", it.PhotoName)
	require.Equal(t, "image/jpeg", it.PhotoType)

	// And not by somebody else's.
	_, found, err = store.ItemByID(ctx, p+1000, id)
	require.NoError(t, err)
	require.False(t, found)
}
