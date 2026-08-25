package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The shelf. A page until 25 August 2026 and a message since.

func aShelf() *fakeStore {
	return &fakeStore{items: []squirrel.Item{
		note(1, "meter reading 48213", squirrel.ItemKept),
		note(2, "buy milk", squirrel.ItemOpen),
		note(3, "boiler service code is 4471", squirrel.ItemKept),
		note(4, "ring the dentist", squirrel.ItemDone),
	}}
}

func TestTheShelfHoldsOnlyWhatWasKept(t *testing.T) {
	drew := drewFor(t, aShelf(), "kept")

	require.Contains(t, drew, "meter reading 48213")
	require.Contains(t, drew, "boiler service code is 4471")
	require.NotContains(t, drew, "buy milk", "an open note is in the pile, not on the shelf")
	require.NotContains(t, drew, "ring the dentist", "a done note is not reference")
}

// The shelf exists because a kept note had nowhere to be read back. What it
// offers is the way to change your mind, and nothing else: a kept note was
// never going to be done, so DONE here would answer the wrong question.
func TestTheShelfOffersOnlyTheWayBack(t *testing.T) {
	drew := drewFor(t, aShelf(), "kept")

	require.Contains(t, drew, `"open"`)
	require.NotContains(t, drew, `"done"`)
	require.NotContains(t, drew, `"drop"`)
	require.NotContains(t, drew, `"task"`)
	require.NotContains(t, drew, `"keep"`, "they are already kept")
}

func TestTheShelfNeverEmitsACount(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 12; i++ {
		items = append(items, note(i, "kept thing", squirrel.ItemKept))
	}
	drew := strings.ToLower(drewFor(t, &fakeStore{items: items}, "kept"))

	for _, total := range []string{"of 12", "(12)", "1 of ", "12 things"} {
		require.NotContains(t, drew, total)
	}
}

func TestTheEmptyShelfDoesNotInstruct(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=kept"))

	require.Contains(t, f.appended[1].Words, "Nothing on the shelf yet")
	for _, forbidden := range []string{"well done", "start by", "you should", "try to"} {
		require.NotContains(t, strings.ToLower(f.appended[1].Words), forbidden)
	}
}

// The shelf is reachable from the pile — and from every branch of it.
//
// It hung off the drawn card, so the moment there was nothing to decide about
// it was reachable from nowhere at all. That is the bug this fixes, and it is
// the one the comment beside the chip had been warning about since the deck
// came out.
func TestThePileOffersTheShelfEvenWithNothingToDecide(t *testing.T) {
	require.Contains(t, opened(t, aShelf(), "pile"), "the things you kept")

	empty := opened(t, &fakeStore{}, "pile")
	require.Contains(t, empty, "the things you kept", "an empty pile reached nowhere")
	require.Contains(t, empty, "what you set aside")
}

// And at the bottom, which you reach by skipping rather than deciding.
func TestTheBottomOfThePileAlsoReachesThem(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	routed(t, f).call(t, "POST", "/pile/later", strings.NewReader("after=1"))

	drew := string(f.appended[len(f.appended)-1].Shown)
	require.Contains(t, drew, "the things you kept")
	require.Contains(t, drew, "what you set aside")
}

// And the page itself is gone.
func TestThereIsNoPageForTheShelf(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.NotContains(t, m.routes, "GET /kept")
}
