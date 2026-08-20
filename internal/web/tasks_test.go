package web

import (
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
	m := mounted(t, aDeciding())
	body := m.call(t, "GET", "/tasks", nil).Body.String()

	require.Contains(t, body, "ring the vet")
	require.NotContains(t, body, "buy milk", "a thought is not a decision")
	require.NotContains(t, body, "cancel the old insurance", "done is not still to do")

	// And the reverse: the pile does not show a task.
	pile := m.call(t, "GET", "/pile", nil).Body.String()
	require.Contains(t, pile, "buy milk")
	require.NotContains(t, pile, "ring the vet")
}

func TestTheArchiveHoldsWhatYouDid(t *testing.T) {
	body := mounted(t, aDeciding()).call(t, "GET", "/tasks/done", nil).Body.String()

	require.Contains(t, body, "cancel the old insurance")
	require.NotContains(t, body, "ring the vet")
}

// Two actions and only two. Dropping is absent: a task you no longer want is a
// note you no longer want, and it gets there by ceasing to be a task first.
func TestATaskOffersTwoWaysOut(t *testing.T) {
	body := mounted(t, aDeciding()).call(t, "GET", "/tasks", nil).Body.String()

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

	w := post(t, mounted(t, f), "/tasks/act",
		url.Values{"id": {"3"}, "act": {"open"}, "from": {"archive"}})

	require.Equal(t, "/tasks/done", w.Header().Get("Location"), "back where you pressed it")
	require.Equal(t, squirrel.ItemOpen, f.items[2].State)
	require.Equal(t, squirrel.ItemTask, f.items[2].Kind)
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

// A list of things to do is where a counter most wants to appear.
func TestTheTasksScreenNeverCounts(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 7; i++ {
		items = append(items, task(i, "a decided thing", squirrel.ItemOpen))
	}
	body := strings.ToLower(mounted(t, &fakeStore{items: items}).call(t, "GET", "/tasks", nil).Body.String())

	for _, total := range []string{"7 tasks", "7 things", "of 7", "(7)", "1 of ", "7 left"} {
		require.NotContains(t, body, total)
	}
}

// No deadline, nothing red, no urgency — a task is a thing to do, not a thing
// to be late for.
func TestATaskIsNeverLate(t *testing.T) {
	body := strings.ToLower(mounted(t, aDeciding()).call(t, "GET", "/tasks", nil).Body.String())

	for _, word := range []string{"overdue", "due", "late", "deadline", "urgent", "priority"} {
		require.NotContains(t, body, word)
	}
}

// Absence, not encouragement: an empty task list is a normal state and not a
// failure to set up.
func TestNothingDecidedDoesNotNag(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/tasks", nil).Body.String()

	require.Contains(t, body, "nothing decided")
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
