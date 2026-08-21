package web

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A photograph goes where the note goes.
//
// It was drawn on the deck and nowhere else. So keeping a note, or deciding it
// was a task, made the picture vanish — and a note with no words and only a
// photograph, which is a perfectly good note here and most of the reason for
// having a camera, became a row that said nothing at all.
//
// One test per screen that shows a note, because the failure was silent and
// screen-shaped: each of these was written correctly and simply never asked
// for the field.

func withPhoto(id int64, text string, state squirrel.ItemState, kind squirrel.ItemKind) squirrel.Item {
	it := note(id, text, state)
	it.Kind = kind
	it.PhotoName = "letter.jpg"
	it.PhotoType = "image/jpeg"
	return it
}

func TestAPhotographIsShownWhereverTheNoteIs(t *testing.T) {
	for _, screen := range []struct {
		name, path string
		store      *fakeStore
	}{
		{"the pile", "/pile", &fakeStore{items: []squirrel.Item{
			withPhoto(1, "the tax letter", squirrel.ItemOpen, squirrel.ItemNote)}}},
		{"the shelf", "/kept", &fakeStore{items: []squirrel.Item{
			withPhoto(1, "the tax letter", squirrel.ItemKept, squirrel.ItemNote)}}},
		{"the tasks", "/tasks", &fakeStore{items: []squirrel.Item{
			withPhoto(1, "the tax letter", squirrel.ItemOpen, squirrel.ItemTask)}}},
		{"the archive", "/tasks/done", &fakeStore{items: []squirrel.Item{
			withPhoto(1, "the tax letter", squirrel.ItemDone, squirrel.ItemTask)}}},
		{"search", "/pile?q=tax", &fakeStore{items: []squirrel.Item{
			withPhoto(1, "the tax letter", squirrel.ItemOpen, squirrel.ItemNote)}}},
	} {
		t.Run(screen.name, func(t *testing.T) {
			body := mountedWithCamera(t, screen.store, &fakeSpool{}, &fakePhotos{}).
				call(t, "GET", screen.path, nil).Body.String()
			require.Contains(t, body, `src="/photo/1"`,
				"%s drops the photograph", screen.name)
		})
	}
}

// And on the screen for things you cannot act on, which needed the store to
// carry the photograph at all — its rows never selected the column.
func TestAPhotographSurvivesBeingSetAside(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{{
		ID: 20, Text: "chase the vet", State: squirrel.ItemWaiting,
		Because: "the vet", Kind: squirrel.ItemTask, PhotoName: "letter.jpg",
	}}}
	body := mountedWithCamera(t, f, &fakeSpool{}, &fakePhotos{}).
		call(t, "GET", "/held", nil).Body.String()

	require.Contains(t, body, `src="/photo/20"`,
		"setting something aside loses its photograph")
}

// By the row's id, never by the file's name. The name is the one string in
// this product that becomes a path.
func TestAPhotographIsNeverAddressedByItsFilename(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		withPhoto(1, "the tax letter", squirrel.ItemKept, squirrel.ItemNote)}}
	body := mountedWithCamera(t, f, &fakeSpool{}, &fakePhotos{}).
		call(t, "GET", "/kept", nil).Body.String()

	require.NotContains(t, body, "letter.jpg")
}
