package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func aBoardStore() *fakeStore {
	return &fakeStore{
		items: []squirrel.Item{
			{ID: 1, RawText: "boiler service code is 4471", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
			{ID: 2, RawText: "kaas", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
			{ID: 3, RawText: "vet about the booster", State: squirrel.ItemOpen, Kind: squirrel.ItemTask, ReceivedAt: time.Now()},
		},
		chores: []squirrel.Chore{
			{ID: 7, Name: "bins out", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 7},
		},
	}
}

func TestTheBoardDrawsEveryBayFromTheStore(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, want := range []string{
		"the notes", "the chores", "the tasks", "the agenda",
		"boiler service code is 4471", "kaas", "bins out", "vet about the booster",
	} {
		require.Contains(t, body, want)
	}
}

func TestTheBoardIsNotAConversation(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, gone := range []string{"frombuddy", "fromyou", "whensaid", "class=\"dock\"", "bubble"} {
		require.NotContains(t, body, gone)
	}
}

func TestTheShelvesAreReachedFromTheNotesRackAndCountNothing(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, "what you set aside")
	require.Contains(t, body, "the things you kept")
	require.NotContains(t, body, "ledge\"><span class=\"tab\">what you set aside <span")
}

func TestAStripCarriesTheAnswersItsBayAllows(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, want := range []string{">done<", ">keep<", ">drop<", ">did it<", ">later<"} {
		require.Contains(t, body, want)
	}
}

func TestAnsweringAStripMovesItAndComesBackToTheBoard(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/act", strings.NewReader("what=note&id=1&answer=keep&bay=notes"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/?bay=notes", w.Header().Get("Location"))
	require.Equal(t, squirrel.ItemKept, f.states[1], "the note is kept")
}

func TestTheTrayHoldsWhatLeftTheBoardTodayAndOffersTheWayBack(t *testing.T) {
	f := aBoardStore()
	f.triaged = []squirrel.Item{
		{ID: 9, RawText: "washing machine one", State: squirrel.ItemDone, Kind: squirrel.ItemNote},
		{ID: 8, RawText: "the thing about the bike lights", State: squirrel.ItemDropped, Kind: squirrel.ItemNote},
	}
	m := mounted(t, f)

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, "washing machine one")
	require.Contains(t, body, "the thing about the bike lights")
	require.Contains(t, body, "put it back")
	require.NotContains(t, body, "tray\"><span class=\"sign\">today's tray</span> <span class=\"n\">")
}

// A row that is not yours is not yours to answer. SetItemState takes an item id
// and no person, so the guard has to be the read before it: the board looks the
// strip up as yours, and writes nothing when it is not.
func TestAStripThatIsNotYoursIsNotYoursToAnswer(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items, squirrel.Item{
		ID: 99, RawText: "somebody else's note", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
	})
	f.notMine = map[int64]bool{99: true}
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/act", strings.NewReader("what=note&id=99&answer=drop"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.NotContains(t, f.states, int64(99), "a strip that is not yours was answered")
}

func TestPuttingBackWhatIsNotYoursWritesNothing(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items, squirrel.Item{
		ID: 99, RawText: "somebody else's note", State: squirrel.ItemDropped, Kind: squirrel.ItemNote,
	})
	f.notMine = map[int64]bool{99: true}
	m := mounted(t, f)

	m.call(t, "POST", "/board/undo", strings.NewReader("id=99"))

	require.NotContains(t, f.states, int64(99), "a strip that is not yours was put back")
}

// A thought typed into the notes rack goes to the spool, not to the database.
// Capture is sacred: the words reach fsynced disk before anybody answers, and
// the drain resolves whose they are — the same path a capture from the
// conversation takes.
func TestWritingOnABlankStripSpoolsTheWords(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	w := m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=meter+reading+48213"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/?bay=notes", w.Header().Get("Location"))
	require.Len(t, sp.written, 1, "the words did not reach the spool")
	require.Equal(t, "meter reading 48213", sp.written[0].Text)
	require.Len(t, f.items, 3, "a note typed on the board went straight to the database")
}

