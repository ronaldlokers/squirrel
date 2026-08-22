package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// One heading, on every screen.
//
// There were five treatments — <h1 class="deckhead"> on two screens, a bare
// <h1> on the empty states, and three different <p> classes doing the same job
// in three sizes and two cases everywhere else — and ten of thirteen templates
// had no <h1> at all, so heading navigation did not work anywhere in a product
// built for someone who might well use it.
//
// These are the fence around that. A sixth treatment is one careless template
// away, and the whole reason there were five is that nothing ever said so.

// notATitle are the templates that legitimately have no title of their own:
// the layout every page is poured into, and the fragments poured in with it.
var notATitle = map[string]string{
	"layout.html":  "the frame, not a screen",
	"card.html":    "a note in the deck, titled by the deck",
	"every.html":   "the interval picker, drawn inside a card",
	"split.html":   "a proposal, drawn inside a card",
	"step.html":    "one step, drawn inside whatever raised it",
	"results.html": "has one, and two of them, one per branch",
	// The front door, and the one screen without a title. Every other screen
	// is a place you navigated to, and its title answers "where am I"; home is
	// where you start, and nobody arrives there wondering. It carried "right
	// now" for an afternoon and read as a label for a room you were already
	// standing in.
	"home.html": "the front door — you do not arrive there wondering",
}

func templates(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	require.NoError(t, fs.WalkDir(templateFS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := templateFS.ReadFile(p)
		if err != nil {
			return err
		}
		out[strings.TrimPrefix(p, "templates/")] = string(b)
		return nil
	}))
	return out
}

func TestEveryScreenHasExactlyOneHeadingTreatment(t *testing.T) {
	for name, body := range templates(t) {
		if _, skip := notATitle[name]; skip {
			continue
		}
		require.Contains(t, body, "<h1", "%s has no heading at all", name)
	}
}

// The three paragraph classes are gone, and stay gone.
func TestNoScreenTitlesItselfWithAParagraph(t *testing.T) {
	for name, body := range templates(t) {
		for _, dead := range []string{`class="head"`, `class="chorehead"`, `class="resultsHead"`} {
			require.NotContains(t, body, dead,
				"%s titles itself with %s rather than a heading", name, dead)
		}
	}
}

// And the stylesheet does not still carry them, which is how a class comes
// back: the rule outlives the markup and the next screen finds it.
func TestTheStylesheetHasNoRuleForTheOldHeadings(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)
	for _, dead := range []string{".chorehead", ".resultsHead"} {
		require.NotContains(t, string(css), dead,
			"%s is still styled, so it is still available to be used", dead)
	}
}

// A title says what the screen is. It is not a count, and this is the one
// place a count would look most reasonable — "3 things you kept".
var digits = regexp.MustCompile(`\d`)

func TestNoTitleCountsAnything(t *testing.T) {
	head := regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)
	for name, body := range templates(t) {
		for _, m := range head.FindAllStringSubmatch(body, -1) {
			require.NotRegexp(t, digits, m[1], "%s puts a number in its title", name)
		}
	}
}
