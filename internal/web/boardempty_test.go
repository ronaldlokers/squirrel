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

func TestTheBarLightsNoBayWhenYouAreNotInOne(t *testing.T) {
	m := mounted(t, aBoardStore())

	for _, where := range []string{"/?find=kaas", "/?shelf=kept", "/?open=1"} {
		require.NotContains(t, m.call(t, "GET", where, nil).Body.String(), `class="baytab in"`,
			"%s lights a bay you are not standing in", where)
	}
	require.Contains(t, m.call(t, "GET", "/?bay=tasks", nil).Body.String(), `class="baytab in"`)
}

func TestEveryBayWearsItsOwnIcon(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	for _, bay := range []string{"notes", "chores", "tasks", "agenda"} {
		require.Contains(t, body, `src="/static/bay-`+bay+`.png`,
			"the %s has no icon in the bar", bay)
		_, err := staticFS.ReadFile("static/bay-" + bay + ".png")
		require.NoError(t, err, "the %s asks for an icon that is not shipped", bay)
	}
}

func TestTheCountIsABadgeOnTheIconAndOnlyWhenThereIsOne(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()
	bar := body[strings.Index(body, `<nav class="baytabs">`):]

	require.Equal(t, 3, strings.Count(bar, `<span class="n">`),
		"the three bays that hold something do not each wear one badge")
	for _, tab := range strings.Split(bar, `<span class="pic">`)[1:] {
		badge := strings.Index(tab, `<span class="n">`)
		if badge < 0 {
			continue
		}
		require.Less(t, badge, strings.Index(tab, `<span class="says">`),
			"the count is beside the name rather than on the icon")
	}
	require.NotContains(t, bar, `<span class="n">0</span>`, "an empty bay wears a badge saying nought")

	empty := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, empty[strings.Index(empty, `<nav class="baytabs">`):], `<span class="n">`,
		"a board with nothing on it still wears badges")
}

func TestOnlyTheFootOfThePhoneClaimsTheSafeArea(t *testing.T) {
	css, err := staticFS.ReadFile("static/board.css")
	require.NoError(t, err)

	phone, depth, selector := false, 0, ""
	claimed := map[string]bool{}
	for _, line := range strings.Split(string(css), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@media") && strings.Contains(line, "max-width: 620px") {
			phone, depth = true, 0
		}
		if phone {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 && strings.Contains(line, "}") && selector == "" {
				phone = false
			}
		}
		if i := strings.Index(line, "{"); i > 0 {
			selector = strings.TrimSpace(line[:i])
		}
		if strings.Contains(line, "}") && !strings.Contains(line, "{") {
			selector = ""
		}
		if phone && strings.Contains(line, "safe-area-inset-bottom") && selector != "" {
			claimed[selector] = true
		}
	}

	require.Equal(t, map[string]bool{".baytabs": true}, claimed,
		"more than the bar at the foot pads for the home indicator, so the phone shows a band of nothing above it")
}
