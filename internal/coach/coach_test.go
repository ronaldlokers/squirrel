package coach_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// The zero value is the shipping configuration whenever no key is set, so it
// has to behave, not panic.
func TestNoCoachIsUsableAsTheZeroValue(t *testing.T) {
	var c coach.Coach = coach.NoCoach{}
	reply, err := c.Answer(context.Background(), coach.Turn{Said: "I can't start"})
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, reply.Text)
}

func TestTrimKeepsTheNewestFewExchanges(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	recent := []coach.Exchange{
		{Said: "one", At: now.Add(-4 * time.Minute)},
		{Said: "two", At: now.Add(-3 * time.Minute)},
		{Said: "three", At: now.Add(-2 * time.Minute)},
		{Said: "four", At: now.Add(-time.Minute)},
	}

	got := coach.Trim(recent, now)
	require.Len(t, got, coach.WindowSize)
	require.Equal(t, "two", got[0].Said)
	require.Equal(t, "four", got[2].Said)
}

// Half an hour, because a conversation from before lunch is not about now. The
// window is what lets the coach hear "no, something else"; it is not a memory.
func TestTrimDropsAnythingOlderThanTheWindow(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	recent := []coach.Exchange{
		{Said: "this morning", At: now.Add(-3 * time.Hour)},
		{Said: "an hour ago", At: now.Add(-time.Hour)},
		{Said: "just now", At: now.Add(-time.Minute)},
	}

	got := coach.Trim(recent, now)
	require.Len(t, got, 1)
	require.Equal(t, "just now", got[0].Said)
}

func TestTrimOnNothingIsNothing(t *testing.T) {
	require.Empty(t, coach.Trim(nil, time.Now()))
}
