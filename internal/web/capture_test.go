package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheSlotKeepsAThought(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)

	w := post(t, m, "/capture", url.Values{"text": {"ask the garage about the rattle"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?kept=1", w.Header().Get("Location"))
	require.Len(t, f.items, 1)
	require.Equal(t, "ask the garage about the rattle", f.items[0].RawText)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "a captured note is in the pile")
}

// The one word the slot says back, and it names no place you are behind.
func TestTheSlotSaysKeptAndNothingElse(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/?kept=1", nil).Body.String()

	require.Contains(t, body, "kept")
	for _, total := range []string{"1 note", "added", "in the pile now", "to review"} {
		require.NotContains(t, strings.ToLower(body), total)
	}
}

// There is no spool behind this write. The chat's 👀 means the words reached
// disk before anything else could go wrong; here there is no such stage, so an
// unreachable database is a note that was never taken — and the only honest
// answer is to say so and give the words back.
func TestAFailedCaptureKeepsTheWords(t *testing.T) {
	m := mounted(t, &fakeStore{err: errTest})

	w := post(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}})

	require.Equal(t, 303, w.Code)
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "1", loc.Query().Get("nokeep"))
	require.Equal(t, "the boiler makes a noise", loc.Query().Get("said"),
		"the words come back rather than disappearing")

	body := mounted(t, &fakeStore{}).call(t, "GET", "/?"+loc.RawQuery, nil).Body.String()
	require.Contains(t, body, "Not kept")
	require.Contains(t, body, "the boiler makes a noise", "still in the field")
}

// A thought that reads like a command is still a thought. In a chat room the
// only thing separating the two is the words; the slot has no commands to be
// confused with, so it never interprets. Without the payload marker these
// would be stored and then hidden by the pile's own definition of a note —
// a thought lost silently.
func TestTheSlotNeverReadsAThoughtAsACommand(t *testing.T) {
	for _, text := range []string{"done 2", "!notes", "every day vacuum", "?"} {
		f := &fakeStore{}
		w := post(t, mounted(t, f), "/capture", url.Values{"text": {text}})

		require.Equal(t, 303, w.Code, text)
		require.Len(t, f.items, 1, text)
		require.Equal(t, text, f.items[0].RawText, text)
	}
}

func TestAnEmptySlotDoesNothing(t *testing.T) {
	f := &fakeStore{}
	w := post(t, mounted(t, f), "/capture", url.Values{"text": {"   "}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Empty(t, f.items, "whitespace is not a thought")
}

// The words come back through the address bar, so they are escaped on the way
// out like any other text on this screen.
func TestTheSlotEscapesWhatItGivesBack(t *testing.T) {
	body := mounted(t, &fakeStore{}).
		call(t, "GET", "/?nokeep=1&said=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil).Body.String()

	require.NotContains(t, body, "<script>alert(1)</script>")
	require.Contains(t, body, "&lt;script&gt;")
}
