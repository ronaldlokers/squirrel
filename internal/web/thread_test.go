package web

import (
	"strings"
	"testing"
	"time"

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
	// A fresh reading, so Buddy does not ask how you are and become the live
	// edge itself. The interaction is real and worth naming: anything Buddy
	// adds on arrival takes the controls off whatever was there before.
	body := thread(t, &fakeStore{checkin: fresh(), turns: []squirrel.Turn{
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

// Two turns for one press: what you said, and what Buddy said back. A test
// that only checked the words were kept would pass with the conversation
// missing.
func TestSayingSomethingWritesBothTurns(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/capture", strings.NewReader("text=milk"))

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
	routedSpooling(t, &fakeStore{}, sp).call(t, "POST", "/capture", strings.NewReader("text=milk"))

	require.Len(t, sp.written, 1)
	require.Equal(t, "milk", sp.written[0].Text)
}

// An empty slot is not a turn. Pressing the button by accident must not put a
// blank bubble into a record that is never rewritten.
func TestSayingNothingWritesNothing(t *testing.T) {
	f := &fakeStore{}
	sp := &fakeSpool{}
	routedSpooling(t, f, sp).call(t, "POST", "/capture", strings.NewReader("text=%20%20"))

	require.Empty(t, f.appended)
	require.Empty(t, sp.written)
}

// Words that could not be kept must not be acknowledged as kept. Buddy saying
// "kept" over a failed write is the two views disagreeing about the pile.
func TestSayingSomethingThatCannotBeKeptSaysSo(t *testing.T) {
	f := &fakeStore{}
	sp := &fakeSpool{err: errTest}
	routedSpooling(t, f, sp).call(t, "POST", "/capture", strings.NewReader("text=milk"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "cannot reach")
}

// And the words are still in the record, so a failed keep does not eat the
// thought — the same promise the slot has always made, kept by the turn rather
// than by putting the text back in the box.
func TestAFailedKeepStillLeavesTheWordsInTheThread(t *testing.T) {
	f := &fakeStore{}
	routedSpooling(t, f, &fakeSpool{err: errTest}).
		call(t, "POST", "/capture", strings.NewReader("text=milk"))

	require.Equal(t, "milk", f.appended[0].Words)
}

// Buddy asks while the reading is stale, and the question is a turn — so the
// morning is still in the record this evening. Home used to let the answer
// replace the question, and the morning was gone.
func TestBuddyAsksHowYouAreWhenTheReadingIsStale(t *testing.T) {
	f := &fakeStore{}
	thread(t, f)

	require.Len(t, f.appended, 1)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[0].Who)
	require.Contains(t, string(f.appended[0].Shown), "faces")
}

// And does not ask again while the answer still describes now. A question
// re-asked on every render would fill the record with the same turn.
func TestBuddyDoesNotAskWhileTheReadingIsFresh(t *testing.T) {
	f := &fakeStore{checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}}
	thread(t, f)

	require.Empty(t, f.appended)
}

// The five drawings, not five words. They are the control the capacity gate
// depends on and they are the product's own faces.
func TestTheQuestionCarriesTheFiveFaces(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Equal(t, 5, strings.Count(body, `class="face"`))
	require.Contains(t, body, "mood-frazzled.png")
}

// Walking up the conversation does not ask anything. A page of the past is
// being read, and reading it must not add to it.
func TestWalkingBackDoesNotAsk(t *testing.T) {
	f := &fakeStore{turns: []squirrel.Turn{{ID: 9, Who: squirrel.SpeakerYou, Words: "hello"}}}
	routed(t, f).call(t, "GET", "/?before=9", nil)

	require.Empty(t, f.appended)
}

// Answering records the reading and writes both turns.
func TestAnsweringTheCheckinWritesTurnsAndRecords(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/mood", strings.NewReader("mood=good"))

	require.Equal(t, squirrel.MoodGood, f.recorded)
	require.Len(t, f.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, f.appended[0].Who)
	require.Equal(t, "good", f.appended[0].Words)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[1].Who)
}

// Not one of the five is no answer rather than a wrong one, and no turn at all.
func TestAnAnswerThatIsNotOneOfTheFiveWritesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/mood", strings.NewReader("mood=splendid"))

	require.Empty(t, f.appended)
}

func fresh() *squirrel.Checkin {
	return &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}
}

// The one thing Squirrel picked, offered as a turn with something to press.
func TestTheOfferArrivesAsATurn(t *testing.T) {
	body := thread(t, &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"},
	})

	require.Contains(t, body, "water the plants")
	require.Contains(t, body, `action="/now/act"`)
	require.Contains(t, body, "DID IT")
	require.Contains(t, body, "not now")
}

