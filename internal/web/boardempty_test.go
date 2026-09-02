package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
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

// Buddy's acorn became a chat chip on 2 September 2026, beside the bell and in
// front of it. What this has always held is the half that matters: a control
// drawn as a glyph must still carry a name, or it is a link nobody can follow.
func TestEveryChipInTheBarCarriesAName(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()
	bar := body[strings.Index(body, `<header class="ops">`):strings.Index(body, "</header>")]

	for _, name := range []string{"talk to Buddy", "what Squirrel told you", "who you are, and what this can be told to do"} {
		require.Contains(t, bar, `aria-label="`+name+`"`, "no chip is named %q", name)
	}
	require.Equal(t, 3, strings.Count(bar, `class="chip`),
		"the bar carries a different number of chips than it is named for")
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
		".rhythms .count":           true,
		".rhythms select":           true,
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

	require.Equal(t, map[string]bool{":root": true}, claimed,
		"more than the bar at the foot pads for the home indicator, so the phone shows a band of nothing above it")
}

func TestTheBarNamesABayWithoutItsArticle(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()
	bar := body[strings.Index(body, `<nav class="baytabs">`):]

	for _, bay := range []string{"notes", "chores", "tasks", "agenda"} {
		require.Contains(t, bar, `<span class="says">`+bay+`</span>`,
			"the bar does not name the %s", bay)
	}
	require.NotContains(t, bar, `<span class="says">the `,
		"a cell in the bar still carries the article")
	require.Contains(t, body, `<h2 class="baysign">the notes`,
		"the rack's own sign lost the article with it")
}

func TestTheBellShowsWhatWasSaidAndSaysSoWhenNothingWas(t *testing.T) {
	f := aBoardStore()
	f.said = []squirrel.Said{
		{ID: 2, Title: "time to leave", Body: "the dentist is at 14:30", At: time.Now()},
		{ID: 1, Title: "the bins", Body: "they go out today", At: time.Now().Add(-4 * time.Hour)},
	}
	body := mounted(t, f).call(t, "GET", "/?told=1", nil).Body.String()

	require.Contains(t, body, "what Squirrel told you")
	require.Contains(t, body, "time to leave")
	require.Contains(t, body, "the dentist is at 14:30")
	require.Contains(t, body, "the bins")

	quiet := mounted(t, aBoardStore()).call(t, "GET", "/?told=1", nil).Body.String()
	require.Contains(t, quiet, "nothing has been sent to you")
}

func TestTheBellIsMarkedOnlyWhenSomethingWasSaid(t *testing.T) {
	f := aBoardStore()
	f.said = []squirrel.Said{{ID: 1, Title: "the bins", At: time.Now()}}

	require.Contains(t, mounted(t, f).call(t, "GET", "/", nil).Body.String(), `class="chip bell full"`)
	require.NotContains(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String(),
		`class="chip bell full"`, "the bell is marked when nothing has been sent")
}

func TestARecordThatCannotBeReadDrawsNoList(t *testing.T) {
	f := aBoardStore()
	f.saidErr = errTest
	body := mounted(t, f).call(t, "GET", "/?told=1", nil).Body.String()

	require.Contains(t, body, "nothing has been sent to you")
	require.NotContains(t, body, `class="chip bell full"`,
		"the bell is marked from a read that failed")
}

func TestTheFaceOpensWhatCanBeChangedRatherThanBuddy(t *testing.T) {
	m := mounted(t, aBoardStore())

	board := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, board, `class="chip face" href="/r/everything?you=1"`,
		"the face opens the room rather than the settings")

	shut := m.call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, shut, `<details class="youare">`, "the panel is open before it was asked for")

	open := m.call(t, "GET", "/r/everything?you=1", nil).Body.String()
	require.Contains(t, open, `<details class="youare" open>`, "the face opened nothing")
	require.Contains(t, open, "log out")
	require.Contains(t, open, "What Squirrel knows about you")
}
