package coach_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// The only thing in this product that speaks without being spoken to. What
// these tests are mostly about is the direction it can fail in.

func verdict(now bool, say, why string) map[string]any {
	return turnOf(call("a", "interrupt", map[string]any{"now": now, "say": say, "why": why}))
}

var afternoon = coach.Now{Clock: "14:00", PartOfDay: "afternoon", Capacity: "ok"}

func TestDecliningToInterruptStaysQuiet(t *testing.T) {
	api := newToolAPI(t, verdict(false, "", "they started something ten minutes ago"))
	log := &fakeLog{}

	say, ok := deciderFor(api, &fakeFacts{}, log).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.False(t, ok)
	require.Empty(t, say)

	require.Len(t, log.recorded, 1)
	require.Equal(t, "interrupt", log.recorded[0].Kind)
	require.Equal(t, "gpt-5.6-terra", log.recorded[0].Model)
}

func TestDecliningToInterruptDoesNotLogTheChoreName(t *testing.T) {
	logs := captureLogs(t)
	api := newToolAPI(t, verdict(false, "", "they started something ten minutes ago"))

	_, ok := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "call about the biopsy results", afternoon)
	require.False(t, ok)

	require.NotContains(t, logs.String(), "biopsy")
	require.Contains(t, logs.String(), "about_len")
}

func TestAgreeingCanSupplyTheWording(t *testing.T) {
	api := newToolAPI(t, verdict(true, "The bins go out tonight.", "collection is tomorrow"))

	say, ok := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok)
	require.Equal(t, "The bins go out tonight.", say)
}

// Empty wording with yes means the fixed message stands, which is what it was
// before this existed.
func TestAgreeingWithoutWordingLeavesTheFixedMessage(t *testing.T) {
	api := newToolAPI(t, verdict(true, "", ""))

	say, ok := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok)
	require.Empty(t, say)
}

// The wording is held to the same shape as everything else said here — and
// failing it costs the wording rather than the interruption, because the rules
// allowed it and the model agreed with them.
func TestWordingShapedLikeAPlanFallsBackToTheFixedMessage(t *testing.T) {
	api := newToolAPI(t, verdict(true, "Here's what to do:\n- one\n- two", ""))

	say, ok := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok)
	require.Empty(t, say)
}

// It fails open, and this is the direction that matters: a coach that is down
// must not silently turn off a feature that worked without one for months.
func TestEveryFailureLetsTheInterruptionThrough(t *testing.T) {
	// A reply that is not JSON.
	broken := newToolAPI(t)
	broken.turns = nil
	_, ok := deciderFor(broken, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok, "prose instead of a verdict stopped the nudge")

	// Over budget.
	api := newToolAPI(t, verdict(false, "", "no"))
	_, ok = deciderFor(api, &fakeFacts{}, &fakeLog{spent: 10_000_000}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok, "an exhausted budget stopped the nudge")
	require.Empty(t, api.sent)
}

// Staying quiet has to be something the model chose, never something a parse
// failure did on its behalf.
func TestAMalformedVerdictIsAYes(t *testing.T) {
	api := newToolAPI(t, turnOf(map[string]any{
		"id": "a", "type": "function",
		"function": map[string]any{"name": "interrupt", "arguments": "{not json"},
	}))

	_, ok := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok)
}

// With no coach at all every interruption goes out unchanged. Alone among
// these methods it does not answer "unavailable" — there is nothing to be
// unavailable for, because the rules already decided.
func TestNoCoachLetsEveryInterruptionThrough(t *testing.T) {
	say, ok := coach.NoCoach{}.ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
	require.True(t, ok)
	require.Empty(t, say)
}

// It is told the moment, not the pile. What it is deciding is whether now is a
// bad time to speak into, and what is on the pile is not evidence about that.
func TestItIsToldTheMomentAndTheThingAndNothingElse(t *testing.T) {
	api := newToolAPI(t, verdict(true, "", ""))
	_, _ = deciderFor(api, &fakeFacts{}, &fakeLog{}).
		ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)

	sent, ok := api.sent[0]["messages"].([]any)
	require.True(t, ok)
	require.Len(t, sent, 2)

	system, _ := sent[0].(map[string]any)["content"].(string)
	require.Contains(t, system, "Saying no is always allowed")

	said, _ := sent[1].(map[string]any)["content"].(string)
	require.Contains(t, said, "put the bins out")
	require.Contains(t, said, "It is 14:00")

	// One call and no tool loop. There is nothing to look up: the rules
	// already decided what this is about.
	require.Len(t, api.sent, 1)
}
