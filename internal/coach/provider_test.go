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

// A stand-in for the provider, so the tests exercise the wire format rather
// than a mock of it. What comes back is a real HTTP response parsed by the real
// parser; only the far side is ours.
type fakeAPI struct {
	server   *httptest.Server
	requests []map[string]any
	reply    string
	status   int
	body     string
}

func newFakeAPI(t *testing.T, reply string) *fakeAPI {
	t.Helper()
	api := &fakeAPI{reply: reply, status: http.StatusOK}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(raw, &parsed))
		api.requests = append(api.requests, parsed)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(api.status)
		if api.body != "" {
			_, _ = io.WriteString(w, api.body)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"role": "assistant", "content": api.reply},
			}},
			"usage": map[string]any{"prompt_tokens": 2000, "completion_tokens": 200},
		}))
	}))
	t.Cleanup(api.server.Close)
	return api
}

func (a *fakeAPI) messages(t *testing.T, n int) []map[string]any {
	t.Helper()
	require.Greater(t, len(a.requests), n)
	raw, ok := a.requests[n]["messages"].([]any)
	require.True(t, ok)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		out = append(out, m.(map[string]any))
	}
	return out
}

func providerFor(api *fakeAPI, log *fakeLog) *coach.Provider {
	p := coach.NewProvider(api.server.URL, "sk-test", "gpt-5.6-luna", "gpt-5.6-terra",
		coach.Budget{Log: log, CeilingMicros: 10_000_000})
	p.Clock = func() time.Time { return august }
	return p
}

func TestProviderAnswersAndRecordsWhatItCost(t *testing.T) {
	api := newFakeAPI(t, "Start with the envelope.")
	log := &fakeLog{}
	p := providerFor(api, log)

	reply, err := p.Answer(context.Background(), coach.Turn{
		PersonID: 1,
		Kind:     "chat",
		Now:      coach.Now{Clock: "14:00", PartOfDay: "afternoon", Capacity: "ok"},
		Said:     "I can't face the tax thing",
	})
	require.NoError(t, err)
	require.Equal(t, "Start with the envelope.", reply.Text)
	require.Equal(t, "gpt-5.6-luna", reply.Model)

	require.Len(t, log.recorded, 1)
	require.True(t, log.recorded[0].Used)
	require.Equal(t, "chat", log.recorded[0].Kind)
	// What the person said, not the rendered conversation: the preamble is
	// identical every time and the window is already rows of this same table.
	require.Equal(t, "I can't face the tax thing", log.recorded[0].Prompt)
	require.Equal(t, int64(640), log.recorded[0].CostMicros)
}

// The ceiling has to be checked before the call. A ceiling checked afterwards
// is not a ceiling.
func TestProviderMakesNoCallWhenOverBudget(t *testing.T) {
	api := newFakeAPI(t, "Start with the envelope.")
	log := &fakeLog{spent: 10_000_000}
	p := providerFor(api, log)

	_, err := p.Answer(context.Background(), coach.Turn{PersonID: 1, Said: "help"})
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.requests)
	require.Empty(t, log.recorded)
}

// The step that is easy to get backwards: a reply the guard throws away was
// still paid for, so it is recorded before its shape is judged.
func TestProviderRecordsARejectedReplyAndStillRefusesIt(t *testing.T) {
	api := newFakeAPI(t, "Here's the plan:\n- open it\n- read it\n- reply")
	log := &fakeLog{}
	p := providerFor(api, log)

	_, err := p.Answer(context.Background(), coach.Turn{PersonID: 1, Kind: "chat", Said: "help"})
	require.ErrorIs(t, err, coach.ErrUnavailable)

	require.Len(t, log.recorded, 1)
	require.False(t, log.recorded[0].Used)
	require.Equal(t, int64(640), log.recorded[0].CostMicros)
}

func TestProviderRoutesToTheDeepModelWhenAsked(t *testing.T) {
	api := newFakeAPI(t, "Get in the shower. The other four are written down.")
	log := &fakeLog{}
	p := providerFor(api, log)

	reply, err := p.Answer(context.Background(), coach.Turn{
		PersonID: 1, Kind: "overwhelm", Deep: true, Said: "everything at once",
	})
	require.NoError(t, err)
	require.Equal(t, "gpt-5.6-terra", reply.Model)
	require.Equal(t, "gpt-5.6-terra", api.requests[0]["model"])
	// Terra is ten times the price, and the log has to say so or the month's
	// sum is wrong by that factor.
	require.Equal(t, coach.Cost("gpt-5.6-terra", 2000, 200), log.recorded[0].CostMicros)
}

