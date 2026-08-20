package coach_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// fakeFacts is the read surface. What is under test is the loop and the check
// on what comes back, not the SQL underneath — that lives in internal/boot's
// own integration tests.
type fakeFacts struct {
	clock  coach.Now
	work   []coach.Work
	fixed  coach.Fixed
	hasFix bool
	lately []coach.Happened
	item   coach.Work
	hasIt  bool
	err    error
	// minutes is what a measured duration comes back as, and zero stands in
	// for "not timed enough to say" — which must reach the model as an absence
	// rather than as a zero.
	minutes int
	labels  []string
	// asked names every tool that was actually run, so a test can assert the
	// caps were applied here rather than hoped for in the prompt.
	asked  []string
	limits []int
}

func (f *fakeFacts) Clock(context.Context, int64) (coach.Now, error) {
	f.asked = append(f.asked, "now")
	return f.clock, f.err
}

func (f *fakeFacts) OpenWork(_ context.Context, _ int64, limit int) ([]coach.Work, error) {
	f.asked = append(f.asked, "open_work")
	f.limits = append(f.limits, limit)
	return f.work, f.err
}

func (f *fakeFacts) NextFixed(context.Context, int64) (coach.Fixed, bool, error) {
	f.asked = append(f.asked, "next_fixed")
	return f.fixed, f.hasFix, f.err
}

func (f *fakeFacts) Lately(_ context.Context, _ int64, limit int) ([]coach.Happened, error) {
	f.asked = append(f.asked, "lately")
	f.limits = append(f.limits, limit)
	return f.lately, f.err
}

func (f *fakeFacts) Item(context.Context, int64, int64) (coach.Work, bool, error) {
	f.asked = append(f.asked, "item")
	return f.item, f.hasIt, f.err
}

func (f *fakeFacts) Typically(_ context.Context, _ int64, label string) (int, bool, error) {
	f.asked = append(f.asked, "typically")
	f.labels = append(f.labels, label)
	return f.minutes, f.minutes > 0, f.err
}

// toolAPI answers with a scripted sequence of turns, so a test can drive a
// two-round loop without a model.
type toolAPI struct {
	server *httptest.Server
	turns  []map[string]any
	sent   []map[string]any
}

func call(id, name string, args map[string]any) map[string]any {
	raw, _ := json.Marshal(args)
	return map[string]any{
		"id": id, "type": "function",
		"function": map[string]any{"name": name, "arguments": string(raw)},
	}
}

func turnOf(calls ...map[string]any) map[string]any {
	return map[string]any{"tool_calls": calls}
}

func said(text string) map[string]any { return map[string]any{"content": text} }

func newToolAPI(t *testing.T, turns ...map[string]any) *toolAPI {
	t.Helper()
	api := &toolAPI{turns: turns}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(raw, &parsed))
		api.sent = append(api.sent, parsed)

		turn := map[string]any{"content": "nothing scripted"}
		if len(api.sent) <= len(api.turns) {
			turn = api.turns[len(api.sent)-1]
		}
		message := map[string]any{"role": "assistant"}
		for k, v := range turn {
			message[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": message}},
			"usage":   map[string]any{"prompt_tokens": 1500, "completion_tokens": 60},
		}))
	}))
	t.Cleanup(api.server.Close)
	return api
}

func deciderFor(api *toolAPI, f *fakeFacts, log *fakeLog) *coach.Provider {
	p := coach.NewProvider(api.server.URL, "sk-test", "gpt-5.6-luna", "gpt-5.6-terra",
		coach.Budget{Log: log, CeilingMicros: 10_000_000})
	p.Clock = func() time.Time { return august }
	p.Facts = f
	return p
}

func TestDecideReadsThenChooses(t *testing.T) {
	f := &fakeFacts{
		clock: coach.Now{Clock: "14:00", PartOfDay: "afternoon", Capacity: "ok"},
		work: []coach.Work{
			{ID: 7, Kind: "task", Text: "ring the vet"},
			{ID: 3, Kind: "chore", Text: "put the bins out"},
		},
	}
	log := &fakeLog{}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", map[string]any{"limit": 5})),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 7, "because": "it takes two minutes and it is bothering you",
		})),
	)

	d, err := deciderFor(api, f, log).Decide(context.Background(), 1)
	require.NoError(t, err)

	require.Equal(t, "task", d.Kind)
	require.Equal(t, int64(7), d.RefID)
	// The text is the one the tool returned, not one the model retyped. A
	// model paraphrasing your own words back at you is the thing !fix exists
	// to keep it away from.
	require.Equal(t, "ring the vet", d.Text)
	require.Equal(t, "it takes two minutes and it is bothering you", d.Because)

	require.Equal(t, []string{"now", "open_work"}, f.asked)
	// Two round trips, and the whole loop billed as one decision.
	require.Len(t, api.sent, 2)
	require.Len(t, log.recorded, 1)
	require.Equal(t, "decide", log.recorded[0].Kind)
	require.Equal(t, "gpt-5.6-terra", log.recorded[0].Model)
	require.Equal(t, 3000, log.recorded[0].InTokens)
}

