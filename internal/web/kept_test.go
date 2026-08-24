package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func aShelf() *fakeStore {
	return &fakeStore{items: []squirrel.Item{
		note(1, "meter reading 48213", squirrel.ItemKept),
		note(2, "buy milk", squirrel.ItemOpen),
		note(3, "boiler service code is 4471", squirrel.ItemKept),
		note(4, "ring the dentist", squirrel.ItemDone),
	}}
}

func TestTheShelfHoldsOnlyWhatWasKept(t *testing.T) {
	body := mounted(t, aShelf()).call(t, "GET", "/kept", nil).Body.String()

	require.Contains(t, body, "meter reading 48213")
	require.Contains(t, body, "boiler service code is 4471")
	require.NotContains(t, body, "buy milk", "an open note is in the pile, not on the shelf")
	require.NotContains(t, body, "ring the dentist", "a done note is not reference")
}

// The shelf exists because a kept note had nowhere to be read back. What it
// offers is the way to change your mind, and nothing else: a kept note was
// never going to be done, so DONE here would answer the wrong question.
func TestTheShelfOffersOnlyTheWayBack(t *testing.T) {
	body := mounted(t, aShelf()).call(t, "GET", "/kept", nil).Body.String()

	require.Contains(t, body, `value="open"`)
	require.NotContains(t, body, `value="done"`)
	require.NotContains(t, body, `value="drop"`)
	require.NotContains(t, body, `value="keep"`, "they are already kept")
}

func TestTheShelfNeverEmitsACount(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 12; i++ {
		items = append(items, note(i, "kept thing", squirrel.ItemKept))
	}
	body := strings.ToLower(mounted(t, &fakeStore{items: items}).call(t, "GET", "/kept", nil).Body.String())

	for _, total := range []string{"12 ", "of 12", "(12)", "1 of "} {
		require.NotContains(t, body, total)
	}
}

func TestTheEmptyShelfDoesNotInstruct(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/kept", nil).Body.String()

	require.Contains(t, body, "nothing on the shelf yet")
	for _, forbidden := range []string{"well done", "start by", "you should", "try to"} {
		require.NotContains(t, strings.ToLower(body), forbidden)
	}
}

// The shelf is reachable from the pile, which is where you would look for it.
//
// It hung off the deck's foot, at the two moments with nothing left to triage.
// The deck went, so it hangs off the pile's turn instead — and without a way
// here it would be reachable from nowhere at all, which is the bug the mood
// history had for an afternoon.
func TestThePileOffersTheShelf(t *testing.T) {
	require.Contains(t, opened(t, aShelf(), "pile"), `href="/kept"`)
}
