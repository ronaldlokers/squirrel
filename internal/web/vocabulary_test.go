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
func TestBothCapturePathsSayTheSameWords(t *testing.T) {
	home, err := templateFS.ReadFile("templates/home.html")
	require.NoError(t, err)
	js, err := staticFS.ReadFile("static/pile.js")
	require.NoError(t, err)

	// The server's four, as they are written in the template.
	said := regexp.MustCompile(`<p class="slotsaid[^"]*"[^>]*>([^<]+)</p>`)
	found := said.FindAllStringSubmatch(string(home), -1)
	require.Len(t, found, 4, "the slot no longer says four things")

	for _, m := range found {
		words := strings.TrimSpace(m[1])
		// Entities are how the template writes a dash; the script writes the
		// character. Compare what a person reads.
		words = strings.ReplaceAll(words, "&mdash;", "—")
		require.Contains(t, string(js), words,
			"the script does not say what the server says:\n  %q", words)
	}
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
	for _, name := range []string{"home.html", "coach.html"} {
		body, err := templateFS.ReadFile("templates/" + name)
		require.NoError(t, err)
		require.NotContains(t, string(body), ">Keep it<",
			"%s captures with a word that already means shelve", name)
	}
	split, err := templateFS.ReadFile("templates/split.html")
	require.NoError(t, err)
	require.NotContains(t, string(split), "KEEP THESE",
		"accepting a split uses the shelve verb")
}
