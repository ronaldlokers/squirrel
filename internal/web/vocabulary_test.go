package web

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The two capture paths say the same words.
//
// pile.js carried a comment asserting exactly this — "the outcome is read out
// of the URL it redirects to, in exactly the vocabulary the scriptless path
// uses, so there is one set of answers rather than two that can disagree" —
// and they disagreed. The script lowercased all three and dropped the
// recovery clause from both failures: no "try again in a moment", no "keep
// them without it, or try another picture".
//
// The path with the script is the one nearly every session takes, so the
// enhanced path gave less help than the floor. A comment cannot hold two
// files together; this can.
// The thread's own path cannot drift, because the script renders the server's
// HTML rather than words of its own — see static/thread.js. What still has two
// copies is pile.js, which enhances the slots on the pages that are still
// pages, and this pins those to the sentences the Go code says.
func TestBothCapturePathsSayTheSameWords(t *testing.T) {
	js, err := staticFS.ReadFile("static/pile.js")
	require.NoError(t, err)

	for _, words := range []string{
		refusalOf(errNotAPhotograph),
		refusalOf(errTest),
	} {
		require.Contains(t, string(js), words,
			"the script does not say what the server says:\n  %q", words)
	}

	// And the one answer that is not a turn, because there is no database to
	// put a turn in when the worker takes it.
	thread, err := templateFS.ReadFile("templates/thread.html")
	require.NoError(t, err)
	held := regexp.MustCompile(`<p class="slotsaid held"[^>]*>([^<]+)</p>`).
		FindStringSubmatch(string(thread))
	require.Len(t, held, 2, "the dock no longer says what happens with no network")
	require.Contains(t, string(js), strings.ReplaceAll(strings.TrimSpace(held[1]), "&mdash;", "—"))
}

// And the recovery clause specifically, because that is what went missing and
// it is the half that tells you what to do next.
func TestTheFailureMessagesSayWhatToDoNext(t *testing.T) {
	js, err := staticFS.ReadFile("static/pile.js")
	require.NoError(t, err)

	for _, clause := range []string{
		"try again in a moment",
		"keep them without it, or try another picture",
	} {
		require.Contains(t, string(js), clause,
			"a failure message lost the sentence that says what to do")
	}
}

// One word, one meaning. `keep` was the capture button, the shelve verb, and
// the accept-a-split verb — three controls that could sit on adjacent screens
// meaning three different things, in a product whose argument is that
// deciding is the scarce resource.
func TestKeepMeansOneThing(t *testing.T) {
	// coach.html went on 25 August 2026 — Buddy is turns, and what he draws is
	// built in Go. turn.html is where a card's buttons are rendered now.
	for _, name := range []string{"thread.html", "turn.html"} {
		body, err := templateFS.ReadFile("templates/" + name)
		require.NoError(t, err)
		require.NotContains(t, string(body), ">Keep it<",
			"%s captures with a word that already means shelve", name)
	}
	// The split's own words live in Go now rather than in a template of its
	// own — the proposal is a turn. `keep` is the shelve verb and the button
	// that accepts a split must not borrow it.
	require.NotContains(t, proposeInThread(1, []string{"a", "b"}).Words, "Keep these")
}