// The check the whole design rests on. A model that names a task it was never
// shown has invented one, and inventing work is the failure this product
// cannot ship.
func TestDecideRefusesAThingItWasNeverShown(t *testing.T) {
	f := &fakeFacts{work: []coach.Work{{ID: 7, Kind: "task", Text: "ring the vet"}}}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 99, "because": "it is the important one",
		})),
	)

	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// An offer that cannot say why it is the offer is a demand. The picker's own
// clause is better than none, and the picker is what answers here.
func TestDecideRefusesAChoiceWithNoReason(t *testing.T) {
	f := &fakeFacts{work: []coach.Work{{ID: 7, Kind: "task", Text: "ring the vet"}}}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{"kind": "task", "ref_id": 7, "because": "  "})),
	)

	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// Nothing downstream renders prose, so an answer in prose is no answer.
func TestDecideTreatsTalkingAsNoAnswer(t *testing.T) {
	api := newToolAPI(t, said("I think you should probably do the vet one."))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// A loop with no ceiling can spend a month's budget answering one question.
func TestDecideStopsAskingAfterThreeRounds(t *testing.T) {
	f := &fakeFacts{}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "lately", nil)),
		turnOf(call("c", "next_fixed", nil)),
		turnOf(call("d", "next_fixed", nil)),
	)

	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Len(t, api.sent, 3, "the loop ran past its ceiling")
}

// The whole loop is paid for even when it ends in the picker answering. A
// budget that only counted the successful ones would be wrong in the direction
// that spends money.
func TestDecideBillsALoopThatGaveUp(t *testing.T) {
	log := &fakeLog{}
	api := newToolAPI(t, said("no thanks"))

	_, err := deciderFor(api, &fakeFacts{}, log).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Len(t, log.recorded, 1)
	require.Greater(t, log.recorded[0].CostMicros, int64(0))
}

func TestDecideMakesNoCallWhenOverBudget(t *testing.T) {
	api := newToolAPI(t)
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{spent: 10_000_000}).
		Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent)
}

// The cap lives in the code, not in the prompt. A cap the model is asked to
// respect is a cap it can ignore.
func TestDecideCapsWhatAToolWillReturn(t *testing.T) {
	f := &fakeFacts{}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", map[string]any{"limit": 500})),
		turnOf(call("b", "lately", map[string]any{"limit": 0})),
	)

	_, _ = deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.Equal(t, []int{10, 10}, f.limits)
}

// A tool that cannot answer is a fact the model does not have. Telling it the
// database is unreachable invites it to say so to someone who asked what to do
// next.
func TestDecideSurvivesAToolThatFails(t *testing.T) {
	f := &fakeFacts{err: errors.New("no database")}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 7, "because": "anything",
		})),
	)

	// It still refuses, because nothing was handed over for choose to name —
	// which is the same protection, arrived at from the other direction.
	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Len(t, api.sent, 2, "a failing tool ended the loop early")
}

// Without facts there is nothing to read, so there is no decision to make and
// no call to pay for.
func TestDecideWithNoFactsAsksNothing(t *testing.T) {
	api := newToolAPI(t)
	p := coach.NewProvider(api.server.URL, "sk", "luna", "terra", coach.Budget{Log: &fakeLog{}})
	p.Clock = func() time.Time { return august }

	_, err := p.Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent)
}

func TestNoCoachDecidesNothing(t *testing.T) {
	_, err := coach.NoCoach{}.Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// Measured rather than guessed. A model's estimate is fine for a first run and
// should not survive a few real ones.
func TestTypicallyHandsBackAMeasuredDuration(t *testing.T) {
	f := &fakeFacts{minutes: 10}
	api := newToolAPI(t,
		turnOf(call("a", "typically", map[string]any{"label": "put the bins out"})),
		turnOf(call("b", "say", map[string]any{"text": "ok"})),
	)

	_, _ = deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)

	require.Equal(t, []string{"put the bins out"}, f.labels)

	sent, _ := api.sent[1]["messages"].([]any)
	var told string
	for _, m := range sent {
		msg, _ := m.(map[string]any)
		if msg["role"] == "tool" {
			told, _ = msg["content"].(string)
		}
	}
	require.Equal(t, `{"minutes":10}`, told)
}

// Absent rather than zero. Zero is a measurement and this is the absence of
// one, and a model told "0 minutes" will believe it.
func TestTooFewRunsReachTheModelAsAnAbsence(t *testing.T) {
	api := newToolAPI(t,
		turnOf(call("a", "typically", map[string]any{"label": "put the bins out"})),
		turnOf(call("b", "say", map[string]any{"text": "ok"})),
	)

	_, _ = deciderFor(api, &fakeFacts{}, &fakeLog{}).Decide(context.Background(), 1)

	sent, _ := api.sent[1]["messages"].([]any)
	for _, m := range sent {
		msg, _ := m.(map[string]any)
		if msg["role"] == "tool" {
			content, _ := msg["content"].(string)
			require.Equal(t, "{}", content)
			require.NotContains(t, content, "0")
		}
	}
}
