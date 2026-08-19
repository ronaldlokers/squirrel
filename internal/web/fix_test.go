package web

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Only the words change. The arrival time, the state and the place in the pile
// are facts about the note; only the sentence was wrong.
func TestFixingANoteChangesOnlyTheWords(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boler makes a noise", squirrel.ItemOpen)}}
	was := f.items[0]

	w := post(t, mounted(t, f), "/pile/fix",
		url.Values{"id": {"1"}, "text": {"the boiler makes a noise on tuesdays"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "the boiler makes a noise on tuesdays", f.items[0].RawText)
	require.Equal(t, was.ID, f.items[0].ID)
	require.Equal(t, was.ReceivedAt, f.items[0].ReceivedAt, "it arrived when it arrived")
	require.Equal(t, was.State, f.items[0].State, "and it is where it was")
}

// A note cannot be emptied into nothing. That is what dropping is for, and
// dropping is reversible.
func TestANoteCannotBeFixedIntoNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}

	w := post(t, mounted(t, f), "/pile/fix", url.Values{"id": {"1"}, "text": {"   "}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "buy milk", f.items[0].RawText)
}

// A correction made three notes down comes back to the same place, like every
// other transition on this screen.
func TestFixingReturnsToWhereYouWere(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(1, "one", squirrel.ItemOpen),
		note(2, "two", squirrel.ItemOpen),
	}}

	w := post(t, mounted(t, f), "/pile/fix",
		url.Values{"id": {"2"}, "text": {"two, corrected"}, "after": {"1"}})

	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "1", loc.Query().Get("after"))
}

// The field carries the note's own words, so a correction starts from what is
// there rather than from an empty box.
func TestTheFieldStartsFromWhatTheNoteSays(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boler makes a noise", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `<textarea name="text" rows="3" aria-label="What the note should say">the boler makes a noise</textarea>`)
}
