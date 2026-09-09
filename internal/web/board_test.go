package web

import (
	"net/http"
	"strconv"
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

// The board is the chores. What is on it is three racks and the doors to the
// other three places, and the chores themselves are the only strips.
func TestTheBoardIsTheChoresAndThreeDoors(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/board", nil).Body.String()

	for _, want := range []string{"daily", "weekly", "seldom", "bins out"} {
		require.Contains(t, body, want)
	}
	for _, door := range []string{"the notes", "the tasks"} {
		require.Contains(t, body, door, "the %s cannot be reached from the board", door)
	}
	require.Contains(t, body, "what is coming",
		"the agenda stopped being a door and is not in the sidebar either")
	for _, behind := range []string{"boiler service code is 4471", "kaas", "vet about the booster"} {
		require.NotContains(t, body, behind,
			"%q is on the board, so the doors did not demote anything", behind)
	}
}

// And what is behind a door is behind it, whole.
func TestADoorOpensOntoWhatItHolds(t *testing.T) {
	m := mounted(t, aBoardStore())

	for door, want := range map[string]string{
		"notes": "boiler service code is 4471",
		"tasks": "vet about the booster",
	} {
		body := m.call(t, "GET", "/?bay="+door, nil).Body.String()
		require.Contains(t, body, want, "the %s door opens onto nothing", door)
		require.Contains(t, body, "back to the board", "there is no way out of the %s", door)
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

	body := m.call(t, "GET", "/?bay=notes", nil).Body.String()

	require.Contains(t, body, "what you set aside")
	require.Contains(t, body, "the things you kept")
	require.NotContains(t, body, "ledge\"><span class=\"tab\">what you set aside <span")
}

func TestAStripCarriesTheAnswersItsBayAllows(t *testing.T) {
	m := mounted(t, aBoardStore())

	board := m.call(t, "GET", "/board", nil).Body.String()
	for _, want := range []string{">did it<", ">later<"} {
		require.Contains(t, board, want)
	}

	notes := m.call(t, "GET", "/?bay=notes", nil).Body.String()
	for _, want := range []string{">done<", ">keep<", ">drop<"} {
		require.Contains(t, notes, want)
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
func TestWritingOnABlankStripKeepsTheWords(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	w := m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=meter+reading+48213"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/?bay=notes", w.Header().Get("Location"))
	require.Len(t, sp.written, 1, "the words were not kept")
	require.Equal(t, "meter reading 48213", sp.written[0].Text)
	require.Contains(t, f.inserted, "meter reading 48213",
		"the words were accepted and never written")
}

func TestWritingInTheTasksRackDecidesSomething(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=tasks&words=book+the+boiler+service"))

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

	body := m.call(t, "GET", "/?bay=notes", nil).Body.String()

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

	m.call(t, "POST", "/board/new", strings.NewReader("bay=weekly&words=defrost+the+freezer&every=14"))

	require.Equal(t, "defrost the freezer", f.reinterval.name)
	require.Equal(t, 14*24*time.Hour, f.reinterval.every)
	require.Empty(t, sp.written, "a chore with a rhythm is a chore, not a note")
}

// Words with no rhythm are a question, and the board asks it. They went to the
// notes until 3 September 2026, which is the wrong kind of helpful: you typed a
// chore and what you got was a note in another rack, found on the next refresh.
func TestChoreWordsWithNoRhythmAreAskedAboutRatherThanFiled(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	res := m.call(t, "POST", "/board/new", strings.NewReader("bay=weekly&words=defrost+the+freezer"))

	require.Empty(t, f.reinterval.name, "a chore was made without a rhythm")
	require.Empty(t, sp.written, "the words were filed as a note instead of being asked about")
	require.Equal(t, 303, res.Code)
	require.Equal(t, "/?bay=daily&rhythm=defrost+the+freezer", res.Header().Get("Location"),
		"the words were dropped rather than carried back to the question")
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
func TestAgendaWordsWithNoTimeInThemAreAskedAbout(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	w := m.call(t, "POST", "/board/new", strings.NewReader("bay=agenda&words=ring+the+dentist"))

	require.Empty(t, f.moments)
	require.Empty(t, sp.written, "an appointment with no time in it was filed as a note")
	require.Equal(t, "/?bay=agenda&when=ring+the+dentist", w.Header().Get("Location"))
}

func TestTheFrontDoorIsTheBoard(t *testing.T) {
	m := mounted(t, aBoardStore())

	front := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, front, `class="racks"`, "the front door is not the board")
	require.NotContains(t, front, `id="thread"`)
}

// On a phone the four racks become one and the bay signs become the tabs above
// it. The server draws all four either way — which rack you are in is a class,
// so the desktop board is untouched and the phone needs no script.
func TestTheRackYouAreInIsTheOneThatIsLit(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/?bay=weekly", nil).Body.String()

	require.Contains(t, body, `class="rack in" data-bay="weekly"`)
	require.Contains(t, body, `class="rack" data-bay="daily"`)
	require.Contains(t, body, `<a class="baytab in" href="/?bay=weekly" aria-current="page">`)
	require.Contains(t, body, `<a class="baytab" href="/?bay=daily">`)
}

// Nothing named lands you in "now", which is the cut across all three rather
// than any one of them. A phone that lands nowhere shows nothing at all.
func TestNowIsTheRackYouLandIn(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `class="rack in" data-bay="now"`)
}

// A press in a bay comes back to that bay. Answering a chore on a phone and
// being returned to the notes is the board losing your place, which on this
// surface is the whole complaint the redesign started from.
func TestAPressComesBackToTheBayItWasMadeIn(t *testing.T) {
	f := aBoardStore()
	m := mountedSpooling(t, f, &fakeSpool{})

	act := m.call(t, "POST", "/board/act", strings.NewReader("what=chore&id=7&answer=did&bay=weekly"))
	require.Equal(t, "/?bay=weekly", act.Header().Get("Location"))

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

// Opening the board spends nothing, and neither does pressing anything on it.
// A model wrote the pulled strip's clause between 3 and 4 September 2026, one
// call per newly picked thing, made inside the render. NOT TODAY invalidates
// that decision by design, so the next card could not be drawn until a tool
// loop of up to three round trips had finished — seconds, on the press that
// means "not this one". What the product notices is written in the margin
// instead, once a day, where nothing waits for it.
func TestTheBoardDrawsWithoutAskingAModel(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading", Because: "the oldest thing you decided to do"}
	c := &fakeCoach{}
	m := mountedWith(t, f, c)

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "the oldest thing you decided to do", "the picker's own clause is gone")
	require.Contains(t, body, "send the meter reading")
	require.NotContains(t, body, "it is five minutes", "a model was asked while the board was drawn")
	require.NotContains(t, body, "noticed", "the pulled strip still carries a mark")
	require.NotContains(t, body, `action="/board/badly"`,
		"there is a way to refuse a sentence no model wrote")
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

// An appointment is not answered the way work is: it happened, or it stopped
// mattering, and nothing anywhere records which of the two it was.
// The diary is read, not worked. A fixed point that has started leaves the
// list on its own — Upcoming only ever returns what is still ahead — so there
// is nothing to close from the sidebar, and closing one is what the pulled
// strip is for during the window where leaving matters.
func TestTheDiaryIsReadRatherThanWorked(t *testing.T) {
	f := aBoardStore()
	f.upcoming = []squirrel.Moment{{ID: 21, Label: "dentist", Starts: time.Now().Add(2 * time.Hour)}}
	m := mounted(t, f)

	body := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, "dentist")
	require.NotContains(t, body, `value="over"`, "the diary grew a control")

	// The way to close one is still open to anything that has an id for it.
	m.call(t, "POST", "/board/act", strings.NewReader("what=moment&id=21&answer=over"))
	require.Equal(t, []int64{21}, f.momentsDone)
}

// Making a chore out of a note needs a rhythm, and the note already exists —
// so the question is asked on the strip rather than in a field, and the strip
// that was asked about is the only one that shows it.
func TestMakingAChoreAsksForTheRhythmOnThatStripOnly(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	body := m.call(t, "GET", "/?bay=notes&chore=1", nil).Body.String()

	require.Contains(t, body, `name="every" value="7"`)
	require.Equal(t, 1, strings.Count(body, `action="/board/chore"`),
		"more than one strip is asking how often")
}

func TestPressingARhythmMakesTheChore(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/chore", strings.NewReader("id=1&every=14"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, int64(1), f.promoted.id)
	require.Equal(t, 14*24*time.Hour, f.promoted.every)
}

func TestANoteOffersToBecomeAChore(t *testing.T) {
	m := mounted(t, aBoardStore())

	body := m.call(t, "GET", "/?bay=notes", nil).Body.String()

	require.Contains(t, body, `formaction="/" name="chore" value="1"`)
}

// The same guard every press has: a note that is not yours is not yours to
// turn into a chore.
func TestPromotingWhatIsNotYoursMakesNothing(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items, squirrel.Item{
		ID: 99, RawText: "somebody else's note", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
	})
	f.notMine = map[int64]bool{99: true}
	m := mounted(t, f)

	m.call(t, "POST", "/board/chore", strings.NewReader("id=99&every=7"))

	require.Zero(t, f.promoted.id, "a note that is not yours became a chore")
}

// The ledge opens a shelf where the racks are, the way search does. Its links
// pointed at rooms until now, which meant the two shelves were the one part of
// the board that left the board.
func TestTheNotesShowEverythingWithTheUndecidedFirst(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items, squirrel.Item{ID: 41, RawText: "the boiler serial plate", State: squirrel.ItemKept, Kind: squirrel.ItemNote})
	f.aside = []squirrel.HeldItem{{ID: 42, Text: "chase the landlord", State: squirrel.ItemWaiting, Because: "he replies"}}
	m := mounted(t, f)

	rack := theRackIn(t, m.call(t, "GET", "/?bay=notes", nil).Body.String(), "bay=notes")
	undecided := strings.Index(rack, "boiler service code is 4471")
	aside := strings.Index(rack, "chase the landlord")
	kept := strings.Index(rack, "the boiler serial plate")

	require.Positive(t, undecided, "the notes needing a decision are not on the rack")
	require.Greater(t, aside, undecided, "what you set aside is above what needs deciding")
	require.Greater(t, kept, aside, "what you kept is above what you set aside")
	require.Contains(t, rack, `class="seam"`, "the three groups run together")
	require.NotContains(t, rack, `href="/?shelf=held"`, "the ledge is still there")

	require.Contains(t, rack, "he replies", "the rack does not say what would move it")
	require.Contains(t, rack, "back in the pile", "a settled strip carries no way back")
}

