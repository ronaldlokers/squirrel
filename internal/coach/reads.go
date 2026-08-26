package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Reading what somebody typed into the box.
//
// The box has always been a capture slot: whatever you put in it becomes a
// note, and the answer is "Kept." That is the offline durability promise and
// it is the product's oldest guarantee — but it means typing into a
// conversation files a thing rather than saying it, which is why the thread
// still did not feel like one.
//
// So Buddy reads it. He answers whatever you typed, and he says whether it was
// a thought worth keeping or a question he has just dealt with.
//
// Ronald asked for this on 25 August 2026 knowing the risk, which was named at
// the time: a model between you and the capture promise can be wrong. What the
// application does with the answer is where that risk is actually managed —
// the words are kept first and dropped afterwards if he says so, so an
// unreachable model, a spent budget or a wrong answer all cost a note that is
// in the pile rather than a note that is gone. See captureHandler.

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
// The judgement is made before this is called now — by the model in the house,
// or by the rule under it — so what reaches here is already believed to be a
// question. It is asked anyway, and its answer wins: the caller was working
// from a one-word classification and this is the one that has read the whole
// sentence.
//
// So the tiers are: the rule for free, the house for a better guess, and this
// only for what survives both. Which is the argument the splitter and the
// interruption pre-filter are already built on — rules narrow, and the model
// answers the few that survive.
//
// ErrUnavailable means the box behaves exactly as it did before any of this:
// kept, and the kept wording said back. That is the floor, and it is the
// reason this can be built at all — every failure lands on the old guarantee
// rather than on a lost thought.
func (p *Provider) Reads(ctx context.Context, personID int64, said string, n Now) (string, bool, string, error) {
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

	// The state of the day, the same line every other turn gets. What somebody
	// types at eleven at night reads differently from the same words at nine
	// in the morning, and this is the one call that sees everything typed.
	asked := said
	if line := Context(n); line != "" {
		asked = line + "\n\n" + said
	}

	// Fast. This runs on everything typed into the box, which makes it the
	// most frequent call in the product by a wide margin — and telling a
	// thought from a question is not the kind of judgement the deeper tier
	// exists for.
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
// A reply that fails the shape is no reply, and no reply keeps the note —
// which is the direction every uncertainty in this file resolves in. The one
// thing that must never happen is a thought being thrown away because a model
// wrote a paragraph.
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
		// A place that is not one of them is no place. The same lookup the
		// acting path uses, for the same reason: a name nobody recognises
		// should be a miss here rather than an empty turn on the screen.
		if !places[args.Open] {
			return say, args.Keep, "", true
		}
		return say, args.Keep, args.Open, true
	}
	return "", true, "", false
}
