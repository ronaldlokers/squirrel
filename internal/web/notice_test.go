package web

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// One line about what is actually there.

func someTasks(n int) *fakeStore {
	items := []squirrel.Item{}
	for i := int64(1); i <= int64(n); i++ {
		items = append(items, task(i, "a thing about the car", squirrel.ItemOpen))
	}
	return &fakeStore{items: items}
}

// The door's own sentence first, then the model's. Principle 8 draws the line
// at authorship: the count is Squirrel's own fact and goes first.
func TestADoorMaySayWhatItNoticed(t *testing.T) {
	noticed.Clear()
	f := someTasks(3)
	c := &fakeCoach{reply: "these are all about the car"}
	m := mountedWith(t, f, c)

	f.appended = nil
	m.call(t, "GET", "/r/tasks", nil)
	said := f.appended[len(f.appended)-1].Words

	require.Contains(t, said, "3 things you decided")
	require.Contains(t, said, "all about the car")
	require.Less(t, strings.Index(said, "things you decided"), strings.Index(said, "about the car"),
		"the model spoke before the product did")
}

// Opening the same door twice costs one call, not two.
func TestTheSameSetIsOnlyNoticedOnce(t *testing.T) {
	noticed.Clear()
	f := someTasks(3)
	c := &fakeCoach{reply: "these are all about the car"}
	m := mountedWith(t, f, c)

	m.call(t, "GET", "/r/tasks", nil)
	m.call(t, "GET", "/r/tasks", nil)

	require.Len(t, c.asked, 1, "it paid twice for the same set")
}

func TestASetThatChangedIsNoticedAgain(t *testing.T) {
	noticed.Clear()
	f := someTasks(3)
	c := &fakeCoach{reply: "these are all about the car"}
	m := mountedWith(t, f, c)

	m.call(t, "GET", "/r/tasks", nil)
	f.items = append(f.items, task(9, "ring the dentist", squirrel.ItemOpen))
	m.call(t, "GET", "/r/tasks", nil)

	require.Len(t, c.asked, 2)
}

// With no key there is nothing to ask, and the door says what it always said.
func TestWithNoCoachADoorJustCounts(t *testing.T) {
	noticed.Clear()
	f := someTasks(3)
	m := routed(t, f)

	f.appended = nil
	m.call(t, "GET", "/r/tasks", nil)

	require.Equal(t, "3 things you decided.", f.appended[len(f.appended)-1].Words)
}

// One thing is not a set, and "one thing you notice about this one thing" is a
// description of it.
func TestOneCardIsNotWorthNoticingAnythingAbout(t *testing.T) {
	noticed.Clear()
	f := someTasks(1)
	c := &fakeCoach{reply: "should not be called"}
	m := mountedWith(t, f, c)

	m.call(t, "GET", "/r/tasks", nil)

	require.Empty(t, c.asked)
}

// A model that cannot be reached costs a missing sentence and nothing else.
// The cards are what you came for.
func TestADoorThatCannotAskStillOpens(t *testing.T) {
	noticed.Clear()
	f := someTasks(3)
	c := &fakeCoach{err: errTest}
	m := mountedWith(t, f, c)

	f.appended = nil
	m.call(t, "GET", "/r/tasks", nil)

	require.Equal(t, "3 things you decided.", f.appended[len(f.appended)-1].Words)
	require.Contains(t, string(f.appended[len(f.appended)-1].Shown), "a thing about the car")
}

// The bounds are the product's, not the model's, and they are checked rather
// than asked for. A prompt is a request; this is a rule.
func TestWhatAModelMaySayOnADoorIsBounded(t *testing.T) {
	for _, refused := range []string{
		"You should start with the car one",
		"Maybe start with the smallest.",
		"Try the dentist first",
		"Consider doing these together",
		"These are about the car. The dentist is the odd one out.",
		strings.Repeat("about the car ", 12),
		"   ",
	} {
		require.Empty(t, keepIfItIsALine(refused), "%q was allowed through", refused)
	}
	require.Equal(t, "these are all about the car",
		keepIfItIsALine("  these are all about the car  "))
}

// The chores say it too — the same machinery, the same bounds.
func TestTheChoresMaySayWhatTheyNoticed(t *testing.T) {
	noticed.Clear()
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bins out", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7},
		{ID: 2, Name: "recycling out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14},
	}}
	c := &fakeCoach{reply: "both of these are bin day"}
	m := mountedWith(t, f, c)

	f.appended = nil
	m.call(t, "GET", "/r/chores", nil)

	require.Contains(t, f.appended[len(f.appended)-1].Words, "both of these are bin day")
}

// It shipped as a sync.Map nothing ever evicted, growing by one entry per
// distinct set anybody ever looked at — the shape of leak discovered by a pod
// being OOM-killed on a quiet Tuesday months later.
func TestTheDoorCacheForgetsTheOldest(t *testing.T) {
	noticed.Clear()

	for i := range noticeKeep + 20 {
		noticed.Store("key-"+strconv.Itoa(i), "said "+strconv.Itoa(i))
	}

	require.Len(t, noticed.said, noticeKeep, "it kept everything")
	require.Len(t, noticed.order, noticeKeep)

	_, oldest := noticed.Load("key-0")
	require.False(t, oldest, "the oldest is still there")
	_, newest := noticed.Load("key-" + strconv.Itoa(noticeKeep+19))
	require.True(t, newest, "the newest was forgotten")
}

// Storing the same key twice does not grow the ring, or a door pressed a
// hundred times would evict everything else on its own.
func TestStoringTheSameSetTwiceDoesNotGrowIt(t *testing.T) {
	noticed.Clear()

	for range 200 {
		noticed.Store("one", "about the car")
	}

	require.Len(t, noticed.order, 1)
	said, ok := noticed.Load("one")
	require.True(t, ok)
	require.Equal(t, "about the car", said)
}

// An empty answer is remembered as an empty answer. Nothing worth saying is a
// result, and asking again would be paying twice for the same silence.
func TestNothingWorthSayingIsRememberedToo(t *testing.T) {
	noticed.Clear()
	f := someTasks(3)
	c := &fakeCoach{reply: "You should start with the car one"} // refused: advice
	m := mountedWith(t, f, c)

	m.call(t, "GET", "/r/tasks", nil)
	m.call(t, "GET", "/r/tasks", nil)

	require.Len(t, c.asked, 1, "it paid twice to be told nothing")
}

// It was `string(rune(personID))`, a rune conversion rather than a number, so
// every id above U+10FFFF and every negative one collapsed onto the same
// replacement character and shared a cache entry.
func TestEveryPersonGetsTheirOwnKey(t *testing.T) {
	set := []string{"the tax letter", "the boiler"}
	seen := map[string]bool{}

	for _, id := range []int64{1, 2, 65537, 1 << 40, -1, -2} {
		key := noticeKey(id, "the tasks", set)
		require.False(t, seen[key], "person %d shares a key with somebody else", id)
		seen[key] = true
		require.NotContains(t, key, "�", "person %d has an unreadable key", id)
	}
}

// A different set is a different key, which is what makes the cache correct
// rather than merely small.
func TestADifferentSetIsADifferentKey(t *testing.T) {
	a := noticeKey(1, "the tasks", []string{"the boiler", "the vet"})
	b := noticeKey(1, "the tasks", []string{"the vet", "the boiler"})
	c := noticeKey(1, "the chores", []string{"the boiler", "the vet"})

	require.NotEqual(t, a, b, "the order of the set does not change the key")
	require.NotEqual(t, a, c, "two doors share a key")
}