// A shelf never counts, and the sign is where that would show.
func TestAShelfCountsNothing(t *testing.T) {
	f := aBoardStore()
	f.items = append(f.items,
		squirrel.Item{ID: 41, RawText: "one kept thing", State: squirrel.ItemKept, Kind: squirrel.ItemNote},
		squirrel.Item{ID: 42, RawText: "another kept thing", State: squirrel.ItemKept, Kind: squirrel.ItemNote},
	)
	body := mounted(t, f).call(t, "GET", "/?shelf=kept", nil).Body.String()

	require.Contains(t, body, "the things you kept")
	require.NotContains(t, body, `the things you kept <span class="n"`)
}

// An empty rack and a rack that could not be read look identical, and one of
// them is a lie: if the database is down the board says so rather than showing
// a quiet morning.
func TestARackThatCannotBeReadSaysSo(t *testing.T) {
	m := mounted(t, &fakeStore{err: errTest})

	board := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, board, "cannot reach the chores")
	require.Contains(t, board, "nothing is lost")

	require.Contains(t, m.call(t, "GET", "/?bay=notes", nil).Body.String(),
		"cannot reach the notes", "an unreachable door looks exactly like an empty one")
}

// Every rack says it, not only the first. Three racks are one read of the
// chores, so a failure that reached one of them reached all three, and a
// quiet Sunday drawn in the other two would be the lie this guards against.
func TestEveryRackSaysSoWhenTheChoresCannotBeRead(t *testing.T) {
	body := mounted(t, &fakeStore{choresErr: errTest}).call(t, "GET", "/", nil).Body.String()

	require.Equal(t, 4, strings.Count(body, "cannot reach the chores"),
		"a rack drew itself empty on a read that failed")
}

