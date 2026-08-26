package web

import (
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
	m.call(t, "POST", "/open", strings.NewReader("where=tasks"))
	first := string(f.appended[1].Shown)
	require.Contains(t, first, "decided thing a")
	require.Contains(t, first, "decided thing e")
	require.NotContains(t, first, "decided thing f")
	require.Contains(t, first, `"from":"5"`, "the chip does not say where to carry on from")

	f.appended = nil
	m.call(t, "POST", "/open", strings.NewReader("where=tasks&from=5"))
	second := string(f.appended[1].Shown)
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

	require.NotContains(t, string(f.appended[1].Shown), "a new task")
}

// Past the end is a sentence rather than an empty reply, which reads as a
// press that did not land.
func TestPastTheEndSaysSo(t *testing.T) {
	f := nineTasks()
	f.appended = nil
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks&from=99"))

	require.Contains(t, f.appended[1].Words, "That is all of them")
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
	m.call(t, "POST", "/open", strings.NewReader("where=chores"))
	require.Contains(t, string(f.appended[1].Shown), `"from":"5"`)

	f.appended = nil
	m.call(t, "POST", "/open", strings.NewReader("where=chores&from=5"))
	require.Contains(t, string(f.appended[1].Shown), "chore f")
	require.NotContains(t, string(f.appended[1].Shown), "chore a")
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
	m.call(t, "POST", "/open", strings.NewReader("where=at"))
	require.Contains(t, string(f.appended[1].Shown), `"from":"5"`)

	f.appended = nil
	m.call(t, "POST", "/open", strings.NewReader("where=at&from=5"))
	require.Contains(t, string(f.appended[1].Shown), "thing f")
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

	drew := string(f.appended[1].Shown)
	require.NotContains(t, drew, "/pile", "it still points at the deck")
	require.Contains(t, drew, "/find/ask")
}

// Every chip that acts is a form. A link out of the conversation is a link to
// somewhere that is not the conversation, and there is nowhere else.
func TestNoChipInTheConversationIsALink(t *testing.T) {
	f := nineTasks()
	f.checkin = fresh()
	body := opened(t, f, "tasks")

	require.NotContains(t, body, `class="chip" href="/?open`)
	require.NotContains(t, body, `href="/pile`)
}

// The clearance was on `.thread`, and `.thread` is not the last thing on the
// page: the two chips and the way out sit below it, outside its padding. On a
// phone they were behind the dock with no way to scroll them clear — reported
// from a phone, and invisible on a laptop where the page is short enough that
// nothing needs scrolling at all.
func TestThePageLeavesRoomForTheDock(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)
	sheet := string(css)

	require.Contains(t, sheet, "padding-bottom: var(--dockspace",
		"the column reserves no room for the dock")
	require.NotContains(t, sheet, "padding: 22px 22px 132px",
		"the clearance is back on .thread, which is not the last thing on the page")
	require.NotContains(t, sheet, "padding: 18px 14px 128px")
}

// That the reserve follows the slot as it grows is measured in a browser —
// see TestBrowserTheReserveFollowsTheSlot. A source-text assertion passed with
// the observer disabled, because the word was still in the file.
