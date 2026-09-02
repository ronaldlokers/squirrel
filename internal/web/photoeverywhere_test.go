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

// The shelf is a message rather than a screen since 25 August 2026, so its
// photograph is checked where the message is drawn.
func TestAPhotographIsShownOnTheShelf(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		withPhoto(1, "the tax letter", squirrel.ItemKept, squirrel.ItemNote)}}
	f.appended = nil
	fDrew := drewIn(t, f, "kept")

	require.Contains(t, string(fDrew[len(fDrew)-1].Shown), `"photo":"/photo/1"`,
		"the shelf drops the photograph")
}

// The tasks are a message rather than a screen since 24 August 2026, so their
// photograph is checked where the message is drawn.
func TestATaskInTheThreadKeepsItsPhotograph(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		withPhoto(1, "the tax letter", squirrel.ItemOpen, squirrel.ItemTask)}}

	// The card draws the smaller copy and links to the whole one, both by the
	// note's id. See thumb_test.go for why the card asks for the copy.
	body := opened(t, f, "tasks")
	require.Contains(t, body, `href="/?open=1"`, "the strip is not a way in to the photograph")
}

// And on the screen for things you cannot act on, which needed the store to
// carry the photograph at all — its rows never selected the column.
func TestAPhotographSurvivesBeingSetAside(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{{
		ID: 20, Text: "chase the vet", State: squirrel.ItemWaiting,
		Because: "the vet", Kind: squirrel.ItemTask, PhotoName: "letter.jpg",
	}}}
	f.appended = nil
	fDrew := drewIn(t, f, "held")

	require.Contains(t, string(fDrew[len(fDrew)-1].Shown), `"photo":"/photo/20"`,
		"setting something aside loses its photograph")
}

// By the row's id, never by the file's name. The name is the one string in
// this product that becomes a path.
func TestAPhotographIsNeverAddressedByItsFilename(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		withPhoto(1, "the tax letter", squirrel.ItemKept, squirrel.ItemNote)}}
	f.appended = nil
	fDrew := drewIn(t, f, "kept")

	require.NotContains(t, string(fDrew[len(fDrew)-1].Shown), "letter.jpg")
}