// A rack says that there is more and never how much. What is further back is
// not a thing you can act on, so a number beside it would be counting what you
// have not got to.
func TestARackSaysThereIsMoreWithoutSayingHowMuch(t *testing.T) {
	f := aBoardStore()
	for i := int64(100); i < 160; i++ {
		f.items = append(f.items, squirrel.Item{
			ID: i, RawText: "note " + strconv.FormatInt(i, 10),
			State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now(),
		})
	}
	body := mounted(t, f).call(t, "GET", "/?bay=notes", nil).Body.String()

	require.Contains(t, body, "there is more further back")
	for _, count := range []string{"60 more", "more (", "of 63"} {
		require.NotContains(t, body, count)
	}
}

func TestARackThatHoldsEverythingSaysNothingAboutMore(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "there is more further back")
}

func TestThePickersMakeAMomentWithoutASentence(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/new",
		strings.NewReader("bay=agenda&words=dentist&day=2026-09-10&hour=14&minute=30"))

	require.Len(t, f.moments, 1, "the day and the time beside the field made nothing")
	require.Equal(t, "dentist", f.moments[0].Label)
	require.Equal(t, 2026, f.moments[0].Starts.Year())
	require.Equal(t, 14, f.moments[0].Starts.Hour())
	require.Equal(t, 30, f.moments[0].Starts.Minute())
	require.Empty(t, sp.written)
}