func TestWritingInTheTasksRackDecidesSomething(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=tasks&words=book+the+boiler+service"))

	require.Empty(t, sp.written, "a task is a decision rather than a capture")
	var found bool
	for _, it := range f.items {
		if it.RawText == "book the boiler service" && it.Kind == squirrel.ItemTask {
			found = true
		}
	}
	require.True(t, found, "the words are not a task in the store")
}

func TestABlankStripWithNothingOnItKeepsNothing(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=+"))

	require.Empty(t, sp.written)
	require.Len(t, f.items, 3, "an empty strip kept something")
}

func TestTheBlankStripIsAFieldYouCanType(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	require.Contains(t, body, `action="/board/new"`)
	require.Contains(t, body, `name="words"`)
	require.Contains(t, body, `value="notes"`)
}

// A chore is a name and a rhythm, and the rack asks for both in one press.
// There is no pending state anywhere: the rhythm is a button beside the field
// rather than a question asked after the words are already typed.
func TestTypingAChoreWithItsRhythmMakesOne(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=chores&words=defrost+the+freezer&every=14"))

	require.Equal(t, "defrost the freezer", f.reinterval.name)
	require.Equal(t, 14*24*time.Hour, f.reinterval.every)
	require.Empty(t, sp.written, "a chore with a rhythm is a chore, not a note")
}

// Words with no rhythm are still a thought, so they go where a thought goes.
// The one thing that may not happen is that they are dropped.
func TestChoreWordsWithNoRhythmGoToTheNotes(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=chores&words=defrost+the+freezer"))

	require.Empty(t, f.reinterval.name, "a chore was made without a rhythm")
	require.Len(t, sp.written, 1, "the words were dropped")
	require.Equal(t, "defrost the freezer", sp.written[0].Text)
}

func TestTypingWhenSomethingHappensMakesAMoment(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=agenda&words=tomorrow+at+14%3A30+dentist"))

	require.Len(t, f.moments, 1, "nothing was put in the agenda")
	require.Equal(t, "dentist", f.moments[0].Label)
	require.Empty(t, sp.written)
}

// An appointment needs a time in it. Without one there is nothing to be on time
// for, so the words are a note — and the board says where they went rather than
// swallowing them.
func TestAgendaWordsWithNoTimeInThemGoToTheNotes(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	w := m.call(t, "POST", "/board/new", strings.NewReader("bay=agenda&words=ring+the+dentist"))

	require.Empty(t, f.moments)
	require.Len(t, sp.written, 1)
	require.Equal(t, "/?bay=notes&kept=1", w.Header().Get("Location"))
}

func TestTheChoresRackAsksForARhythmBesideTheField(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, want := range []string{`value="chores"`, `name="every" value="1"`, `name="every" value="7"`, `name="every" value="14"`, `name="every" value="30"`} {
		require.Contains(t, body, want)
	}
}

// The flip, pinned. The front door is the board; the conversation kept its own
// address and every press made inside it comes back there rather than landing
// somebody on a board they did not ask for.
func TestTheFrontDoorIsTheBoardAndTheConversationHasItsOwnAddress(t *testing.T) {
	m := mounted(t, aBoardStore())

	front := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, front, `class="racks"`, "the front door is not the board")
	require.NotContains(t, front, `id="thread"`)

	room := m.call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, room, `id="thread"`, "the conversation lost its own address")

	w := m.call(t, "POST", "/pile/act", strings.NewReader("id=1&act=keep"))
	require.Equal(t, "/r/everything", w.Header().Get("Location"),
		"a press in the conversation landed on the board")
}

// On a phone the four racks become one and the bay signs become the tabs above
// it. The server draws all four either way — which rack you are in is a class,
// so the desktop board is untouched and the phone needs no script.
func TestTheBayYouAreInIsTheOneThatIsLit(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/?bay=chores", nil).Body.String()

	require.Contains(t, body, `class="rack in" data-bay="chores"`)
	require.Contains(t, body, `class="rack" data-bay="notes"`)
	require.Contains(t, body, `<a class="baytab in" href="/?bay=chores">`)
	require.Contains(t, body, `<a class="baytab" href="/?bay=notes">`)
}

