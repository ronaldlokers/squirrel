package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestARackWithNothingInItSaysSoInItsOwnWords(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	for bay, says := range map[string]string{
		"notes":  "nothing in the notes",
		"chores": "nothing comes back today",
		"tasks":  "nothing in the tasks",
		"agenda": "nothing left today",
	} {
		require.Contains(t, theRackIn(t, body, "bay="+bay), says,
			"an empty %s rack says nothing about being empty", bay)
	}
}

func TestARackThatHoldsSomethingSaysNothingAboutBeingEmpty(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, theRackIn(t, body, "bay=notes"), "nothing in the notes")
	require.NotContains(t, theRackIn(t, body, "bay=chores"), "nothing comes back today")
}

func TestARackThatCannotBeReadDoesNotAlsoCallItselfEmpty(t *testing.T) {
	f := &fakeStore{choresErr: errTest}
	rack := theRackIn(t, mounted(t, f).call(t, "GET", "/", nil).Body.String(), "bay=chores")

	require.Contains(t, rack, "cannot reach the chores")
	require.NotContains(t, rack, "nothing comes back today",
		"a rack that could not be read reports a quiet morning as well")
}

func TestNothingMatchedIsSaidRatherThanDrawnAsAnEmptyChannel(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/?find=zeppelin", nil).Body.String()

	require.Contains(t, body, "nothing matched")
}

func TestAnEmptyShelfSaysSo(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/?shelf=kept", nil).Body.String()

	require.Contains(t, body, "nothing on this shelf")
}

func TestTheFindFieldStaysOpenWhenItCarriesAQuery(t *testing.T) {
	m := mounted(t, aBoardStore())

	require.Contains(t, m.call(t, "GET", "/?find=kaas", nil).Body.String(), `class="find open"`,
		"a phone draws the field shut over the words you searched for")
	require.NotContains(t, m.call(t, "GET", "/", nil).Body.String(), `class="find open"`)
}

func TestBuddysLinkKeepsItsWordsBesideTheAcorn(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `class="acorn"`, "the link has no mark to press on a phone")
	require.Contains(t, body, "talk to Buddy",
		"the link is a glyph with no name, which is a link nobody can follow")
}

func TestAnEmptyRackIsStillAPlaceYouCanPutSomething(t *testing.T) {
	rack := theRackIn(t, mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String(), "bay=notes")

	require.Contains(t, rack, "nothing in the notes")
	require.Contains(t, rack, `placeholder="what is it"`)
}

func TestOnlyTheBlankAndTheNoticesAreDrawnDashed(t *testing.T) {
	css, err := staticFS.ReadFile("static/board.css")
	require.NoError(t, err)

	allowed := map[string]bool{
		".strip.blank":              true,
		".strip.blank:hover":        true,
		".camera":                   true,
		".camera:hover":             true,
		".rhythms .stamp":           true,
		".rhythms .stamp:hover":     true,
		".trouble":                  true,
		".baysign.shelf":            true,
		".strip.blank:focus-within": true,
		".blankstrip .strip.blank":  true,
	}

	selector := ""
	for _, line := range strings.Split(string(css), "\n") {
		line = strings.TrimSpace(line)
		if i := strings.Index(line, "{"); i >= 0 {
			selector = strings.TrimSpace(line[:i])
		}
		if !strings.Contains(line, "dashed") {
			continue
		}
		for _, one := range strings.Split(selector, ",") {
			one = strings.TrimSpace(one)
			require.True(t, allowed[one],
				"%s is drawn dashed, and dashed says a thing is not filled in yet", one)
		}
	}
}