func TestThePickersMakeAMomentFromTheDayAndMonthFieldsRatherThanANativeDate(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return time.Date(2026, time.June, 1, 9, 0, 0, 0, time.UTC) }

	m.call(t, "POST", "/board/new",
		strings.NewReader("bay=agenda&words=dentist&dd=05&mo=09&hour=14&minute=30"))

	require.Len(t, f.moments, 1, "the day and month fields made nothing")
	require.Equal(t, "dentist", f.moments[0].Label)
	require.Equal(t, 2026, f.moments[0].Starts.Year(), "the year was not inferred as this year")
	require.Equal(t, time.September, f.moments[0].Starts.Month())
	require.Equal(t, 5, f.moments[0].Starts.Day())
	require.Equal(t, 14, f.moments[0].Starts.Hour())
	require.Equal(t, 30, f.moments[0].Starts.Minute())
	require.Empty(t, sp.written)
}

func TestAnIntervalCanBeAnyNumberOfDaysWeeksOrMonths(t *testing.T) {
	for _, one := range []struct {
		every, unit string
		want        time.Duration
	}{
		{"3", "days", 3 * 24 * time.Hour},
		{"5", "weeks", 35 * 24 * time.Hour},
		{"2", "months", 60 * 24 * time.Hour},
		{"9", "", 9 * 24 * time.Hour},
	} {
		f := aBoardStore()
		m := mountedSpooling(t, f, &fakeSpool{})
		m.call(t, "POST", "/board/new", strings.NewReader(
			"bay=weekly&words=descale+the+kettle&every="+one.every+"&unit="+one.unit))

		require.Equal(t, one.want, f.reinterval.every,
			"every %s %s came out as %v", one.every, one.unit, f.reinterval.every)
	}
}

func TestTheRackCarriesTheQuestionAndTheWordsBack(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/?bay=daily&rhythm=defrost+the+freezer", nil).Body.String()
	writer := theChoreWriter(t, body)

	require.Contains(t, writer, `value="defrost the freezer"`, "the words were not carried back")
	require.Contains(t, body, "how often does it come back?")

	when := mounted(t, aBoardStore()).call(t, "GET", "/?bay=agenda&when=ring+the+dentist", nil).Body.String()
	require.Contains(t, theRackIn(t, when, "bay=agenda"), "when is it?")
}

func TestARackAsksNothingWhenNothingWasAsked(t *testing.T) {
	rack := theChoreWriter(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String())

	require.NotContains(t, rack, "how often does it come back?")
	require.NotContains(t, rack, "when is it?")
}

// The board you are sent back to has the strip on it, because the row is
// written before the redirect rather than spooled for a drain to find.
func TestACaptureIsOnTheBoardYouAreSentBackTo(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=the+boiler+code"))

	require.Contains(t, f.inserted, "the boiler code", "the capture was not written")
	require.Contains(t, theRackIn(t, m.call(t, "GET", "/?bay=notes", nil).Body.String(), "bay=notes"),
		"the boiler code", "the strip is not behind the door that was drawn next")
}

