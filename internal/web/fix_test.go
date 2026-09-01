package web

import (
	"net/url"
	"strings"
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

func TestANoteCannotBeFixedIntoNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}

	w := post(t, mounted(t, f), "/pile/fix", url.Values{"id": {"1"}, "text": {"   "}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "buy milk", f.items[0].RawText)
}

// The field carries the note's own words, so a correction starts from what is
// there rather than from an empty box.
func TestTheFieldStartsFromWhatTheNoteSays(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boler makes a noise", squirrel.ItemOpen)}}
	// The box opens on what it says now, so rewording is a correction rather
	// than typing it again from nothing.
	// A fresh reading, so Buddy does not ask how you are and become the live
	// edge himself — which takes the box off the question.
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}
	m := routed(t, f)
	m.call(t, "POST", "/pile/reword", strings.NewReader("id=1"))
	f.turns, f.appended = append(f.turns, f.appended...), nil
	body := m.call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, `>the boler makes a noise</textarea>`)
}
