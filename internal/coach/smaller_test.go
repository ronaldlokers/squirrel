package coach_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// The one method that returns a list, and it is safe because nothing renders
// one: the sequence is stored and handed back a step at a time.

func stepsTurn(steps ...string) map[string]any {
	items := make([]any, 0, len(steps))
	for _, s := range steps {
		items = append(items, s)
	}
	return turnOf(call("a", "steps", map[string]any{"steps": items}))
}

func TestSmallerReturnsASequence(t *testing.T) {
	api := newToolAPI(t, stepsTurn(
		"Open the letter", "Find the reference number", "Ring the number on it"))
	log := &fakeLog{}
	p := deciderFor(api, &fakeFacts{}, log)

	steps, err := p.Smaller(context.Background(), 1, "the tax thing", "too big")
	require.NoError(t, err)
	require.Equal(t, []string{
		"Open the letter", "Find the reference number", "Ring the number on it",
	}, steps)

	// What is in the way changes what a good first step is, so the model is
	// told rather than asked.
	sent, ok := api.sent[0]["messages"].([]any)
	require.True(t, ok)
	last, ok := sent[len(sent)-1].(map[string]any)["content"].(string)
	require.True(t, ok)
	require.Contains(t, last, "the tax thing")
	require.Contains(t, last, "too big")

	require.Len(t, log.recorded, 1)
	require.Equal(t, "smaller", log.recorded[0].Kind)
	require.Equal(t, "gpt-5.6-terra", log.recorded[0].Model)
}

// One step is not a breakdown, it is the same task reworded — and rewording
// your own words back at you is what !fix exists to keep a model away from.
func TestSmallerRefusesASingleStep(t *testing.T) {
	api := newToolAPI(t, stepsTurn("Do the tax thing"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// It numbered them after being told not to. Everything else it was told is now
// in doubt too, so the whole sequence goes.
func TestSmallerRefusesANumberedPlan(t *testing.T) {
	api := newToolAPI(t, stepsTurn("1. Open the letter", "2. Ring the number"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// A step too long to read at a glance is a step that needs breaking down
// itself. Half a breakdown is worse than none, because the half you keep is
// the half that happened to fit.
func TestSmallerRefusesAStepTooLongToRead(t *testing.T) {
	api := newToolAPI(t, stepsTurn(
		"Open the letter",
		strings.Repeat("gather the necessary supporting documentation ", 3)))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// Past five it stops being a way in and becomes a project plan.
func TestSmallerKeepsAtMostFive(t *testing.T) {
	api := newToolAPI(t, stepsTurn("one", "two", "three", "four", "five", "six", "seven"))
	steps, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.NoError(t, err)
	require.Len(t, steps, 5)
}

// Talking instead of calling the tool is no sequence. Nothing here parses
// prose, because a parser is a second place for this to go wrong.
func TestSmallerTreatsProseAsNoAnswer(t *testing.T) {
	api := newToolAPI(t, said("Well, first you'd want to open the letter..."))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

func TestSmallerMakesNoCallWhenOverBudget(t *testing.T) {
	api := newToolAPI(t, stepsTurn("one", "two"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{spent: 10_000_000}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent)
}

func TestSmallerWithNothingToBreakDownAsksNothing(t *testing.T) {
	api := newToolAPI(t, stepsTurn("one", "two"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "  ", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent)
}

// The line every breakdown a model writes unprompted gets wrong: it starts
// with "gather the necessary documents", which is the original task in a hat.
func TestSmallerAsksForAFirstStepYouCanActuallyDo(t *testing.T) {
	api := newToolAPI(t, stepsTurn("Open the letter", "Ring the number"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Smaller(context.Background(), 1, "the tax thing", "too big")
	require.NoError(t, err)

	sent, _ := api.sent[0]["messages"].([]any)
	system, _ := sent[0].(map[string]any)["content"].(string)
	require.Contains(t, system, "doable in two minutes")
	require.Contains(t, system, "Never number them")
}

func TestNoCoachBreaksNothingDown(t *testing.T) {
	_, err := coach.NoCoach{}.Smaller(context.Background(), 1, "the tax thing", "too big")
	require.ErrorIs(t, err, coach.ErrUnavailable)
}
