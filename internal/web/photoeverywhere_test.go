package web

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func withPhoto(id int64, text string, state squirrel.ItemState, kind squirrel.ItemKind) squirrel.Item {
	it := note(id, text, state)
	it.Kind = kind
	it.PhotoName = "letter.jpg"
	it.PhotoType = "image/jpeg"
	return it
}

func TestATaskOnTheBoardKeepsItsPhotograph(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		withPhoto(1, "the tax letter", squirrel.ItemOpen, squirrel.ItemTask)}}

	body := opened(t, f, "tasks")
	require.Contains(t, body, `href="/?open=1"`, "the strip is not a way in to the photograph")
}