// Nothing to hand you is a normal state and renders nothing at all — not an
// empty region, and not a reassuring sentence in its place.
func TestNoOfferIsNoTurn(t *testing.T) {
	f := &fakeStore{checkin: fresh()}
	thread(t, f)

	require.Empty(t, f.appended)
}

// Buddy does not hand you a job in the same breath as asking how you are. That
// was home's rule and it survives the move.
func TestNoOfferUntilTheQuestionIsAnswered(t *testing.T) {
	body := thread(t, &fakeStore{
		offer: &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"},
	})

	require.NotContains(t, body, "water the plants")
}

// The card carries every field the press needs. One hidden input is not
// enough — /now/act wants the kind and the row as well as the act.
func TestTheOfferCarriesTheKindAndTheRow(t *testing.T) {
	body := thread(t, &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"},
	})

	require.Contains(t, body, `name="kind" value="chore"`)
	require.Contains(t, body, `name="id" value="4"`)
}

// A running timer is a thing you are doing rather than a row that was picked,
// so it carries no buttons: the lid already has the one control it needs.
func TestARunningTimerOffersNothingToPress(t *testing.T) {
	body := thread(t, &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferTimer, Text: "water the plants"},
	})

	require.Contains(t, body, "water the plants")
	require.NotContains(t, body, "DID IT")
}

// Turning it down stays in the record. Stopping partway is a normal ending,
// and a record that keeps it is what that looks like when it is structural
// rather than a reassuring sentence.
func TestTurningTheOfferDownStaysInTheThread(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/now/act",
		strings.NewReader("kind=chore&id=4&act=later&label=water+the+plants"))

	require.Len(t, f.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, f.appended[0].Who)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[1].Who)
}

// And doing it says so, in different words: the two answers must not read the
// same, because which one you gave is the whole of what happened.
func TestDoingItAndTurningItDownSayDifferentThings(t *testing.T) {
	did := &fakeStore{}
	routed(t, did).call(t, "POST", "/now/act",
		strings.NewReader("kind=chore&id=4&act=did&label=water+the+plants"))

	later := &fakeStore{}
	routed(t, later).call(t, "POST", "/now/act",
		strings.NewReader("kind=chore&id=4&act=later&label=water+the+plants"))

	require.NotEqual(t, did.appended[0].Words, later.appended[0].Words)
}

// The offer is not put on the table twice. Without this a reload appended a
// second copy to a record that is never rewritten, and — worse — it took the
// live edge off whatever Buddy had just said.
func TestTheOfferIsNotOfferedAgainOverItself(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"},
	}
	m := routed(t, f)

	m.call(t, "GET", "/", nil)
	require.Len(t, f.appended, 1)

	f.turns, f.appended = f.appended, nil
	m.call(t, "GET", "/", nil)
	require.Empty(t, f.appended, "a reload must not say it again")
}

// And nothing is offered over an answer Buddy has just given, whatever kind of
// thing it put on the table.
func TestNothingIsOfferedOverSomethingAlreadyOnTheTable(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"},
		turns: []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "the way through",
			Shown: []byte(`{"cards":[{"title":"the smallest piece","acts":[{"label":"5 MIN","action":"/timer"}]}]}`)}},
	}
	body := thread(t, f)

	require.Empty(t, f.appended)
	require.Contains(t, body, "5 MIN", "the answer keeps its button")
}

// The fragment and the page render the same card. This is the only thing
// holding the single rendering path in place, and it fails the moment somebody
// adds a client-side template.
func TestAFragmentAndAPageRenderTheSameTurn(t *testing.T) {
	page := thread(t, &fakeStore{turns: []squirrel.Turn{
		{ID: 9, Who: squirrel.SpeakerBuddy, Words: "Kept."}, {ID: 10, Who: squirrel.SpeakerYou, Words: "milk"},
	}})

	f := &fakeStore{}
	fragment := routed(t, f).callFragment(t, "/capture", "text=milk").Body.String()

	require.Contains(t, page, `<p class="bub">Kept.</p>`)
	require.Contains(t, fragment, `<p class="bub">Kept.</p>`)
	require.NotContains(t, fragment, "<html", "a fragment is turns and nothing else")
	require.NotContains(t, fragment, "railwrap")
}

// A fragment press answers with the new turns rather than a redirect.
func TestAFragmentPressAnswersWithTheNewTurns(t *testing.T) {
	w := routed(t, &fakeStore{}).callFragment(t, "/capture", "text=milk")

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "milk")
	require.Contains(t, w.Body.String(), "Kept.")
}

