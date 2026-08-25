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
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `id="say"`)
	require.Contains(t, body, `aria-live="polite"`)

	stage := body[strings.Index(body, `<main`):]
	require.NotContains(t, stage, `id="say"`, "inside the stage it would be swapped away")
}

// The key badges and the stack behind the card were tested here. Both were
// the deck's markup and neither renders anywhere since it came out, so both
// assertions had been comparing zero to zero. Deleted on 25 August 2026 rather
// than retargeted: a test that cannot fail is worse than no test, because it
// reads like cover.
//
// The live edge's own keys are proved in chorekeys_test.go, which does press
// them.

// TestTheCardSaysWhatItIs went with the deck's card, whose region label named
// the screen it was the whole of. A card in a turn is one thing among many in
// a conversation, and what names it is the <h2> the turn carries — pinned by
// TestATurnThatOpensAPlaceCarriesAHeading.
