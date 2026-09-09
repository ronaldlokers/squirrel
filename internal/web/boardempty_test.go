package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Three sentences and not one, because "nothing here" three times down a
// screen reads as a fault, and what is true of each is different: no daily
// chores is a fact about your life rather than a gap in the day.
func TestARackWithNothingInItSaysSoInItsOwnWords(t *testing.T) {
	m := mounted(t, &fakeStore{})
	body := m.call(t, "GET", "/", nil).Body.String()

	for rack, says := range map[string]string{
		"now":    "nothing comes back today",
		"daily":  "nothing every day",
		"weekly": "nothing this often",
		"seldom": "nothing that comes back slowly",
		"once":   "nothing to do just the one time",
	} {
		require.Contains(t, theRackIn(t, body, "bay="+rack), says,
			"an empty %s rack says nothing about being empty", rack)
	}

	behind := m.call(t, "GET", "/notes", nil).Body.String()
	require.Contains(t, behind, "nothing in the notes",
		"an empty notes says nothing about being empty")
}

func TestARackThatHoldsSomethingSaysNothingAboutBeingEmpty(t *testing.T) {
	m := mounted(t, aBoardStore())
	body := m.call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, theRackIn(t, body, "bay=weekly"), "nothing this often")
	require.NotContains(t, m.call(t, "GET", "/notes", nil).Body.String(),
		"nothing in the notes")
}

func TestARackThatCannotBeReadDoesNotAlsoCallItselfEmpty(t *testing.T) {
	f := &fakeStore{choresErr: errTest}
	rack := theRackIn(t, mounted(t, f).call(t, "GET", "/", nil).Body.String(), "bay=weekly")

	require.Contains(t, rack, "cannot reach the chores")
	require.NotContains(t, rack, "nothing this often",
		"a rack that could not be read reports a quiet morning as well")
}

func TestNothingMatchedIsSaidRatherThanDrawnAsAnEmptyChannel(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/?find=zeppelin", nil).Body.String()

	require.Contains(t, body, "nothing matched")
}

func TestAnEmptyWallSaysSo(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/notes", nil).Body.String()

	require.Contains(t, body, "nothing in the notes")
}

func TestAShelfIsNotSomewhereYouCanStillGo(t *testing.T) {
	for _, gone := range []string{"/?shelf=kept", "/?shelf=held", "/?bay=notes"} {
		res := mounted(t, &fakeStore{}).call(t, "GET", gone, nil)

		require.Equal(t, 301, res.Code, "%s still draws something", gone)
		require.Equal(t, "/notes", res.Header().Get("Location"), "%s lands in the wrong place", gone)
	}
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

	for _, name := range []string{"what Squirrel told you", "who you are, and what this can be told to do", "the notes"} {
		require.Contains(t, bar, `aria-label="`+name+`"`, "no chip is named %q", name)
	}
	require.Contains(t, bar, `aria-label="how you are`, "the mood chip is a picture with no name")
	require.Equal(t, 4, strings.Count(bar, `class="chip`),
		"the bar carries a different number of chips than it is named for")
}

func TestAnEmptyRackIsStillAPlaceYouCanPutSomething(t *testing.T) {
	m := mounted(t, &fakeStore{})

	rack := m.call(t, "GET", "/notes", nil).Body.String()
	require.Contains(t, rack, "nothing in the notes")
	require.Contains(t, rack, `placeholder="what is it"`)

	// And so is an empty board, which has one writer for everything.
	board := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, board, `href="#add"`, "an empty board has nowhere to put anything")
	require.Equal(t, 1, strings.Count(board, `class="addform`),
		"more than one place asks what it is")
}

func TestOnlyTheBlankAndTheNoticesAreDrawnDashed(t *testing.T) {
	css, err := staticFS.ReadFile("static/board.css")
	require.NoError(t, err)

	allowed := map[string]bool{
		".strip.blank":              true,
		".strip.blank:hover":        true,
		".camera":                   true,
		".camera:hover":             true,
		".blankstrip .inline":       true,
		".seam::after":              true,
		".blankstrip .under .count": true,
		".trouble":                  true,
		".baysign.shelf":            true,
		".strip.blank:focus-within": true,
		".blankstrip .strip.blank":  true,
		".dots i.nought":            true,
		".addpress":                 true,
		".moodkey b.nought":         true,
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

func TestTheRailIsNotDrawnWhereTheBoardIsNot(t *testing.T) {
	m := mounted(t, aBoardStore())

	for _, where := range []string{"/?find=kaas", "/?open=1"} {
		require.NotContains(t, m.call(t, "GET", where, nil).Body.String(), `class="dayrail"`,
			"%s draws the phone's rail behind what you came to read", where)
	}
	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), `class="dayrail"`)
}