// Without the header it still redirects, because that is what a form does and
// the handler must not have two behaviours by accident.
func TestAnOrdinaryPressStillRedirects(t *testing.T) {
	w := routed(t, &fakeStore{}).call(t, "POST", "/capture", strings.NewReader("text=milk"))

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
}

// The turns a press produces arrive with the controls on the last of them, so
// the live edge moves with the conversation rather than staying where the page
// was painted.
func TestTheFragmentCarriesTheLiveEdge(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"}}
	body := routed(t, f).callFragment(t, "/mood", "mood=good").Body.String()

	require.Contains(t, body, `href="/moods"`, "the newest turn keeps its chips")
}

// Pressing a door says its name, and Buddy answers with what is behind it.
func TestOpeningADoorSaysItsName(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "water the plants", Every: 7 * 24 * time.Hour,
			EveryDays: 7, SinceDays: 8, Active: true, EverDone: true},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=chores"))

	require.Len(t, f.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, f.appended[0].Who)
	require.Equal(t, "the chores", f.appended[0].Words)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[1].Who)
	require.Contains(t, string(f.appended[1].Shown), "water the plants")
}

// And the turn carries the place's name as a heading, so heading navigation
// still walks the app.
func TestTheReplyToADoorCarriesItsHeading(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=chores"))

	require.Len(t, f.appended, 2)
	require.Contains(t, string(f.appended[1].Shown), `"place":"the chores"`)
}

// A door nobody offered does nothing and says nothing. It arrives from a form,
// so it is read the way a stranger's typing is read.
func TestADoorThatDoesNotExistDoesNothing(t *testing.T) {
	f := &fakeStore{}
	w := routed(t, f).call(t, "POST", "/open", strings.NewReader("where=cellar"))

	require.Equal(t, 303, w.Code)
	require.Empty(t, f.appended)
}

// Nothing behind it says where chores come from, rather than counting to zero
// or saying nothing at all.
//
// Asserting on the words rather than on the absence of cards: `omitempty` makes
// an empty card list structurally impossible, so a test about that would pass
// with the empty branch deleted.
func TestAnEmptyPlaceSaysWhereChoresComeFrom(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=chores"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "Nothing comes back on its own")
	require.NotContains(t, f.appended[1].Words, "0")
}

// The rail posts rather than links, because opening a place is something you
// said and a GET that writes would write again on every reload.
func TestTheRailPostsRatherThanLinks(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Contains(t, body, `action="/open"`)
	require.NotContains(t, body, `<a class="rdoor`)
}

// A question is something on the table too.
//
// Without this the offer was appended over an unanswered picker and took the
// live edge from it, so "how often" arrived with no way to answer it. Found in
// a screenshot, which is the only place it was visible.
func TestNothingIsOfferedOverAnUnansweredQuestion(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		offer:   &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 4, Text: "water the plants"},
		turns: []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "How often should it come back?",
			Shown: []byte(`{"pick":{"action":"/chores/act","do":"that's it","rows":[{"lead":"every","name":"count","options":["1","2"]}]}}`)}},
	}
	body := thread(t, f)

	require.Empty(t, f.appended)
	require.Contains(t, body, `class="pick"`, "the question keeps its answers")
}

// The pile hands you one note, with the four ways out of it.
func TestOpeningThePileHandsYouOneNote(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(9, "the boiler makes a noise", squirrel.ItemOpen),
		note(8, "meter reading 48213", squirrel.ItemOpen),
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=pile"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, "the boiler makes a noise")
	require.NotContains(t, shown, "meter reading", "one at a time, like the deck")
	for _, act := range []string{"done", "keep", "drop", "task"} {
		require.Contains(t, shown, `"act":"`+act+`"`)
	}
}

// Acting on it says what happened and hands you the next one, so triage is a
// loop rather than a place you have to go back to.
func TestActingOnANoteHandsYouTheNext(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(9, "the boiler makes a noise", squirrel.ItemOpen),
		note(8, "meter reading 48213", squirrel.ItemOpen),
	}}
	routed(t, f).call(t, "POST", "/pile/act",
		strings.NewReader("id=9&act=keep&was=open&from=thread"))

	require.Equal(t, squirrel.ItemKept, f.states[9])
	// Three turns: what you did, what Buddy said, and the next one. The next
	// one is its own turn rather than part of the answer, so pressing DONE on
	// it does not read as pressing DONE on what you just decided.
	require.Len(t, f.appended, 3)
	require.Contains(t, f.appended[0].Words, "the boiler")
	require.Contains(t, string(f.appended[2].Shown), "meter reading 48213")
}