// A write that fails says so rather than sending you back to a board that
// does not have your words on it.
func TestACaptureThatCannotBeKeptSaysSo(t *testing.T) {
	f := aBoardStore()
	f.kept = &fakeSpool{err: errTest}
	m := mounted(t, f)

	res := m.call(t, "POST", "/board/new", strings.NewReader("bay=notes&words=the+boiler+code"))

	require.Equal(t, 503, res.Code, "a lost capture was answered with a redirect")
}

func TestTheFacesAreOnTheDialAndAlwaysPressable(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `action="/board/mood"`, "the board cannot be told how you are")
	require.Equal(t, 5, strings.Count(body, `class="face" name="mood"`), "the five are not all there")
	for _, m := range []string{"good", "calm", "low", "frazzled", "wiped"} {
		require.Contains(t, body, `value="`+m+`"`, "the %s face is missing", m)
	}
	require.Contains(t, body, `<aside class="dial">`,
		"the faces are drawn somewhere other than the dial")
}

// The change: saying how you are stopped being something you wait to be asked
// for. The five are there whatever the last answer was; what stops is the ask.
func TestTheFacesStayWhenTheAskHasGone(t *testing.T) {
	f := aBoardStore()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Equal(t, 5, strings.Count(body, `class="face" name="mood"`),
		"you cannot change your mind about how you are until Squirrel asks again")
	require.NotContains(t, body, "how do you feel?",
		"it asks again while the last answer is still true")
}

func TestTheAskComesBackWhenTheAnswerHasGoneStale(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "how do you feel?",
		"a quiet day is a day the board never asks how you are")
}

// Today's face rides in the ops bar, which is what a phone presses to reach
// the dial at a width the dial does not fit.
func TestTodaysFaceIsInTheOpsBar(t *testing.T) {
	f := aBoardStore()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}

	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, `class="chip mood" href="/?bay=mood"`)
	require.Contains(t, body, `src="/static/mood-calm.png"`, "the chip does not wear today's face")

	dial := mounted(t, f).call(t, "GET", "/?bay=mood", nil).Body.String()
	require.Contains(t, dial, `<aside class="dial">`, "the chip leads nowhere")
	require.Contains(t, dial, "back to the board")
}

func TestAnsweringOnTheBoardKeepsAReadingAndSaysNothing(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	res := m.call(t, "POST", "/board/mood", strings.NewReader("mood=calm&bay=weekly"))

	require.Equal(t, 303, res.Code)
	require.Equal(t, "/?bay=weekly", res.Header().Get("Location"))
	require.Equal(t, squirrel.MoodCalm, f.recorded, "the reading was not kept")
	require.Empty(t, f.appended, "answering on the board wrote into the conversation")
}

func TestAWordThatIsNotOneOfTheFiveKeepsNothing(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	m.call(t, "POST", "/board/mood", strings.NewReader("mood=splendid"))

	// Nothing was written at all, rather than an empty reading written: the
	// store records whatever it is handed, so "no mood was kept" and "the
	// zero mood was kept" look the same from the outside unless this asks
	// whether the write happened.
	require.Nil(t, f.checkin, "something that is not one of the five was kept")
	require.Empty(t, f.recorded)
}

func TestASettledStripSaysWhyBesideItsWordsRatherThanInTheMark(t *testing.T) {
	f := aBoardStore()
	f.aside = []squirrel.HeldItem{{ID: 42, Text: "chase the landlord", State: squirrel.ItemWaiting, Because: "he replies"}}

	rack := theRackIn(t, mounted(t, f).call(t, "GET", "/?bay=notes", nil).Body.String(), "bay=notes")
	settled := rack[strings.Index(rack, "chase the landlord"):]
	settled = settled[:strings.Index(settled, "</article>")]

	require.Contains(t, settled, `<p class="state">waiting on he replies`,
		"the reason is not on the words, so it is forcing the mark column wide")
	require.NotContains(t, settled, `class="mark`,
		"a settled strip still carries a mark it cannot fit")
	require.Contains(t, settled, `class="backtab"`)
	require.Contains(t, settled, `aria-label="back in the pile"`,
		"the way back is a picture with no name")
}

