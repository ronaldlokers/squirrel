package coach_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// What the tool schemas look like once they are JSON.
//
// These exist because the fake API in the other files accepts anything. It
// answers whatever the test scripted regardless of what was asked, so a
// request the real API rejects outright looks identical to one it likes — and
// a hundred passing tests said nothing about whether a single call would work.
//
// The bug that prompted them: `required` was built as a nil slice for every
// tool but one, which marshals to `null`. The API answered
// `None is not of type 'array'` and rejected the request, so every call that
// offered tools failed and every surface fell through to its deterministic
// answer. The floor worked perfectly, which is exactly why nobody could tell
// the model was never being reached.
//
// So: assert on the bytes, not on the map.

// sentSchemas pulls the tools out of a real request the provider built, by
// letting it make one against the fake API and reading what went on the wire.
func sentSchemas(t *testing.T, ask func(p *coach.Provider)) []map[string]any {
	t.Helper()

	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{"text": "ok"})))
	p := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{})
	ask(p)

	require.NotEmpty(t, api.sent, "nothing was sent")
	raw, ok := api.sent[0]["tools"].([]any)
	require.True(t, ok, "the request carried no tools")

	out := make([]map[string]any, 0, len(raw))
	for _, tool := range raw {
		out = append(out, tool.(map[string]any))
	}
	return out
}

func schemaOf(t *testing.T, tool map[string]any) (string, map[string]any) {
	t.Helper()
	fn, ok := tool["function"].(map[string]any)
	require.True(t, ok)
	name, _ := fn["name"].(string)
	params, ok := fn["parameters"].(map[string]any)
	require.True(t, ok, "%s has no parameters", name)
	return name, params
}

// The check that was missing. `required` has to survive the round trip as an
// array — including when it is empty, which is when a nil slice betrays you.
func TestEveryToolSchemaSurvivesJSON(t *testing.T) {
	both := [][]map[string]any{
		sentSchemas(t, func(p *coach.Provider) {
			_, _ = p.Answer(context.Background(), aTurn())
		}),
		sentSchemas(t, func(p *coach.Provider) {
			_, _ = p.Decide(context.Background(), 1)
		}),
	}

	for _, tools := range both {
		require.NotEmpty(t, tools)
		for _, tool := range tools {
			name, params := schemaOf(t, tool)

			required, present := params["required"]
			require.True(t, present, "%s has no required key", name)
			require.NotNil(t, required, "%s sent required as null, which the API refuses", name)
			_, isArray := required.([]any)
			require.True(t, isArray, "%s sent required as %T rather than an array", name, required)

			props, present := params["properties"]
			require.True(t, present, "%s has no properties key", name)
			_, isObject := props.(map[string]any)
			require.True(t, isObject, "%s sent properties as %T rather than an object", name, props)

			require.Equal(t, "object", params["type"], "%s is not an object schema", name)
			require.Equal(t, "function", tool["type"], "%s is not a function tool", name)
		}
	}
}

// The other three tool schemas are written out by hand rather than through
// spec(), so they get the same check by the same route.
func TestTheHandWrittenSchemasSurviveJSON(t *testing.T) {
	for name, ask := range map[string]func(p *coach.Provider){
		"smaller": func(p *coach.Provider) {
			_, _ = p.Smaller(context.Background(), 1, "the tax thing", "too big")
		},
		"split": func(p *coach.Provider) {
			_, _ = p.Split(context.Background(), 1, dump)
		},
		"interrupt": func(p *coach.Provider) {
			_, _ = p.ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
		},
	} {
		for _, tool := range sentSchemas(t, ask) {
			toolName, params := schemaOf(t, tool)

			required, present := params["required"]
			require.True(t, present, "%s/%s has no required key", name, toolName)
			_, isArray := required.([]any)
			require.True(t, isArray, "%s/%s sent required as %T", name, toolName, required)
		}
	}
}

// choose is the one tool whose arguments all matter, and it has to keep saying
// so — an empty required there would let a decision arrive with no id in it.
func TestChooseStillRequiresItsArguments(t *testing.T) {
	for _, tool := range sentSchemas(t, func(p *coach.Provider) {
		_, _ = p.Decide(context.Background(), 1)
	}) {
		name, params := schemaOf(t, tool)
		if name != "choose" {
			continue
		}
		required, _ := params["required"].([]any)
		require.ElementsMatch(t, []any{"kind", "ref_id", "because"}, required)
		return
	}
	t.Fatal("choose was never offered")
}

// The read tools require nothing, deliberately: their arguments all have
// sensible defaults, and requiring them would make the model spell out a limit
// it does not care about.
func TestTheReadToolsRequireNothing(t *testing.T) {
	for _, tool := range sentSchemas(t, func(p *coach.Provider) {
		_, _ = p.Decide(context.Background(), 1)
	}) {
		name, params := schemaOf(t, tool)
		if name == "choose" {
			continue
		}
		required, _ := params["required"].([]any)
		require.Empty(t, required, "%s asks the model for arguments it need not give", name)
	}
}

// Tools and reasoning cannot both be asked for on this endpoint. The API
// refuses the request outright — "Function tools with reasoning_effort are not
// supported ... set reasoning_effort to 'none'" — which was the second 400 in
// a row that only production could tell us about.
func TestEveryRequestWithToolsTurnsReasoningOff(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{"text": "ok"})))
	p := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{})
	_, _ = p.Answer(context.Background(), aTurn())

	require.Equal(t, "none", api.sent[0]["reasoning_effort"],
		"a request carrying tools did not turn reasoning off")
}

// And a turn with no tools does not mention it at all, so a provider that has
// never heard of the field is never shown it.
func TestARequestWithoutToolsSaysNothingAboutReasoning(t *testing.T) {
	api := newFakeAPI(t, "Start with the envelope.")
	p := coach.NewProvider(api.server.URL, "sk", "gpt-5.6-luna", "gpt-5.6-terra",
		coach.Budget{Log: &fakeLog{}})
	p.Clock = func() time.Time { return august }

	_, err := p.Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.NotContains(t, api.requests[0], "reasoning_effort")
}

// The same, for the three calls whose tools are written out by hand.
func TestTheHandWrittenCallsAlsoTurnReasoningOff(t *testing.T) {
	for name, ask := range map[string]func(p *coach.Provider){
		"smaller": func(p *coach.Provider) {
			_, _ = p.Smaller(context.Background(), 1, "the tax thing", "too big")
		},
		"split": func(p *coach.Provider) {
			_, _ = p.Split(context.Background(), 1, dump)
		},
		"interrupt": func(p *coach.Provider) {
			_, _ = p.ShouldInterrupt(context.Background(), 1, "put the bins out", afternoon)
		},
		"decide": func(p *coach.Provider) {
			_, _ = p.Decide(context.Background(), 1)
		},
	} {
		api := newToolAPI(t, turnOf(call("a", "say", map[string]any{"text": "ok"})))
		p := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{})
		ask(p)

		require.NotEmpty(t, api.sent, "%s sent nothing", name)
		require.Equal(t, "none", api.sent[0]["reasoning_effort"], "%s left reasoning on", name)
	}
}

// A last, blunt one: no `null` anywhere in a request. Every field this builds
// is either a value or absent, and null is neither — it is what a nil slice or
// map turns into on the way out, and it is what the API refused.
func TestNoRequestCarriesANull(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{"text": "ok"})))
	p := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{})
	_, _ = p.Answer(context.Background(), aTurn())

	raw, err := json.Marshal(api.sent[0])
	require.NoError(t, err)
	require.NotContains(t, string(raw), "null")
}