func TestTheNotesAreTheBayYouLandIn(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `class="rack in" data-bay="notes"`)
}

// A press in a bay comes back to that bay. Answering a chore on a phone and
// being returned to the notes is the board losing your place, which on this
// surface is the whole complaint the redesign started from.
func TestAPressComesBackToTheBayItWasMadeIn(t *testing.T) {
	f := aBoardStore()
	m := mountedSpooling(t, f, &fakeSpool{})

	act := m.call(t, "POST", "/board/act", strings.NewReader("what=chore&id=7&answer=did&bay=chores"))
	require.Equal(t, "/?bay=chores", act.Header().Get("Location"))

	made := m.call(t, "POST", "/board/new", strings.NewReader("bay=tasks&words=book+it"))
	require.Equal(t, "/?bay=tasks", made.Header().Get("Location"))
}

func TestThePulledStripCarriesItsThreeAnswers(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading", Because: "the oldest thing you decided to do"}
	m := mounted(t, f)

	body := m.call(t, "GET", "/", nil).Body.String()

	for _, want := range []string{`action="/board/now"`, `value="did"`, `value="later"`, `value="stuck"`} {
		require.Contains(t, body, want)
	}
}

func TestDoingTheOfferedThingRecordsIt(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 7, Text: "bins out"}
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/now", strings.NewReader("act=did&kind=chore&id=7"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Contains(t, f.answers, "did:chore", "the offered thing was not recorded as done")
}

func TestTurningTheOfferDownRefusesItForTheDay(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 7, Text: "bins out"}
	m := mounted(t, f)

	m.call(t, "POST", "/board/now", strings.NewReader("act=later&kind=chore&id=7"))

	require.Equal(t, []int64{7}, f.refused, "the offer was not turned down")
}

// Being stuck asks what is in the way, and the four answers are the product's
// own four. Nothing is stored between the question and the answer: which
// blocker you pressed is in the address, so a reload shows the same sentence
// rather than repeating a press.
func TestBeingStuckAsksWhatIsInTheWay(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/now", strings.NewReader("act=stuck&kind=task&id=3"))
	require.Equal(t, "/?stuck=1", w.Header().Get("Location"))

	body := m.call(t, "GET", "/?stuck=1", nil).Body.String()
	// The apostrophe arrives escaped, as it does in a browser, so the words are
	// matched by the half that carries no punctuation.
	for _, want := range []string{"too big", "know how", "boring", "not today"} {
		require.Contains(t, body, want)
	}
}

func TestAnAnswerToBeingStuckSaysOneSentence(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	m := mounted(t, f)

	body := m.call(t, "GET", "/?stuck=too+big", nil).Body.String()

	require.Contains(t, body, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
	require.NotContains(t, body, "know how", "the four answers are still on screen after one was pressed")
}

// "Not today" reached through being stuck is the same no as turning it down,
// and it has to leave the same mark.
func TestNotTodayFromTheLadderIsStillARefusal(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 7, Text: "bins out"}
	m := mounted(t, f)

	m.call(t, "POST", "/board/now", strings.NewReader("act=stuck&why=not+today&kind=chore&id=7"))

	require.Equal(t, []int64{7}, f.refused)
}

// Opening the board costs nothing. The picker's own clause is what the pulled
// strip says until you ask for better, which is the rule the coach has always
// been under: a surface that has to cost nothing to open may not spend a call.
func TestOpeningTheBoardSpendsNoCall(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading", Because: "the oldest thing you decided to do"}
	c := &fakeCoach{decision: &fakeDecision{kind: "task", refID: 3, text: "start with the meter", because: "it is five minutes"}}
	m := mountedWith(t, f, c)

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Equal(t, 1, c.peeked, "the board asked a model on load")
	require.Contains(t, body, "the oldest thing you decided to do")
	require.NotContains(t, body, "it is five minutes")
	require.Contains(t, body, `action="/board/buddy"`, "there is no way to ask")
}

