package coach_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

func TestConversationsCarryTheLastFewExchanges(t *testing.T) {
	c := coach.NewConversations()
	now := august

	c.Add(1, "buddy", "what now", "The envelope.", now.Add(-2*time.Minute))
	c.Add(1, "buddy", "no, something else", "The bins, then.", now.Add(-time.Minute))

	recent := c.Recent(1, "buddy", now)
	require.Len(t, recent, 2)
	require.Equal(t, "what now", recent[0].Said)
	require.Equal(t, "The bins, then.", recent[1].Replied)
}

func TestConversationsAreNotShared(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "buddy", "mine", "yours", august)
	require.Empty(t, c.Recent(2, "buddy", august))
}

// Bounded per person and per room, which is what makes holding this in memory
// safe: at most WindowSize exchanges in each, and nothing past WindowAge. See
// window.go for the axis this is not bounded on.
func TestConversationsKeepOnlyTheNewestFew(t *testing.T) {
	c := coach.NewConversations()
	for i, said := range []string{"one", "two", "three", "four", "five"} {
		c.Add(1, "buddy", said, "ok", august.Add(time.Duration(i)*time.Minute))
	}
	recent := c.Recent(1, "buddy", august.Add(5*time.Minute))
	require.Len(t, recent, coach.WindowSize)
	require.Equal(t, "three", recent[0].Said)
}

func TestConversationsForgetAnOldConversationOnRead(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "buddy", "this morning", "ok", august.Add(-3*time.Hour))
	require.Empty(t, c.Recent(1, "buddy", august))
}

// Turning something down ends the conversation about it. Without this the next
// exchange would open carrying what you said the last time — which is a thing
// you have to think about before saying anything at all.
func TestForgetDropsTheConversation(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "buddy", "what now", "The envelope.", august)
	c.Forget(1, "buddy")
	require.Empty(t, c.Recent(1, "buddy", august))
}

// The nil receiver is the no-coach build, and it must not panic on a path that
// only exists because a coach might have been there.
func TestNilConversationsAreSafe(t *testing.T) {
	var c *coach.Conversations
	require.NotPanics(t, func() {
		c.Add(1, "buddy", "said", "replied", august)
		c.Forget(1, "buddy")
		require.Empty(t, c.Recent(1, "buddy", august))
	})
}

// Two surfaces can ask at once — chat drains on its own goroutine while the
// screen serves a request.
//
// The detector is this test's only assertion, so it is worth nothing unless
// the suite runs with -race. It did not: the Makefile's own comment now says
// why the flag is there, because a flag nobody can see the reason for is a
// flag somebody removes.
func TestConversationsSurviveConcurrentUse(t *testing.T) {
	c := coach.NewConversations()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		// Two rooms as well as three people: the map is nested now, and a
		// race on the inner map would be invisible to a test that only ever
		// touched one room.
		room := []string{"buddy", "chores"}[i%2]
		go func() { defer wg.Done(); c.Add(int64(i%3), room, "said", "replied", august) }()
		go func() { defer wg.Done(); c.Recent(int64(i%3), room, august) }()
	}
	wg.Wait()
}

// Two rooms are two conversations. Carrying what you said in the chores into
// the pile is this surface remembering across a boundary drawn on purpose,
// which is the thing rooms exist to stop.
func TestAConversationDoesNotLeakBetweenRooms(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "chores", "the bins", "Which bin.", august)

	require.Len(t, c.Recent(1, "chores", august), 1)
	require.Empty(t, c.Recent(1, "pile", august),
		"what was said in the chores came back in the pile")
}

// Ending one conversation ends one. The way out is per room for the same
// reason the window is: they are separate conversations.
func TestForgettingOneRoomLeavesTheOthers(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "chores", "the bins", "Which bin.", august)
	c.Add(1, "pile", "a letter", "Open it.", august)

	c.Forget(1, "chores")

	require.Empty(t, c.Recent(1, "chores", august))
	require.Len(t, c.Recent(1, "pile", august), 1,
		"forgetting one room forgot another")
}

// The bound is still a bound, and it is per room. Seven rooms of three is the
// same order as the three this held before.
func TestEachRoomIsBoundedOnItsOwn(t *testing.T) {
	c := coach.NewConversations()
	for i := 0; i < coach.WindowSize+4; i++ {
		c.Add(1, "pile", "said", "replied", august)
		c.Add(1, "chores", "said", "replied", august)
	}
	require.Len(t, c.Recent(1, "pile", august), coach.WindowSize)
	require.Len(t, c.Recent(1, "chores", august), coach.WindowSize)
}