// The way to change your mind travels with what Buddy said, because the card
// is about to be scrollback and scrollback carries no controls.
func TestTheAnswerCarriesTheWayBack(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)}}
	routed(t, f).call(t, "POST", "/pile/act",
		strings.NewReader("id=9&act=keep&was=open&from=thread"))

	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, `"act":"open"`)
	require.Contains(t, shown, "put it back")
}

// Later is not a decision: it hands you the next one and leaves this where it
// was, which is the deck's own LATER.
func TestLaterHandsYouTheNextAndDecidesNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(9, "the boiler", squirrel.ItemOpen),
		note(8, "meter reading 48213", squirrel.ItemOpen),
	}}
	routed(t, f).call(t, "POST", "/pile/later", strings.NewReader("after=9"))

	require.Empty(t, f.states, "later decided something")
	require.Contains(t, string(f.appended[1].Shown), "meter reading 48213")
}

// Nothing left says so rather than handing you an empty card.
func TestAnEmptyPileSaysSo(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=pile"))

	require.Len(t, f.appended, 2)
	require.NotContains(t, string(f.appended[1].Shown), `"cards"`)
	require.NotEmpty(t, f.appended[1].Words)
}

// The deck still answers. It is deleted in its own piece of work, not in this
// one — a triage loop nobody has used yet is not a reason to take away the one
// that works.
func TestTheDeckStillStands(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "the boiler")
	require.Contains(t, body, `value="done"`)
}

// The three questions a note can be asked, rather than the three verbs that end
// it. Without them the deck can never be deleted, because they would be
// reachable nowhere else.
func TestANoteCanBeAskedTheThreeQuestions(t *testing.T) {
	for _, c := range []struct{ route, wants string }{
		{"/pile/often", `"pick"`},
		{"/pile/reword", `"say"`},
		{"/pile/why", `"chips"`},
	} {
		f := &fakeStore{items: []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)}}
		routed(t, f).call(t, "POST", c.route, strings.NewReader("id=9"))

		require.Len(t, f.appended, 2, c.route)
		require.Contains(t, string(f.appended[1].Shown), c.wants, c.route)
	}
}

// A note that is not yours cannot be asked about either.
func TestANoteThatIsNotYoursIsNotAskedAbout(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/pile/reword", strings.NewReader("id=99"))

	require.Empty(t, f.appended)
}

// Rewording says what it says now, and hands you the note again.
func TestRewordingSaysItAndCarriesOn(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)}}
	routed(t, f).call(t, "POST", "/pile/fix",
		strings.NewReader("id=9&from=thread&text=the+boiler+is+loud"))

	require.Len(t, f.appended, 3)
	require.Contains(t, f.appended[0].Words, "the boiler is loud")
	require.Contains(t, f.appended[1].Words, "what it says now")
}

// Setting one aside says so, and hands you the next.
func TestSettingOneAsideSaysSoAndCarriesOn(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(9, "the boiler", squirrel.ItemOpen),
		note(8, "meter reading", squirrel.ItemOpen),
	}}
	routed(t, f).call(t, "POST", "/held/act",
		strings.NewReader("id=9&aside=waiting&from=thread"))

	require.Len(t, f.appended, 3)
	require.Contains(t, f.appended[1].Words, "Set aside")
	require.Contains(t, string(f.appended[2].Shown), "meter reading")
}

// The lid's one field, answered in the conversation. Both kinds of thing, and
// a result carries no verbs: it is a thing you went looking for rather than a
// thing you are deciding about.
func TestSearchingAnswersInTheConversation(t *testing.T) {
	f := &fakeStore{
		items:  []squirrel.Item{note(9, "the boiler makes a noise", squirrel.ItemKept)},
		chores: []squirrel.Chore{{ID: 1, Name: "bleed the boiler", Active: true, EveryDays: 30}},
	}
	routed(t, f).call(t, "POST", "/find", strings.NewReader("q=boiler"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, "the boiler makes a noise")
	require.Contains(t, shown, "bleed the boiler")
	require.NotContains(t, shown, `"acts"`, "a result is not a control")
}

// A result says which of the seven it is in. Without it an open task reports
// itself as being in the pile.
func TestAResultSaysWhereItIs(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 9, RawText: "ring the bank", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	routed(t, f).call(t, "POST", "/find", strings.NewReader("q=bank"))

	require.Contains(t, string(f.appended[1].Shown), "a task")
}

// Nothing found says so rather than drawing an empty answer.
func TestFindingNothingSaysSo(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/find", strings.NewReader("q=nothing"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "Nothing with that word")
}

// An empty field is not a question, and does not go into the record.
func TestSearchingForNothingAsksNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/find", strings.NewReader("q=+++"))

	require.Empty(t, f.appended)
}