// And asking is a press, whose answer is drawn on the thing it is about, under
// the acorn, with nowhere of its own to live.
func TestAskingBuddyDrawsHisLineOnThePulledStrip(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading", Because: "the oldest thing you decided to do"}
	c := &fakeCoach{decision: &fakeDecision{kind: "task", refID: 3, text: "start with the meter", because: "it is five minutes"}}
	m := mountedWith(t, f, c)

	w := m.call(t, "POST", "/board/buddy", strings.NewReader(""))
	require.Equal(t, "/?ask=1", w.Header().Get("Location"))

	body := m.call(t, "GET", "/?ask=1", nil).Body.String()
	require.Contains(t, body, "it is five minutes")
	require.Contains(t, body, `class="why buddy"`, "his line is not marked as his")
	require.Contains(t, body, `action="/board/badly"`, "there is no way to say it did not land")
}

// A model that answers nothing leaves the picker's clause standing, and no
// acorn: the mark says a model wrote this, so it may not appear when none did.
func TestNoAcornWhenNoModelAnswered(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading", Because: "the oldest thing you decided to do"}
	m := mountedWith(t, f, &fakeCoach{})

	body := m.call(t, "GET", "/?ask=1", nil).Body.String()

	require.Contains(t, body, "the oldest thing you decided to do")
	require.NotContains(t, body, `class="why buddy"`)
}

func TestSayingItLandedBadlyFromTheBoard(t *testing.T) {
	f := aBoardStore()
	m := mountedWith(t, f, &fakeCoach{})

	w := m.call(t, "POST", "/board/badly", strings.NewReader(""))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Len(t, f.landedBadly, 1, "nothing was marked as having landed badly")
}

func aSearchableStore() *fakeStore {
	f := aBoardStore()
	f.items = append(f.items,
		squirrel.Item{ID: 21, RawText: "the boiler serial plate is behind the panel", State: squirrel.ItemKept, Kind: squirrel.ItemNote},
		squirrel.Item{ID: 22, RawText: "the boiler pressure thing", State: squirrel.ItemOpen, Kind: squirrel.ItemNote},
	)
	f.chores = append(f.chores, squirrel.Chore{ID: 31, Name: "bleed the boiler", Active: true, EveryDays: 30, SinceDays: 4})
	return f
}

// Search is the only navigation on this board besides the four bays, so it
// takes the racks' place rather than opening anywhere else.
func TestSearchingTakesTheRacksPlace(t *testing.T) {
	m := mounted(t, aSearchableStore())

	body := m.call(t, "GET", "/?find=boiler", nil).Body.String()

	require.Contains(t, body, "the boiler serial plate is behind the panel")
	require.Contains(t, body, "the boiler pressure thing")
	require.Contains(t, body, "bleed the boiler", "chores are not searched")
	require.NotContains(t, body, `data-bay="tasks"`, "the racks are still drawn behind the results")
	require.Contains(t, body, "back to the board")
}

// Every state, on the same screen: something that left the pile is found, and
// what it carries is the way back rather than the four answers.
func TestAResultThatLeftThePileCarriesTheWayBack(t *testing.T) {
	m := mounted(t, aSearchableStore())

	body := m.call(t, "GET", "/?find=serial", nil).Body.String()

	require.Contains(t, body, "kept")
	require.Contains(t, body, `action="/board/undo"`)
	require.NotContains(t, body, `value="drop"`, "a note that already left is being offered the exits again")
}

func TestAnEmptySearchIsJustTheBoard(t *testing.T) {
	m := mounted(t, aSearchableStore())

	body := m.call(t, "GET", "/?find=+", nil).Body.String()

	require.Contains(t, body, `data-bay="tasks"`)
	require.NotContains(t, body, "back to the board")
}

func TestTheFindFieldIsAField(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `<form class="find" method="get" action="/">`)
	require.Contains(t, body, `name="find"`)
}
