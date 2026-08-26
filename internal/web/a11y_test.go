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
