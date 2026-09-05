package coach_test

import (
	"context"
	"encoding/json"
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
	// written is the rest of the board, which reaches the model as text.
	// writtenErr is separate from err: a board that cannot be read must not
	// stop the choosing.
	writtenErr error
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

		if why := refusedByTheRealAPI(parsed); why != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": why, "type": "invalid_request_error"},
			})
			return
		}

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
		coach.Budget{Log: log, CeilingFor: coach.FlatCeiling(10_000_000)})
	p.Clock = func() time.Time { return august }
	p.Facts = f
	return p
}

// refusedByTheRealAPI answers with what the live endpoint would have said, or
// empty when it would have accepted.
//
// It exists because this fake used to accept anything. It answered whatever
// the test scripted regardless of what was asked, so a request the real API
// rejects outright looked identical to one it likes — and two releases went
// out where every call 400'd while a hundred tests stayed green.
//
// Both rules are ones production taught us, quoted from the refusals it sent
// back. Anything else the real API dislikes has to be added the same way,
// which is the honest cost of not being able to reach it from a test.
func refusedByTheRealAPI(sent map[string]any) string {
	tools, _ := sent["tools"].([]any)
	if len(tools) == 0 {
		return ""
	}

	// "Function tools with reasoning_effort are not supported ... set
	// reasoning_effort to 'none'."
	if effort, _ := sent["reasoning_effort"].(string); effort != "none" {
		return "Function tools with reasoning_effort are not supported for this " +
			"model in /v1/chat/completions. To use function tools, use " +
			"/v1/responses or set reasoning_effort to 'none'."
	}

	// "Invalid schema for function 'now': None is not of type 'array'."
	for _, tool := range tools {
		fn, _ := tool.(map[string]any)["function"].(map[string]any)
		name, _ := fn["name"].(string)
		params, ok := fn["parameters"].(map[string]any)
		if !ok {
			return "Invalid schema for function '" + name + "': parameters is required."
		}
		required, present := params["required"]
		if !present || required == nil {
			return "Invalid schema for function '" + name + "': None is not of type 'array'."
		}
		if _, isArray := required.([]any); !isArray {
			return "Invalid schema for function '" + name + "': required must be an array."
		}
		if _, isObject := params["properties"].(map[string]any); !isObject {
			return "Invalid schema for function '" + name + "': properties must be an object."
		}
	}
	return ""
}
