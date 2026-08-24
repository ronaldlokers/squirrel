package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Live search replaces the whole stage, so anything inside it that was meant
// to announce a change is replaced along with it — a new node with aria-live
// on it says nothing. The region has to outlive the swap, which means it lives
// in the layout and the script writes to it.
func TestThereIsALiveRegionOutsideTheStage(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/kept", nil).Body.String()

	require.Contains(t, body, `id="say"`)
	require.Contains(t, body, `aria-live="polite"`)

	stage := body[strings.Index(body, `<main`):]
	require.NotContains(t, stage, `id="say"`, "inside the stage it would be swapped away")
}

// The key badges are a visual reminder, not part of the button's name: a
// screen reader saying "DONE D" is reading the poster on the wall.
func TestTheKeyBadgesAreNotReadOut(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/kept", nil).Body.String()

	require.Equal(t, strings.Count(body, `class="key"`),
		strings.Count(body, `class="key" aria-hidden="true"`),
		"every key badge is hidden from the accessibility tree")
}

// The stack behind the card is a picture of "there is more". It says that in
// words on the deck itself, so the cards are decoration twice over.
func TestTheStackIsDecorative(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 3; i++ {
		items = append(items, note(i, "note", squirrel.ItemOpen))
	}
	body := mounted(t, &fakeStore{items: items}).call(t, "GET", "/kept", nil).Body.String()

	require.Equal(t, strings.Count(body, `class="behind"`),
		strings.Count(body, `class="behind" aria-hidden="true"`))
}

// TestTheCardSaysWhatItIs went with the deck's card, whose region label named
// the screen it was the whole of. A card in a turn is one thing among many in
// a conversation, and what names it is the <h2> the turn carries — pinned by
// TestATurnThatOpensAPlaceCarriesAHeading.
