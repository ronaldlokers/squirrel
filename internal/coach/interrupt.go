package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Deciding to interrupt.
//
// The only thing in the product that speaks without being spoken to.
//
// The asymmetry is the whole safety argument, and it is enforced by where this
// sits rather than by anything it is told: the rules choose the candidates and
// the model only ever gets a veto. It is asked about something already due,
// already inside its asking window, its budget and past quiet hours. Nothing else
// is ever passed to it, so it can only make Squirrel quieter.
//
// The same arithmetic keeps it affordable: a model on every tick is 1,440 calls a
// day, and the rules narrow that to about five.

// interruptPreamble is what the model is told, and the last line is the one
// that matters. Saying nothing is the answer that must be easy to give.
const interruptPreamble = `You are Buddy. Something is due and you are deciding whether to
interrupt one person with ADHD about it, right now.

Interrupting badly costs more than staying quiet. If they are in the
middle of something, if it can wait, or if there is no good reason for
this to be now rather than later, say no.

If yes, say it in one short sentence, plainly, without urgency.
Saying no is always allowed and needs no reason.`

var interruptTool = []map[string]any{
	{
		"type": "function",
		"function": map[string]any{
			"name":        "interrupt",
			"description": "Whether to say something now, and what to say.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"now": map[string]any{"type": "boolean", "description": "False to stay quiet."},
					"say": map[string]any{"type": "string", "description": "One short sentence. Empty when staying quiet."},
					"why": map[string]any{"type": "string", "description": "For the log. One clause."},
				},
				"required":             []string{"now"},
				"additionalProperties": false,
			},
		},
	},
}

// ShouldInterrupt reports whether to say something, and what.
//
// The wording it returns is optional. An empty string with `true` means the
// deterministic message stands, which is what it was before this existed and
// what it goes back to whenever the model cannot answer.
//
// It **fails open**, unlike almost everything else here: if there is no coach,
// no budget, or the call errors, the interruption happens exactly as it did
// before this phase. That is deliberate and it is the opposite of the budget's
// own rule, for a reason worth stating — a failure here must not silently turn
// off the chore nudge, which is a feature that worked on its own for months.
// The model is an editor, and an absent editor means the piece runs as written.
func (p *Provider) ShouldInterrupt(ctx context.Context, personID int64, about string, n Now) (string, bool) {
	now := p.now()
	permit, err := p.Budget.Ask(ctx, personID, now, "the rules decide alone")
	if err != nil {
		return "", true
	}
	defer permit.Release()

	said := "Due now: " + about
	if line := Context(n); line != "" {
		said = line + "\n" + said
	}

	_, calls, in, out, err := p.completionWithTools(ctx, permit, p.Deep, []chatMessage{
		{Role: "system", Content: interruptPreamble},
		{Role: "user", Content: said},
	}, interruptTool)

	if in+out > 0 {
		if err := p.Budget.Record(ctx, personID, Answer{
			Kind: "interrupt", Model: p.Deep, Prompt: said,
			InTokens: in, OutTokens: out, Used: err == nil, At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}
	if err != nil {
		slog.Error("the coach deciding whether to interrupt", "error", err)
		return "", true
	}

	go2, say, why := interruptIn(calls)
	if !go2 {
		slog.Info("the coach decided not to interrupt", "about", about, "why", why)
		return "", false
	}
	return say, true
}

// interruptIn reads the verdict.
//
// A malformed answer is a yes with no wording, not a no. Staying quiet has to
// be something the model *chose*, never something a parse failure did on its
// behalf — otherwise a broken response silently disables a feature that
// worked without any model at all.
func interruptIn(calls []toolCall) (bool, string, string) {
	for _, call := range calls {
		if call.Function.Name != "interrupt" {
			continue
		}
		var args struct {
			Now bool   `json:"now"`
			Say string `json:"say"`
			Why string `json:"why"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return true, "", ""
		}
		if !args.Now {
			return false, "", strings.TrimSpace(args.Why)
		}
		// The wording is held to the same shape as everything else said here.
		// Failing it means the deterministic message stands, which is a better
		// outcome than not speaking at all — the interruption was allowed by
		// the rules and the model agreed with them.
		say, ok := Guard(args.Say)
		if !ok {
			return true, "", ""
		}
		return true, say, strings.TrimSpace(args.Why)
	}
	return true, "", ""
}