// The appointment writer left the agenda door with the agenda, and pairs with
// the chore writer under the racks. The date pickers cannot be read at a
// sidebar's width, so it did not follow the diary into the sidebar.
func TestTheAppointmentWriterSitsBesideTheChoreWriter(t *testing.T) {
	body := mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String()
	writers := body[strings.Index(body, `class="writers"`):]
	writers = writers[:strings.Index(writers, "</div>")]

	require.Contains(t, writers, `class="newchore`, "the chore writer is not in the row")
	strip := body[strings.Index(body, `class="newmoment`):]
	strip = strip[:strings.Index(strip, "</form>")]

	require.Contains(t, strip, "asit", "the appointment inlet is not shaped like what it makes")
	require.Contains(t, strip, `class="holder"`, "it does not wear the agenda's holder")
	for _, want := range []string{`name="dd"`, `name="mo"`, `name="hour"`, `name="minute"`, `name="words"`} {
		require.Contains(t, strip, want, "%s is outside the strip", want)
	}
}

func TestOnlyTheAgendaIsShapedLikeItsThing(t *testing.T) {
	m := mounted(t, aBoardStore())
	board := m.call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, theChoreWriter(t, board), "asit", "the chore writer took the agenda's shape")
	require.NotContains(t, theChoreWriter(t, board), `name="dd"`, "the chore writer asks for a day")
	for _, door := range []string{"notes", "tasks"} {
		rack := theRackIn(t, m.call(t, "GET", "/?bay="+door, nil).Body.String(), "bay="+door)
		require.NotContains(t, rack, "asit", "the %s inlet took the agenda's shape", door)
		require.NotContains(t, rack, `name="dd"`, "the %s inlet asks for a day", door)
	}
	require.Contains(t, theChoreWriter(t, board), `class="inline"`,
		"the chores lost their interval")
}

// The notes are the only bay with a camera, so in a deployment that keeps
// photographs their blank strip posts here rather than to /board/new. Without
// this the most ordinary act on the board — writing a note down — was the one
// that did nothing until you reloaded.
func TestAPhotographCaptureIsKeptToo(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	m.call(t, "POST", "/board/capture", strings.NewReader("text=the+boiler+code"))

	require.Len(t, sp.written, 1, "the capture was not written")
	require.Equal(t, "the boiler code", sp.written[0].Text)
}

// The week behind today, on the board. Found by mutation: cutting the ring out
// of the dial left every other test green, which is the shape this project
// keeps shipping — built, drawn, documented and pinned by nothing.
func TestTheDialDrawsTheWeekBehindToday(t *testing.T) {
	at := time.Now()
	f := aBoardStore()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: at}
	f.readings = []squirrel.Checkin{
		{Mood: squirrel.MoodGood, SaidAt: at},
		{Mood: squirrel.MoodWiped, SaidAt: at.AddDate(0, 0, -2)},
	}

	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()
	dial := body[strings.Index(body, `<aside class="dial">`):]
	dial = dial[:strings.Index(dial, "</aside>")]

	require.Equal(t, 7, strings.Count(dial, "<circle"), "the ring is not a week")
	require.Contains(t, dial, `class="mgood"`, "today's reading is not on the ring")
	require.Contains(t, dial, `class="mwiped"`, "a reading from earlier in the week is not on the ring")
	require.Equal(t, 5, strings.Count(dial, `class="nought"`),
		"a day you said nothing is drawn as something, or not drawn at all")
	require.Contains(t, dial, strings.ToUpper(at.Format("Mon")), "an arc does not name its day")
	require.Contains(t, dial, "<title>"+strings.ToLower(at.Format("Monday 2 January"))+", good</title>",
		"an arc is a colour with no words, which is a picture you have to already know how to read")
}

// And it stops at the week. Six weeks is the page you go to by name.
func TestTheDialNeverDrawsMoreThanTheWeek(t *testing.T) {
	at := time.Now()
	f := aBoardStore()
	f.readings = []squirrel.Checkin{{Mood: squirrel.MoodLow, SaidAt: at.AddDate(0, 0, -20)}}

	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()
	dial := body[strings.Index(body, `<aside class="dial">`):]
	dial = dial[:strings.Index(dial, "</aside>")]

	require.Equal(t, 7, strings.Count(dial, "<circle"))
	require.NotContains(t, dial, `class="mlow"`, "the dial reached back past the week it draws")
}

