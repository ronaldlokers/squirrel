package web

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func aDeciding() *fakeStore {
	return &fakeStore{items: []squirrel.Item{
		task(1, "ring the vet about the booster", squirrel.ItemOpen),
		note(2, "buy milk", squirrel.ItemOpen),
		task(3, "cancel the old insurance", squirrel.ItemDone),
	}}
}

// The pile holds what you have not decided about; this holds what you have.
// Keeping both in one list would mean triage stopped meaning anything.
func TestTheTasksScreenHoldsOnlyWhatYouDecided(t *testing.T) {
	f := aDeciding()
	body := opened(t, f, "tasks")

	require.Contains(t, body, "ring the vet")
	require.NotContains(t, body, "buy milk", "a thought is not a decision")
	require.NotContains(t, body, "cancel the old insurance", "done is not still to do")

	// And the reverse: the pile does not show a task.
	pile := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()
	require.Contains(t, pile, "buy milk")
	require.NotContains(t, pile, "ring the vet")
}

// The archive was a screen; it is not one any more, and nothing has replaced
// it yet. What you did is still in the store and `!find` still reaches it — the
// thread will draw it when the pile joins the conversation in phase 3.
//
// This is left as a test of the store's own boundary rather than deleted, so
// the day somebody wires the archive back in there is something that already
// says what it should hold.
func TestWhatYouDidIsStillThereToDrawLater(t *testing.T) {
	f := aDeciding()
	done, _, err := f.ArchivedTasks(context.Background(), 1, 10)

	require.NoError(t, err)
	require.Len(t, done, 1)
	require.Equal(t, "cancel the old insurance", done[0].RawText)
}

// Two actions and only two. Dropping is absent: a task you no longer want is a
// note you no longer want, and it gets there by ceasing to be a task first.
func TestATaskOffersTwoWaysOut(t *testing.T) {
	body := opened(t, aDeciding(), "tasks")

	require.Contains(t, body, `value="done"`)
	require.Contains(t, body, `value="untask"`)
	require.NotContains(t, body, `value="drop"`)
	require.NotContains(t, body, `value="keep"`)
}

func TestDoingATaskArchivesIt(t *testing.T) {
	f := aDeciding()

	w := post(t, mounted(t, f), "/tasks/act", url.Values{"id": {"1"}, "act": {"done"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, squirrel.ItemDone, f.items[0].State)
	require.Equal(t, squirrel.ItemTask, f.items[0].Kind, "still a task, now done")
}

func TestUntaskingReturnsItToThePile(t *testing.T) {
	f := aDeciding()

	post(t, mounted(t, f), "/tasks/act", url.Values{"id": {"1"}, "act": {"untask"}})

	require.Equal(t, squirrel.ItemNote, f.items[0].Kind)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "undoing a decision is not finishing it")
}

// Every transition here reverses, including out of the archive.
func TestSomethingCanBeNotDoneAfterAll(t *testing.T) {
	f := aDeciding()

	// No "back where you pressed it" any more: there is one place, and it is
	// the conversation the press was made in.
	post(t, mounted(t, f), "/tasks/act", url.Values{"id": {"3"}, "act": {"open"}})

	require.Equal(t, squirrel.ItemOpen, f.items[2].State)
	require.Equal(t, squirrel.ItemTask, f.items[2].Kind)
	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "Back on the list")
}

func TestDecidingSomethingOutright(t *testing.T) {
	f := &fakeStore{}

	w := post(t, mounted(t, f), "/tasks/new", url.Values{"text": {"book the car in"}})

	require.Equal(t, 303, w.Code)
	require.Len(t, f.items, 1)
	require.Equal(t, "book the car in", f.items[0].RawText)
	require.Equal(t, squirrel.ItemTask, f.items[0].Kind, "it arrived decided")
}

func TestAnEmptyDecisionDoesNothing(t *testing.T) {
	f := &fakeStore{}

	post(t, mounted(t, f), "/tasks/new", url.Values{"text": {"   "}})

	require.Empty(t, f.items)
}