func TestProviderTreatsAnErrorStatusAsNoCoach(t *testing.T) {
	api := newFakeAPI(t, "")
	api.status = http.StatusUnauthorized
	api.body = `{"error":{"message":"invalid api key","type":"invalid_request_error"}}`
	log := &fakeLog{}
	p := providerFor(api, log)

	_, err := p.Answer(context.Background(), coach.Turn{PersonID: 1, Said: "help"})
	require.ErrorIs(t, err, coach.ErrUnavailable)
	// Nothing was said, so nothing is recorded. A failed call has no tokens to
	// bill and inventing zero-cost rows would make the log lie about how often
	// the coach answered.
	require.Empty(t, log.recorded)
}

func TestProviderSurvivesAReplyThatIsNotJSON(t *testing.T) {
	api := newFakeAPI(t, "")
	api.status = http.StatusBadGateway
	api.body = "<html>gateway timeout</html>"
	p := providerFor(api, &fakeLog{})

	_, err := p.Answer(context.Background(), coach.Turn{PersonID: 1, Said: "help"})
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Contains(t, err.Error(), "502")
}

// The order is the design: system, then the window oldest first, then the state
// of now with the question. Now sits with the question rather than in the
// system message because the system message is the part worth caching and the
// clock changes every minute.
func TestProviderSendsTheWindowThenTheQuestion(t *testing.T) {
	api := newFakeAPI(t, "The bins, then.")
	p := providerFor(api, &fakeLog{})

	free := 25
	_, err := p.Answer(context.Background(), coach.Turn{
		PersonID: 1,
		Kind:     "chat",
		Now: coach.Now{
			Clock: "14:00", PartOfDay: "afternoon", Capacity: "ok", FreeUntil: &free,
		},
		Subject: "put the bins out",
		Said:    "no, something else",
		Recent: []coach.Exchange{
			{Said: "what now", Replied: "The envelope.", At: august.Add(-time.Minute)},
		},
	})
	require.NoError(t, err)

	msgs := api.messages(t, 0)
	require.Len(t, msgs, 4)
	require.Equal(t, "system", msgs[0]["role"])
	require.Contains(t, msgs[0]["content"], "Never produce a plan")

	require.Equal(t, "user", msgs[1]["role"])
	require.Equal(t, "what now", msgs[1]["content"])
	require.Equal(t, "assistant", msgs[2]["role"])
	require.Equal(t, "The envelope.", msgs[2]["content"])

	last, ok := msgs[3]["content"].(string)
	require.True(t, ok)
	require.Contains(t, last, "On screen: put the bins out")
	require.Contains(t, last, "It is 14:00")
	require.Contains(t, last, "the next fixed thing is in 25 minutes")
	require.Contains(t, last, "no, something else")
}

// Capacity is in every prompt already, so the plainer voice costs nothing
// extra. It is a signal rather than a neutral setting, and it will be read as
// one — which is why it is only ever sent on a low day.
func TestProviderAsksForAPlainerVoiceOnlyWhenCapacityIsLow(t *testing.T) {
	api := newFakeAPI(t, "Eat something.")
	p := providerFor(api, &fakeLog{})

	_, err := p.Answer(context.Background(), coach.Turn{
		PersonID: 1, Now: coach.Now{Capacity: "low"}, Said: "everything is too much",
	})
	require.NoError(t, err)
	require.Contains(t, api.messages(t, 0)[0]["content"], "drop warmth and character")

	_, err = p.Answer(context.Background(), coach.Turn{
		PersonID: 1, Now: coach.Now{Capacity: "ok"}, Said: "what now",
	})
	require.NoError(t, err)
	require.NotContains(t, api.messages(t, 1)[0]["content"], "drop warmth and character")
}

// An ignored prompt must cost a fraction of a cent rather than a full
// completion that is then thrown away by the guard.
func TestProviderCapsWhatTheModelMayProduce(t *testing.T) {
	api := newFakeAPI(t, "ok.")
	p := providerFor(api, &fakeLog{})

	_, err := p.Answer(context.Background(), coach.Turn{PersonID: 1, Said: "hello"})
	require.NoError(t, err)
	require.Equal(t, float64(200), api.requests[0]["max_completion_tokens"])
	// The older name is rejected outright by current models rather than merely
	// deprecated, so sending it would fail every call.
	require.NotContains(t, api.requests[0], "max_tokens")
}
