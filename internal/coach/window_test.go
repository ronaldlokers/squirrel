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

// Bounded on both axes, which is what makes holding this in memory safe: at
// most WindowSize exchanges per person, and nothing past WindowAge.
func TestConversationsKeepOnlyTheNewestFew(t *testing.T) {
	c := coach.NewConversations()
	for i, said := range []string{"one", "two", "three", "four", "five"} {
		c.Add(1, said, "ok", august.Add(time.Duration(i)*time.Minute))
	}
	recent := c.Recent(1, august.Add(5*time.Minute))
	require.Len(t, recent, coach.WindowSize)
	require.Equal(t, "three", recent[0].Said)
}

// Trimming on read rather than only on write is what makes the age bound real.
// A conversation that stopped an hour ago is dropped when it is next asked
// for, not left waiting for a write that may never come.
func TestConversationsForgetAnOldConversationOnRead(t *testing.T) {
	c := coach.NewConversations()
	c.Add(1, "this morning", "ok", august.Add(-3*time.Hour))
	require.Empty(t, c.Recent(1, august))
}

// Closing the sheet has to mean something. A widget that remembers what you
// said last time is one you have to think about before opening.
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
// screen serves a request. Run with -race, which CI does.
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