// Buddy counts them, and nothing else does.
//
// This forbade every shape of counter here, on the argument that a list of
// things to do is where one most wants to appear. Principle 2 was retired on
// 24 August 2026 and Buddy says how many. What is still refused is the shape
// that reads as a position in a queue or as an amount outstanding: "1 of 7"
// tells you where you are in a list you are behind on, and "7 left" is the same
// sentence with the target said out loud.
func TestTheTasksAreCountedAndNeverRanked(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 7; i++ {
		items = append(items, task(i, "a decided thing", squirrel.ItemOpen))
	}
	body := strings.ToLower(opened(t, &fakeStore{items: items}, "tasks"))

	require.Contains(t, body, "7 things you decided")
	for _, ranked := range []string{"of 7", "(7)", "1 of ", "7 left", "7 remaining"} {
		require.NotContains(t, body, ranked)
	}
}

// No deadline, nothing red, no urgency — a task is a thing to do, not a thing
// to be late for.
func TestATaskIsNeverLate(t *testing.T) {
	body := strings.ToLower(opened(t, aDeciding(), "tasks"))

	for _, word := range []string{"overdue", "due", "late", "deadline", "urgent", "priority"} {
		require.NotContains(t, body, word)
	}
}

// Absence, not encouragement: an empty task list is a normal state and not a
// failure to set up.
func TestNothingDecidedDoesNotNag(t *testing.T) {
	body := opened(t, &fakeStore{}, "tasks")

	require.Contains(t, strings.ToLower(body), "nothing decided")
	for _, nag := range []string{"get started", "add your first", "you should", "why not"} {
		require.NotContains(t, strings.ToLower(body), nag)
	}
}

// The fourth action on the card, beside the three disposals. Deciding is not
// disposing: it does not end the note, it moves it.
func TestTheDeckCanDecideANote(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "ring the vet", squirrel.ItemOpen)}}

	w := post(t, mounted(t, f), "/pile/act", url.Values{"id": {"1"}, "act": {"task"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, squirrel.ItemTask, f.items[0].Kind)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "deciding is not finishing")
}

// The tasks arrive as cards, with what you decided in them.
func TestOpeningTheTasksDrawsThem(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 3, RawText: "ring the bank", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks"))

	require.Len(t, f.appended, 2)
	require.Contains(t, string(f.appended[1].Shown), "ring the bank")
	require.Contains(t, string(f.appended[1].Shown), `"place":"the tasks"`)
}

// A task made from a photograph is a card with the photograph on it. Without
// it that is a card saying nothing, which is what a note with no words is.
func TestATaskWithNoWordsKeepsItsPhotograph(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 3, RawText: "", Kind: squirrel.ItemTask, State: squirrel.ItemOpen,
			PhotoName: "letter.jpg", PhotoType: "image/jpeg"},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks"))

	require.Contains(t, string(f.appended[1].Shown), "photo")
}

// Doing one says so, in the words the note actually holds.
func TestDoingATaskIsSaid(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 3, RawText: "ring the bank", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	routed(t, f).call(t, "POST", "/tasks/act", strings.NewReader("id=3&act=done"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[0].Words, "ring the bank")
}

// And putting one back is a different sentence: it is not finishing it.
func TestUntaskingSaysSomethingElse(t *testing.T) {
	items := func() []squirrel.Item {
		return []squirrel.Item{{ID: 3, RawText: "ring the bank",
			Kind: squirrel.ItemTask, State: squirrel.ItemOpen}}
	}
	did, back := &fakeStore{items: items()}, &fakeStore{items: items()}
	routed(t, did).call(t, "POST", "/tasks/act", strings.NewReader("id=3&act=done"))
	routed(t, back).call(t, "POST", "/tasks/act", strings.NewReader("id=3&act=untask"))

	require.Len(t, did.appended, 2)
	require.Len(t, back.appended, 2)
	require.NotEqual(t, did.appended[1].Words, back.appended[1].Words)
}

// A row that is not yours is not yours to act on. The handler used to write
// against a bare id; reading the row for its words scopes the write too.
func TestATaskThatIsNotYoursDoesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/tasks/act", strings.NewReader("id=99&act=done"))

	require.Empty(t, f.appended)
	require.Empty(t, f.states)
}

// Nothing decided yet is a sentence rather than an empty list.
func TestNoTasksSaysSo(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=tasks"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "Nothing decided yet")
}