// The bay drawings went with the bays, and nothing that was drawn is still
// shipped. The tab bar that replaced them went too, on 9 September 2026.
func TestNoBayPictureIsStillShipped(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "bay-", "something still asks for a bay's drawing")
	for _, gone := range []string{"notes", "chores", "tasks", "agenda"} {
		_, err := staticFS.ReadFile("static/bay-" + gone + ".png")
		require.Error(t, err, "the %s icon is still shipped and nothing asks for it", gone)
	}
}

func TestARackSignCountsWhatIsAskingAndNeverNought(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, theRackIn(t, body, "bay=once"), `<span class="n">1</span>`)
	require.NotContains(t, theRackIn(t, body, "bay=daily"), `<span class="n">`,
		"an empty rack wears a badge with nothing in it")

	empty := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, empty[strings.Index(empty, `<main class="racks">`):], `<span class="n">`,
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

func TestARackSignNamesItWithoutAnArticle(t *testing.T) {
	m := mounted(t, aBoardStore())
	body := m.call(t, "GET", "/", nil).Body.String()

	for _, rack := range []string{"now", "daily", "weekly", "seldom", "once"} {
		require.Contains(t, body, `<h2 class="baysign">`+rack,
			"the board does not name the %s rack", rack)
	}
	require.NotContains(t, body, `<h2 class="baysign">the `,
		"a rack sign still carries an article")
	require.Contains(t, m.call(t, "GET", "/notes", nil).Body.String(),
		`<h2 class="baysign">the notes`, "the wall's own sign lost its article")
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

func TestTheFaceOpensAPageOfItsOwn(t *testing.T) {
	m := mounted(t, aBoardStore())

	require.Contains(t, m.call(t, "GET", "/", nil).Body.String(), `class="chip face" href="/me"`,
		"the face opens somewhere other than who you are")

	page := m.call(t, "GET", "/me", nil).Body.String()
	require.Contains(t, page, "Who you are")
	require.Contains(t, page, "log out")
	require.Contains(t, page, "What Squirrel knows about you")
	require.Contains(t, page, "How you felt before")
	require.NotContains(t, page, `class="thread"`, "the settings are drawn inside a conversation")
}

func TestAStripCarriesWhatWasNoticedAboutIt(t *testing.T) {
	f := aBoardStore()
	f.noticed = []squirrel.Noticed{
		{ID: 9, Kind: "note", RefID: 1, Words: "The code you need for this is in the other note."},
	}
	rack := mounted(t, f).call(t, "GET", "/notes", nil).Body.String()

	require.Contains(t, rack, "The code you need for this is in the other note.")
	require.Contains(t, rack, `<input type="hidden" name="id" value="9">`,
		"the line cannot be refused")
	require.Contains(t, rack, "not useful")
}

func TestALineIsHungOnTheStripItNames(t *testing.T) {
	f := aBoardStore()
	f.noticed = []squirrel.Noticed{
		{ID: 9, Kind: "note", RefID: 2, Words: "about the second one"},
	}
	rack := mounted(t, f).call(t, "GET", "/notes", nil).Body.String()

	// By the words of the strips themselves rather than by their ids: an id
	// appears in half a dozen hidden fields per strip, which is what made the
	// first version of this pass for the wrong reason.
	first := strings.Index(rack, "boiler service code is 4471")
	second := strings.Index(rack, "kaas")
	line := strings.Index(rack, "about the second one")
	require.Positive(t, first)
	require.Greater(t, second, first, "this measured nothing: the strips are in the other order")
	require.Greater(t, line, second, "the line was drawn on a strip it was not about")
	require.Equal(t, -1, strings.Index(rack[:second], `class="seen"`),
		"a line was drawn on the strip before the one it names")
}

func TestAStripWithNothingNoticedCarriesNoLine(t *testing.T) {
	rack := mounted(t, aBoardStore()).call(t, "GET", "/notes", nil).Body.String()

	require.NotContains(t, rack, "not useful",
		"a strip nothing was noticed about still offers a way to refuse it")
	require.NotContains(t, rack, `class="seen"`)
}

func TestARefusalIsRecordedAgainstTheLine(t *testing.T) {
	f := aBoardStore()
	res := mounted(t, f).call(t, "POST", "/board/notuseful",
		strings.NewReader("id=9&bay=notes"))

	require.Equal(t, 303, res.Code)
	require.Equal(t, "/notes", res.Header().Get("Location"))
	require.Equal(t, []int64{9}, f.unuseful, "nothing was refused")
}

func TestARackThatCannotReadWhatWasNoticedStillDraws(t *testing.T) {
	f := aBoardStore()
	f.noticeErr = errTest
	body := mounted(t, f).call(t, "GET", "/notes", nil).Body.String()

	require.Contains(t, body, "boiler service code is 4471",
		"a read that failed took the rack with it")
	require.NotContains(t, body, `class="seen"`)
}
