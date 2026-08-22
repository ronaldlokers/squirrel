package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Making a thing smaller.
//
// This is the one place in the product where a model returns a list, and it is
// allowed because **the list never renders**. It is stored, and the
// application hands back one step at a time. Pacing stays deterministic, which
// is the guarantee that keeps this from becoming the twelve-step plan the rest
// of the design spends its effort avoiding.
//
// The distinction is worth being exact about, because it is subtle and it is
// the whole safety argument: a model that *produces* a sequence is fine. A
// surface that *shows* a sequence is not. Someone who has just said they
// cannot start cannot read five steps; they can read one.

// mostSteps is the ceiling. Five, because past five it stops being a way in
// and becomes a project plan — and because a sequence you will not finish is
// a sequence that ends in a list of things you did not do.
const mostSteps = 5

// fewestSteps is two. One "step" is not a breakdown, it is the same task
// reworded, and rewording your own words back at you is the thing !fix exists
// to keep a model away from.
const fewestSteps = 2

// stepLongest is how long one step may be. Forty-four characters is about what
// fits on a phone without wrapping, and a step that needs two lines is a step
// that needs breaking down itself.
const stepLongest = 60

// smallerPreamble is what the model is told.
//
// "The first one must be doable in two minutes" is the load-bearing line.
// Every breakdown a model writes unprompted starts with something like
// "gather the necessary documents", which is the original task wearing a hat.
const smallerPreamble = `You are Buddy. Someone with ADHD cannot start one thing, and you are
breaking it into steps they will actually do.

Two to five steps. Each one is a single physical action, in plain words,
short enough to read in a glance.

The first one must be doable in two minutes, from where they are sitting,
without deciding anything. "Open the letter" — not "gather the documents".

Never number them. Never explain them. Never add encouragement.`

// stepsTool is how the sequence comes back. A tool call rather than prose,
// because prose has to be parsed and a parser is a second place for this to go
// wrong.
var stepsTool = []map[string]any{
	{
		"type": "function",
		"function": map[string]any{
			"name":        "steps",
			"description": "The sequence, first step first.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"steps": map[string]any{
						"type":     "array",
						"items":    map[string]any{"type": "string"},
						"maxItems": mostSteps,
					},
				},
				"required":             []string{"steps"},
				"additionalProperties": false,
			},
		},
	},
}

// Smaller breaks one thing into steps, or says it cannot.
//
// ErrUnavailable means the ladder's own fixed line stands, which it was doing
// on its own before this existed and does whenever this does not work. The
// line is already on screen when this is called — it is the first paint — so
// a failure here changes nothing and nobody finds out.
func (p *Provider) Smaller(ctx context.Context, personID int64, task, blocker string) ([]string, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return nil, ErrUnavailable
	}
	now := p.now()
	permit, err := p.Budget.Ask(ctx, personID, now, "the ladder answers")
	if err != nil {
		return nil, ErrUnavailable
	}
	// The gate is given back whichever way this returns. `defer` rather
	// than a release at the end, because the end is not the only exit.
	defer permit.Release()

	said := "The thing is: " + task
	if blocker != "" {
		// What is in the way changes what a good first step is: "too big"
		// wants the smallest visible piece, "don't know how" wants the thing
		// to find out first. The ladder already knows which it is, so the
		// model is told rather than asked.
		said += "\nWhat is in the way: " + blocker
	}

	_, calls, in, out, err := p.completionWithTools(ctx, permit, p.Deep, []chatMessage{
		{Role: "system", Content: smallerPreamble},
		{Role: "user", Content: said},
	}, stepsTool)

	if in+out > 0 {
		if err := p.Budget.Record(ctx, personID, Answer{
			Kind: "smaller", Model: p.Deep, Prompt: said,
			InTokens: in, OutTokens: out, Used: err == nil, At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}
	if err != nil {
		slog.Error("the coach making it smaller", "error", err)
		return nil, err
	}

	steps := stepsIn(calls)
	if len(steps) < fewestSteps {
		slog.Warn("the coach did not break it down; the ladder answers", "task", task)
		return nil, ErrUnavailable
	}
	return steps, nil
}

// stepsIn reads the sequence out of the tool call and holds it to its shape.
//
// The shape checks are the same argument guard.go makes about a reply, applied
// to a list: a numbered step is a plan admitting what it is, and a step too
// long to read at a glance is a step that needs breaking down itself. Anything
// failing takes the whole sequence with it — half a breakdown is worse than
// none, because the half you keep is the half that fitted.
func stepsIn(calls []toolCall) []string {
	for _, call := range calls {
		if call.Function.Name != "steps" {
			continue
		}
		var args struct {
			Steps []string `json:"steps"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return nil
		}

		out := make([]string, 0, mostSteps)
		for _, step := range args.Steps {
			s := strings.TrimSpace(step)
			if s == "" || len([]rune(s)) > stepLongest {
				return nil
			}
			if isListOrHeading(s) {
				// It numbered them after being told not to. Everything else it
				// was told is now in doubt too.
				return nil
			}
			out = append(out, s)
		}
		if len(out) > mostSteps {
			out = out[:mostSteps]
		}
		return out
	}
	return nil
}
