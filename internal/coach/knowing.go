package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Coming to know somebody.
//
// The turns table is a complete record of everything that has happened on the
// screen, and reading it back is how a coach stops being a chatbot with a good
// preamble.
//
// Once a week, not once a turn: an observation worth keeping is one that survived
// a week of evidence, and a model asked "what have you noticed" after every
// message will notice something after every message.
//
// What it may conclude is bounded in three places: here, in knowingIn because a
// preamble is a request, and in squirrel.HoldToShape because that is the only
// place every writer passes through.

// mostKnown is how many observations a pass may produce. Six — a model asked
// for twenty things it has noticed will produce twenty, and the last fourteen
// will be invented.
const mostKnown = 6

// knowingPreamble is what the model is told, and every line of it is a refusal.
//
// The ban on "always" and "never" is the ban on counting written out: an
// absolute claim about somebody is a count with the number taken off.
//
// "How they work" rather than "what they are like" is the load-bearing
// distinction. The first produces "phone calls get done, forms get put off",
// which changes what Buddy offers on a Tuesday. The second produces
// "disorganised but well-meaning" — a diagnosis nobody asked for, and upsetting
// to find written down about you.
const knowingPreamble = `You are Buddy. You are reading back a record of your own conversations
with one person, to learn how they actually work.

Write at most six short observations about how they get things done —
what kinds of thing they finish, what they put off, when in the day they
do things, what wording helps, what does not.

Rules, all of them absolute:
- Never count anything. No numbers, no "always", "never", "every time".
- Describe how they work, never what they are like. No judgements about
  them as a person, no diagnosis, no praise.
- One plain sentence each, no longer than a line.
- Only what the record actually shows. If it shows little, say less.
  Two true observations are worth more than six invented ones.
- If the record shows nothing worth keeping, return no observations.`

// knowingTool is how the observations come back — a tool call rather than
// prose, for the reason stepsTool is: prose has to be parsed and a parser is a
// second place for this to go wrong.
var knowingTool = []map[string]any{
	{
		"type": "function",
		"function": map[string]any{
			"name":        "noticed",
			"description": "What the record shows about how this person works.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"observations": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required":             []string{"observations"},
				"additionalProperties": false,
			},
		},
	},
}

// Learn reads the record back and says what it shows, or says nothing.
//
// Nothing is a legitimate and common answer: a quiet week is a quiet week, and
// the empty result replaces what was there rather than leaving last week's in
// place. Keeping the old set because the new one was empty would make the
// record older than it claims to be.
//
// Deep, because this is the one call in the product where judgement is the
// entire output and it happens once a week. Everything else that uses the
// deeper model does so because a person is waiting; this does it because
// nobody is.
func (p *Provider) Learn(ctx context.Context, personID int64, record []string) ([]string, error) {
	if len(record) == 0 {
		return nil, ErrUnavailable
	}
	now := p.now()
	permit, err := p.Budget.Ask(ctx, personID, now, "reading the record back")
	if err != nil {
		return nil, ErrUnavailable
	}
	defer permit.Release()

	said := "Here is the record, oldest first.\n\n" + strings.Join(record, "\n")

	_, calls, in, out, err := p.completionWithTools(ctx, permit, p.Deep, []chatMessage{
		{Role: "system", Content: knowingPreamble},
		{Role: "user", Content: said},
	}, knowingTool)

	if in+out > 0 {
		if err := p.Budget.Record(ctx, personID, Answer{
			Kind: "knowing", Model: p.Deep, Prompt: "reading the record back",
			InTokens: in, OutTokens: out, Used: err == nil, At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}
	if err != nil {
		slog.Error("the coach reading the record back", "error", err)
		return nil, err
	}
	return knowingIn(calls), nil
}

// knowingIn reads the observations out of the tool call and holds them to
// their shape.
//
// One bad observation does not take the others with it, unlike stepsIn: these
// are independent facts, and dropping one loses nothing the others depended on.
// Half a breakdown is worse than none; half a set of observations is half a set
// of observations.
func knowingIn(calls []toolCall) []string {
	for _, call := range calls {
		if call.Function.Name != "noticed" {
			continue
		}
		var args struct {
			Observations []string `json:"observations"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return nil
		}
		out := make([]string, 0, mostKnown)
		for _, one := range args.Observations {
			s := strings.TrimSpace(one)
			if s == "" || isListOrHeading(s) {
				continue
			}
			out = append(out, s)
		}
		if len(out) > mostKnown {
			out = out[:mostKnown]
		}
		return out
	}
	return nil
}

// knowsYou is what the model is shown about the person it is talking to.
//
// Appended to the system prompt the same way badlyLanded is, and for the same
// reason: the sentences themselves are the instruction, and an instruction
// nobody can check is a wish.
//
// The last line is the one that keeps this safe. Without it a model handed a
// list of observations will demonstrate that it has them — "I know forms are
// hard for you" — which is being told what a machine has concluded about you,
// mid-sentence, when you asked about the bins.
func knowsYou(known []string) string {
	if len(known) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nWhat you have come to know about this person, from your own " +
		"conversations. Let it shape what you offer:\n")
	for _, one := range known {
		b.WriteString("- " + strings.TrimSpace(one) + "\n")
	}
	b.WriteString("Never say any of this back to them, and never mention that you know it.")
	return b.String()
}
