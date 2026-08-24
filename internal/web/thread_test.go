package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func thread(t *testing.T, f *fakeStore) string {
	t.Helper()
	return routed(t, f).call(t, "GET", "/", nil).Body.String()
}

// The conversation reads top to bottom, newest at the end.
//
// The words are deliberately nothing the furniture also says: "the chores" is
// on a door and in the menu, so an index of it would find the rail and pass
// whichever way round the turns rendered.
func TestThreadRendersTurnsInOrder(t *testing.T) {
	body := thread(t, &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Morning."},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "milk please"},
	}})

	first := strings.Index(body, "Morning.")
	second := strings.Index(body, "milk please")
	require.NotEqual(t, -1, first)
	require.NotEqual(t, -1, second)
	require.Less(t, first, second, "the conversation reads top to bottom")
}

// Only the newest Buddy turn carries controls. A test asserting the newest turn
// HAS buttons would pass with the rule deleted, so this asserts the older one
// does NOT.
func TestOnlyTheNewestBuddyTurnHasControls(t *testing.T) {
	body := thread(t, &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Two are due.",
			Shown: []byte(`{"cards":[{"title":"water the plants","acts":[{"label":"DID IT","action":"/chores/act"}]}]}`)},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "never mind"},
		{ID: 3, Who: squirrel.SpeakerBuddy, Words: "Right.",
			Shown: []byte(`{"cards":[{"title":"descale the kettle","acts":[{"label":"DID IT","action":"/chores/act"}]}]}`)},
	}})

	require.Contains(t, body, "water the plants", "scrollback keeps its words")
	require.Contains(t, body, "descale the kettle")
	require.Equal(t, 1, strings.Count(body, "DID IT"),
		"a card in scrollback keeps its words and loses its buttons")
}

func TestRailShowsFourDoorsWithTheirNumbers(t *testing.T) {
	body := thread(t, &fakeStore{
		waiting: squirrel.Waiting{Pile: 3, Tasks: 1, Chores: 2, Agenda: 1},
	})

	for _, name := range []string{"the pile", "the tasks", "the chores", "the agenda"} {
		require.Contains(t, body, name)
	}
	require.Contains(t, body, `class="doorcount">3<`)
	require.Contains(t, body, `class="doorcount">2<`)
}

// Zero is no number, not a nought. A door reading "0" is the scoreboard the
// retired rule was actually protecting against.
func TestADoorWithNothingWaitingShowsNoNumber(t *testing.T) {
	body := thread(t, &fakeStore{waiting: squirrel.Waiting{}})

	require.Contains(t, body, "the pile")
	require.NotContains(t, body, "doorcount")
}

// A door that has something shows it while a door that has nothing stays bare,
// in one render — the mixed case is where an "if" is most easily written the
// wrong way round.
func TestOnlyTheDoorsWithSomethingWaitingCarryANumber(t *testing.T) {
	body := thread(t, &fakeStore{waiting: squirrel.Waiting{Chores: 2}})

	require.Equal(t, 1, strings.Count(body, "doorcount"))
	require.Contains(t, body, `class="doorcount">2<`)
}

// The thread has no <h1> — home's own exemption — but a turn that opens a place
// carries that place's name as an <h2>, so heading navigation still works.
func TestATurnThatOpensAPlaceCarriesAHeading(t *testing.T) {
	body := thread(t, &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Two are due.",
			Shown: []byte(`{"place":"the chores"}`)},
	}})

	require.NotContains(t, body, "<h1")
	require.Contains(t, body, `<h2 class="turnplace">the chores</h2>`)
}

func TestThreadOffersThePageAboveWhenThereIsOne(t *testing.T) {
	body := thread(t, &fakeStore{
		turns:     []squirrel.Turn{{ID: 7, Who: squirrel.SpeakerYou, Words: "hello"}},
		moreTurns: true,
	})
	require.Contains(t, body, "/?before=7")
}

func TestThreadDoesNotOfferAPageAboveWhenThereIsNone(t *testing.T) {
	body := thread(t, &fakeStore{
		turns: []squirrel.Turn{{ID: 7, Who: squirrel.SpeakerYou, Words: "hello"}},
	})
	require.NotContains(t, body, "before=")
}

// Walking up the conversation asks for the page above, and does not silently
// answer with the end of it.
func TestThreadWalksBackwardsWhenAsked(t *testing.T) {
	f := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "older"},
		{ID: 9, Who: squirrel.SpeakerYou, Words: "newer"},
	}}
	body := routed(t, f).call(t, "GET", "/?before=9", nil).Body.String()

	require.Equal(t, int64(9), f.pagedBefore)
	require.Contains(t, body, "older")
	require.NotContains(t, body, "newer")
}

// A database that cannot count is not a reason to take the navigation away.
//
// Counting the rail's own doors rather than looking for "the pile": that name
// is in the lid's menu too, so a rail rendered as nothing at all would still
// contain it and the test would pass over a missing rail.
func TestTheDoorsSurviveACountThatFails(t *testing.T) {
	body := thread(t, &fakeStore{waitingErr: errTest})

	require.Equal(t, 4, strings.Count(body, `class="rdoor`))
	require.NotContains(t, body, "doorcount")
}

// Two turns for one press: what you said, and what Buddy said back. A test that
// only checked the words were kept would pass with the conversation missing.
func TestSayingSomethingWritesBothTurns(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/say", strings.NewReader("words=milk"))

	require.Len(t, f.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, f.appended[0].Who)
	require.Equal(t, "milk", f.appended[0].Words)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[1].Who)
	require.NotEmpty(t, f.appended[1].Words)
}

// The words still reach the spool. The thread is a record of the conversation,
// not a replacement for capture — the spool is what survives the database being
// unreachable, and losing a thought is the one failure this product exists to
// prevent.
func TestSayingSomethingSpoolsTheWords(t *testing.T) {
	sp := &fakeSpool{}
	routedSpooling(t, &fakeStore{}, sp).call(t, "POST", "/say", strings.NewReader("words=milk"))

	require.Len(t, sp.written, 1)
	require.Equal(t, "milk", sp.written[0].Text)
}

// An empty slot is not a turn. Pressing the button by accident must not put a
// blank bubble into a record that is never rewritten.
func TestSayingNothingWritesNothing(t *testing.T) {
	f := &fakeStore{}
	sp := &fakeSpool{}
	routedSpooling(t, f, sp).call(t, "POST", "/say", strings.NewReader("words=%20%20"))

	require.Empty(t, f.appended)
	require.Empty(t, sp.written)
}

// Words that could not be kept must not be acknowledged as kept. Buddy saying
// "kept" over a failed write is the two views disagreeing about the pile.
func TestSayingSomethingThatCannotBeKeptSaysSo(t *testing.T) {
	f := &fakeStore{}
	sp := &fakeSpool{err: errTest}
	routedSpooling(t, f, sp).call(t, "POST", "/say", strings.NewReader("words=milk"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "cannot reach")
}

// And the words are still in the record, so a failed keep does not eat the
// thought — the same promise the slot has always made, kept by the turn rather
// than by putting the text back in the box.
func TestAFailedKeepStillLeavesTheWordsInTheThread(t *testing.T) {
	f := &fakeStore{}
	routedSpooling(t, f, &fakeSpool{err: errTest}).
		call(t, "POST", "/say", strings.NewReader("words=milk"))

	require.Equal(t, "milk", f.appended[0].Words)
}
