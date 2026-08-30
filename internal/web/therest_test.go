package web

import (
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// "The rest" shows the rest.
//
// It was a link — `/?open=tasks` — and the thread has never read a query like
// that, so it reloaded the conversation and did nothing. Search's was worse:
// `/pile?q=…`, a route deleted with the deck, which answered 404. All three
// had been dead since the doors became messages; dropping a reply from twelve
// cards to five is what would have made somebody find out.

func nineTasks() *fakeStore {
	items := []squirrel.Item{}
	for i := int64(1); i <= 9; i++ {
		items = append(items, task(i, "decided thing "+string(rune('a'+i-1)), squirrel.ItemOpen))
	}
	return &fakeStore{items: items}
}

func TestTheRestOfTheTasksIsTheRestOfTheTasks(t *testing.T) {
	f := nineTasks()
	m := routed(t, f)

	f.appended = nil
	m.call(t, "GET", "/r/tasks", nil)
	first := string(f.appended[len(f.appended)-1].Shown)
	require.Contains(t, first, "decided thing a")
	require.Contains(t, first, "decided thing e")
	require.NotContains(t, first, "decided thing f")
	require.Contains(t, first, `"from":"5"`, "the chip does not say where to carry on from")

	f.appended = nil
	m.call(t, "POST", "/open", strings.NewReader("where=tasks&from=5"))
	second := string(f.appended[len(f.appended)-1].Shown)
	require.Contains(t, second, "decided thing f")
	require.Contains(t, second, "decided thing i")
	require.NotContains(t, second, "decided thing a", "it started again from the top")
	require.NotContains(t, second, `"label":"the rest"`, "it offered a page that is not there")
}

// And what you said was that you wanted the rest, not that you opened the
// tasks again — the record is a record of what happened.
func TestAskingForTheRestSaysSo(t *testing.T) {
	f := nineTasks()
	f.appended = nil
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks&from=5"))

	require.Equal(t, "the rest of the tasks", f.appended[0].Words)
}

// A new one is offered at the top of a list and not in the middle of one.
func TestTheRestDoesNotOfferANewOne(t *testing.T) {
	f := nineTasks()
	f.appended = nil
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks&from=5"))

	require.NotContains(t, string(f.appended[len(f.appended)-1].Shown), "a new task")
}

// Past the end is a sentence rather than an empty reply, which reads as a
// press that did not land.
func TestPastTheEndSaysSo(t *testing.T) {
	f := nineTasks()
	f.appended = nil
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks&from=99"))

	require.Contains(t, f.appended[len(f.appended)-1].Words, "That is all of them")
}

// The chores page the same way.
func TestTheRestOfTheChores(t *testing.T) {
	chores := []squirrel.Chore{}
	for i := int64(1); i <= 8; i++ {
		chores = append(chores, squirrel.Chore{
			ID: i, Name: "chore " + string(rune('a'+i-1)), Active: true,
			Every: 7 * 24 * time.Hour, EveryDays: 7,
		})
	}
	f := &fakeStore{chores: chores}
	m := routed(t, f)

	f.appended = nil
	m.call(t, "GET", "/r/chores", nil)
	require.Contains(t, string(f.appended[len(f.appended)-1].Shown), `"from":"5"`)

	f.appended = nil
	m.call(t, "POST", "/open", strings.NewReader("where=chores&from=5"))
	require.Contains(t, string(f.appended[len(f.appended)-1].Shown), "chore f")
	require.NotContains(t, string(f.appended[len(f.appended)-1].Shown), "chore a")
}

// And the agenda, which had no chip at all: it drew five and said nothing
// about the sixth.
func TestTheAgendaOffersTheRest(t *testing.T) {
	coming := []squirrel.Moment{}
	for i := int64(1); i <= 8; i++ {
		coming = append(coming, squirrel.Moment{
			ID: i, Label: "thing " + string(rune('a'+i-1)),
			Starts: now().Add(time.Duration(i) * 24 * time.Hour),
		})
	}
	f := &fakeStore{upcoming: coming}
	m := routed(t, f)

	f.appended = nil
	m.call(t, "GET", "/r/at", nil)
	require.Contains(t, string(f.appended[len(f.appended)-1].Shown), `"from":"5"`)

	f.appended = nil
	m.call(t, "POST", "/open", strings.NewReader("where=at&from=5"))
	require.Contains(t, string(f.appended[len(f.appended)-1].Shown), "thing f")
}

// Search's chip led to a 404. There is no second page of search, and
// inventing one to make a chip work would be building a feature to fix a
// link — so the offer is narrowed instead.
func TestSearchNoLongerPointsAtTheDeck(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 40; i++ {
		items = append(items, note(i, "a boiler thing", squirrel.ItemOpen))
	}
	f := &fakeStore{items: items}
	f.appended = nil
	routed(t, f).call(t, "POST", "/find", strings.NewReader("q=boiler"))

	drew := string(f.appended[len(f.appended)-1].Shown)
	require.NotContains(t, drew, "/pile", "it still points at the deck")
	require.Contains(t, drew, "/find/ask")
}

// Every chip that acts is a form. A link out of the conversation is a link to
// somewhere that is not the conversation, and there is nowhere else.
// Every chip that is a link points somewhere that exists.
//
// This was "no chip is a link" until 28 August, and the rule was always the
// narrower one underneath it: the three chips that broke it pointed at
// `/?open=chores`, `/?open=tasks` and `/pile?q=…`, none of which the thread
// has ever read, so pressing one reloaded the conversation and did nothing —
// which reads exactly like a press that did not land.
//
// Rooms are links, so the blanket ban had to go. What replaces it is the check
// the ban was standing in for, and it is stronger: resolve every href against
// the route table.
func TestEveryChipGoesSomewhereThatExists(t *testing.T) {
	f := nineTasks()
	f.checkin = fresh()
	f.aside = []squirrel.HeldItem{{ID: 5, Text: "the referral", State: squirrel.ItemWaiting}}
	body := opened(t, f, "tasks")

	m := mounted(t, &fakeStore{})
	hrefs := regexp.MustCompile(`<a class="chip" href="([^"]+)"`).FindAllStringSubmatch(body, -1)
	require.NotEmpty(t, hrefs, "no linked chips rendered, so this measured nothing")
	for _, h := range hrefs {
		path, query, _ := strings.Cut(h[1], "?")
		require.True(t, reachable(m, path), "a chip points at %q, which nothing serves", h[1])

		// And the query has to be one somebody reads. All three of the dead
		// chips resolved to a path that exists — `/?open=chores` is `/` — so
		// checking the path alone is the check that let them ship.
		q, err := url.ParseQuery(query)
		require.NoError(t, err)
		for key := range q {
			require.Contains(t, readsQuery, key,
				"a chip passes ?%s, which no handler reads — pressing it "+
					"reloads the page and does nothing", key)
		}
	}

	// And anything that is not a link is still a press, never a bare anchor
	// somewhere else in the turn.
	tags := regexp.MustCompile(`<([a-z]+) class="chip`).FindAllStringSubmatch(body, -1)
	for _, tag := range tags {
		// A label is the picker's own radio, which is a press too.
		require.Contains(t, []string{"a", "button", "label"}, tag[1],
			"a chip is a <%s>, which is neither a press nor a way somewhere", tag[1])
	}
}

// readsQuery is every query key a GET handler on this screen actually reads.
// Kept by hand, and short on purpose: a key nobody reads is a chip that looks
// like it did not land.
var readsQuery = []string{"before", "ask", "anyway", "held"}

// reachable says the route table answers a GET for this path.
func reachable(m *testMux, path string) bool {
	for pattern := range m.routes {
		method, p, _ := strings.Cut(pattern, " ")
		if method != "GET" {
			continue
		}
		if exact, ok := strings.CutSuffix(p, "{$}"); ok {
			if path == exact {
				return true
			}
			continue
		}
		if matchesPath(p, path) {
			return true
		}
	}
	return false
}

// The room for the dock is reserved on the column and not on `.thread`.
//
// `.thread` is not the last thing on the page — whatever sits below it is
// outside its padding, and on a phone that means behind the dock with no way
// to scroll it clear. Reported from a phone, and invisible on a laptop where
// the page is short enough that nothing needs scrolling at all.
//
// The measurement is TestBrowserTheEndOfThePageClearsTheDock. This is the
// narrower claim that browser cannot make: which element carries the reserve.
// The dock cannot cover the end of the conversation, and it is a fact about
// the layout rather than about a reserve.
//
// It used to be fixed to the viewport, so the column reserved its measured
// height as padding through a --dockspace custom property that thread.js kept
// up to date with a ResizeObserver. The dock is the last row of the thread's
// own grid now: a slot grown to four lines shortens the scroll region by
// exactly its own growth, which is what a layout does and the observer was
// standing in for.
//
// Asserted on the grid rather than on the absence of the old property, because
// "--dockspace is gone" passes for every reason including the one where the
// dock went back to being fixed and nothing reserves anything.
func TestTheDockIsARowAndNotAThingOnTop(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)
	sheet := string(css)

	require.Contains(t, ruleFor(t, sheet, ".dock"), "position: sticky",
		"the dock does not sit at the foot of the transcript")
	require.NotContains(t, ruleFor(t, sheet, ".dock"), "position: fixed",
		"the dock is fixed to the viewport again")
	require.NotContains(t, sheet, "--dockspace:",
		"something reserves room for the dock again instead of the dock reserving its own")
}