// The seam, and what falls either side of it. Found by mutation: halving the
// rack before marginalia ran left the template rendering rows nobody had
// written the ask press onto, and every other test stayed green.
func TestARackIsSplitAtWhatWantsYouToday(t *testing.T) {
	f := aBoardStore()
	f.chores = []squirrel.Chore{
		{ID: 1, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 7, EverDone: true},
		{ID: 2, Name: "descale the kettle", Active: true, EveryDays: 7, SinceDays: 2, EverDone: true},
	}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()
	rack := theRackIn(t, body, "bay=weekly")

	asking := strings.Index(rack, "bins out")
	seam := strings.Index(rack, `class="seam"`)
	resting := strings.Index(rack, "descale the kettle")
	require.Positive(t, asking)
	require.Greater(t, seam, asking, "the seam is above the thing that is asking")
	require.Greater(t, resting, seam, "a resting row is above the seam")

	require.Contains(t, rack, `h-weekly answerable wants"`, "nothing in the rack is lifted")
	require.NotContains(t, rack[resting:], "wants", "a resting row is dressed as one that is asking")
}

// A rack's sign counts what is asking, not what it holds. It is a count of
// what wants you, which is the only kind this product allows.
func TestARackSignCountsWhatIsAskingToday(t *testing.T) {
	f := aBoardStore()
	f.chores = []squirrel.Chore{
		{ID: 1, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 7, EverDone: true},
		{ID: 2, Name: "descale the kettle", Active: true, EveryDays: 7, SinceDays: 2, EverDone: true},
		{ID: 3, Name: "sort the recycling", Active: true, EveryDays: 7, SinceDays: 1, EverDone: true},
	}
	rack := theRackIn(t, mounted(t, f).call(t, "GET", "/", nil).Body.String(), "bay=weekly")

	require.Contains(t, rack, `weekly <span class="n">1</span>`,
		"the sign is counting the rack rather than counting today")
}

// A resting row is its name and its rhythm. The reason and the usual time are
// still true and still one press away; they are not on the row.
func TestARestingRowCarriesItsNameAndItsRhythmAndNothingElse(t *testing.T) {
	f := aBoardStore()
	// Never done, so the ordering has a sentence for it — "you have not
	// started this one yet" — and it is still ranked last, which is exactly
	// the row that proves a resting row keeps its reason to itself.
	f.chores = []squirrel.Chore{
		{ID: 2, Name: "descale the kettle", Active: true, EveryDays: 14, SinceDays: 400},
	}
	rack := theRackIn(t, mounted(t, f).call(t, "GET", "/", nil).Body.String(), "bay=weekly")

	require.Contains(t, rack, "descale the kettle")
	require.Contains(t, rack, "every 2 weeks")
	require.NotContains(t, rack, "you have not started this one yet",
		"a row nobody is being asked about is explaining itself anyway")
	require.NotContains(t, rack, `class="why"`)
}

func TestOnlyAWeekdayChoreWearsTheLateMark(t *testing.T) {
	at := time.Date(2026, 9, 10, 9, 0, 0, 0, time.UTC) // a Thursday
	was := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = was })

	f := aBoardStore()
	f.chores = []squirrel.Chore{
		{ID: 1, Name: "bins out", Active: true, EveryDays: 7, SinceDays: 7, EverDone: true,
			Weekday: time.Thursday, Weeks: 1},
		{ID: 2, Name: "descale the kettle", Active: true, EveryDays: 7, SinceDays: 9, EverDone: true},
	}
	rack := theRackIn(t, mounted(t, f).call(t, "GET", "/", nil).Body.String(), "bay=weekly")

	bins := rack[strings.Index(rack, "bins out"):]
	kettle := bins[strings.Index(bins, "descale the kettle"):]
	bins = bins[:strings.Index(bins, "descale the kettle")]

	require.Contains(t, bins, `class="lateflag"`, "the lorry has been and the board says nothing")
	require.Contains(t, bins, "late &mdash; today")
	require.NotContains(t, kettle, `class="lateflag"`,
		"a chore on a rhythm is late, so Squirrel invented a time you can miss")
}
