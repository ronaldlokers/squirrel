package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Reading what somebody typed into the box.
//
// The box has always been a capture slot: whatever you put in it becomes a note
// and the answer is "Kept." That is the offline durability promise, and it also
// means typing into a conversation files a thing rather than saying it.
//
// A model between you and that promise can be wrong. The risk is managed by the
// order rather than by the model: the words are kept first and dropped
// afterwards if it says so, so an unreachable model, a spent budget or a wrong
// answer all cost a note in the pile rather than a note that is gone. See
// captureHandler.

// readsPreamble is what the model is told.
//
// The distinction it has to make is narrower than it sounds, and the examples
// are load-bearing: almost everything typed into this box is a thought, and
// the failure that matters is calling one a question and filing nothing.
const readsPreamble = `You are Buddy. Someone with ADHD has typed something into the one box
this product has. Answer them, and say what the words were.

Two kinds, and when in doubt it is a thought:

A thought is anything they want out of their head — a task, a worry, a
number to remember, a half-formed thing. Keep it. Say something short
and real about it. Never just "noted".

A question is asked of you and answered by you. Only the answer matters
afterwards, so it is not kept.

When they ask to *see* something — their chores, their tasks, the pile,
the agenda, what they set aside, what they kept — set open to that place
and say one short line. The place itself is drawn for you underneath; do
not list it, and never say you cannot see it. You can.

One thing. Two sentences at most. Plain words.
Never say "should", "just", or "simply".
Never produce a plan, a checklist, or numbered steps.
Never tell them what to do with the note itself.`

// readsTool is how the two halves come back together: what to say, and whether
// the words were a thought.
//
// One call rather than two, because they are one judgement — "this was a
// question" and "here is the answer" are the same sentence being written.
var readsTool = []map[string]any{
	{
		"type": "function",
		"function": map[string]any{
			"name":        "answer",
			"description": "What to say back, and whether the words are a thought to keep.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"say": map[string]any{"type": "string"},
					"keep": map[string]any{
						"type":        "boolean",
						"description": "True for a thought. False only for a question you have just answered.",
					},
					// A field on the same call rather than a second tool, for
					// the reason the tool is one call in the first place: "this
					// was a request to see the chores" and "here is what to say
					// about it" are the same judgement being written once.
					"open": map[string]any{
						"type":        "string",
						"enum":        []string{"", "pile", "tasks", "chores", "at", "held", "kept"},
						"description": "A place to show them, when they asked to see one rather than asking about it. Empty otherwise.",
					},
				},
				"required":             []string{"say", "keep"},
				"additionalProperties": false,
			},
		},
	},
}

// Reads answers what was typed and says whether it was a thought.
//
// What reaches here is already believed to be a question — by the model in the
// house, or by the rule under it — and this answer wins: the caller was working
// from a one-word classification and this has read the whole sentence.
//
// ErrUnavailable means the box behaves as it did before any of this: kept, and
// the kept wording said back. Every failure lands on the old guarantee rather
// than on a lost thought.
func (p *Provider) Reads(ctx context.Context, personID int64, said string, n Now) (
	say string, keep bool, open string, err error) {

	said = strings.TrimSpace(said)
	if said == "" {
		return "", true, "", ErrUnavailable
	}
	now := p.now()
	permit, err := p.Budget.Ask(ctx, personID, now, "reading the box")
	if err != nil {
		return "", true, "", ErrUnavailable
	}
	defer permit.Release()

	// What somebody types at eleven at night reads differently from the same
	// words at nine in the morning.
	asked := said
	if line := Context(n); line != "" {
		asked = line + "\n\n" + said
	}

	// Fast: this runs on everything typed into the box, and telling a thought
	// from a question is not what the deeper tier exists for.
	_, calls, in, out, err := p.completionWithTools(ctx, permit, p.Fast, []chatMessage{
		{Role: "system", Content: System(n, "dock") + "\n\n" + readsPreamble},
		{Role: "user", Content: asked},
	}, readsTool)

	if in+out > 0 {
		if err := p.Budget.Record(ctx, personID, Answer{
			Kind: "dock", Model: p.Fast, Prompt: said,
			InTokens: in, OutTokens: out, Used: err == nil, At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}
	if err != nil {
		slog.Error("the coach reading the box", "error", err)
		return "", true, "", err
	}

	say, keep, open, ok := answerIn(calls)
	if !ok {
		return "", true, "", ErrUnavailable
	}
	return say, keep, open, nil
}

// answerIn reads the reply out of the tool call and holds it to its shape.
//
// A reply that fails the shape is no reply, and no reply keeps the note: a
// thought must never be thrown away because a model wrote a paragraph.
func answerIn(calls []toolCall) (say string, keep bool, open string, ok bool) {
	for _, call := range calls {
		if call.Function.Name != "answer" {
			continue
		}
		var args struct {
			Say  string `json:"say"`
			Keep bool   `json:"keep"`
			Open string `json:"open"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", true, "", false
		}
		say = strings.TrimSpace(args.Say)
		if say == "" || isListOrHeading(say) {
			return "", true, "", false
		}
		// A name nobody recognises is a miss here rather than an empty turn on
		// the screen. Same lookup the acting path uses.
		if !places[args.Open] {
			return say, args.Keep, "", true
		}
		return say, args.Keep, args.Open, true
	}
	return "", true, "", false
}