// ruleFor is one selector's declarations. The whole sheet is the wrong thing
// to search: a check that only asks whether some string is absent from 1600
// lines passes for every reason including the one where the rule moved.
func ruleFor(t *testing.T, css, selector string) string {
	t.Helper()
	at := strings.Index(css, "\n  "+selector+" {")
	require.Positive(t, at, "%s has no rule at all", selector)
	body := css[at+len(selector)+5:]
	return body[:strings.Index(body, "}")]
}

func TestAPillIsASingleLineLabelYouPress(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)
	sheet := string(css)

	for _, selector := range []string{
		".letmein",
		".roomsheet > summary",
		".post",
		".abtn",
		".chip",
		".enough .leavehere",
	} {
		require.Contains(t, ruleFor(t, sheet, selector), "border-radius: 999px",
			"%s is a single-line label you press and is not a pill", selector)
	}

	// The other half of the rule, and the half the last round got wrong: a
	// pressable cell inside a structure is not a single-line label, and a
	// pill on one costs a fifth of its own hit area to the corners.
	for _, selector := range []string{
		".calgrid label.day, .calgrid span.gone",
		".hit button",
	} {
		require.Contains(t, ruleFor(t, sheet, selector), "border-radius: var(--r)",
			"%s wraps or sits in a grid, so a pill clips what it can be pressed by", selector)
	}
}

func TestTheRadiusVocabularyIsClosed(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)
	sheet := string(css)

	design, err := os.ReadFile("../../DESIGN.md")
	require.NoError(t, err)

	named := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(string(design), "\n") {
		if strings.HasPrefix(line, "rounded:") {
			inBlock = true
			continue
		}
		if inBlock {
			if !strings.HasPrefix(line, "  ") {
				break
			}
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) == 2 {
				named[strings.Trim(strings.TrimSpace(parts[1]), `"`)] = parts[0]
			}
		}
	}
	require.NotEmpty(t, named, "DESIGN.md declares no rounded tokens")

	// The small marks: a documented category in ## Shapes whose values are
	// deliberately not tokens, and the same three the detector is configured
	// to pass over.
	for _, mark := range []string{"7px", "6px", "4px"} {
		named[mark] = "a small mark"
	}
	named["var(--r)"] = "card"

	for _, m := range regexp.MustCompile(`border-radius:\s*([^;]+);`).FindAllStringSubmatch(sheet, -1) {
		for _, corner := range strings.Fields(m[1]) {
			if corner == "0" {
				continue
			}
			require.Contains(t, named, corner,
				"border-radius %q is in the stylesheet and in no documented meaning", corner)
		}
	}
}
