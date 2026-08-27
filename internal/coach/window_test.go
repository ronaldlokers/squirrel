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

	c.Add(1, "what now", "The envelope.", now.Add(-2*time.Minute))
	c.Add(1, "no, something else", "The bins, then.", now.Add(-time.Minute))

	recent := c.Recent(1, now)
	require.Len(t, recent, 2)
	require.Equal(t, "what now", recent[0].Said)
	require.Equal(t, "The bins, then.", recent[1].Replied)
}

func TestConversationsAreNotShared(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "mine", "yours", august)
	require.Empty(t, c.Recent(2, august))
}

// Bounded per person, which is what makes holding this in memory safe: at most
// WindowSize exchanges, and nothing past WindowAge. See window.go for the axis
// this is not bounded on.
func TestConversationsKeepOnlyTheNewestFew(t *testing.T) {
	c := coach.NewConversations()
	for i, said := range []string{"one", "two", "three", "four", "five"} {
		c.Add(1, said, "ok", august.Add(time.Duration(i)*time.Minute))
	}
	recent := c.Recent(1, august.Add(5*time.Minute))
	require.Len(t, recent, coach.WindowSize)
	require.Equal(t, "three", recent[0].Said)
}

func TestConversationsForgetAnOldConversationOnRead(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "this morning", "ok", august.Add(-3*time.Hour))
	require.Empty(t, c.Recent(1, august))
}

// Turning something down ends the conversation about it. Without this the next
// exchange would open carrying what you said the last time — which is a thing
// you have to think about before saying anything at all.
func TestForgetDropsTheConversation(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "what now", "The envelope.", august)
	c.Forget(1)
	require.Empty(t, c.Recent(1, august))
}

// The nil receiver is the no-coach build, and it must not panic on a path that
// only exists because a coach might have been there.
func TestNilConversationsAreSafe(t *testing.T) {
	var c *coach.Conversations
	require.NotPanics(t, func() {
		c.Add(1, "said", "replied", august)
		c.Forget(1)
		require.Empty(t, c.Recent(1, august))
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
		go func() { defer wg.Done(); c.Add(int64(i%3), "said", "replied", august) }()
		go func() { defer wg.Done(); c.Recent(int64(i%3), august) }()
	}
	wg.Wait()
}
