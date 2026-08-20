package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// The tool loop, and the one thing it is for: choosing what to do now.
//
// Six read tools and one way to finish. The model asks for what it needs, and
// then calls choose() with a thing it was actually handed. It cannot invent a
// task — the id has to have come back from a tool in this same call, and
// anything else is thrown away and the picker's answer used instead. That
// check is the reason a model is allowed near this decision at all.
//
// Round trips are capped. A loop with no ceiling is a loop that can spend a
// month's budget answering one question.

// maxRounds is how many times the model may ask for more before it has to
// decide. Three, which is what the architecture predicted it would need:
// what is open, what is coming, and how long one of them usually takes.
const maxRounds = 3

// toolSpecs is the wire description. It is written out rather than generated
// because there are seven of them and a schema generator would be more code
// than the schemas.
//
// Every description is one line. The model is told what a tool returns, not
// when to call it — when to call it is what it is for.
var toolSpecs = []map[string]any{
	spec("now", "The clock, the part of the day, capacity, and minutes until the next fixed thing.", nil),
	spec("open_work", "Tasks that are open and chores that are due, newest first. At most ten.",
		map[string]any{"limit": map[string]any{"type": "integer", "description": "How many, at most ten."}}),
	spec("next_fixed", "The next thing at a fixed time, and how long until you would have to leave.", nil),
	spec("lately", "What was already done today. At most ten.",
		map[string]any{"limit": map[string]any{"type": "integer", "description": "How many, at most ten."}}),
	spec("item", "One thing, by id.",
		map[string]any{"id": map[string]any{"type": "integer", "description": "The id."}}),
	spec("typically", "How many minutes something usually takes, measured from timers that finished. Absent when it has not been timed enough to say.",
		map[string]any{"label": map[string]any{"type": "string", "description": "What it is called."}}),
	spec("choose", "Hand over one thing to do now. Only ever something a tool returned in this conversation.",
		map[string]any{
			"kind":    map[string]any{"type": "string", "enum": []string{"task", "chore", "moment"}},
			"ref_id":  map[string]any{"type": "integer", "description": "The id it came back with."},
			"because": map[string]any{"type": "string", "description": "One clause, lower case, no full stop. Why this one."},
		}),
}

func spec(name, about string, props map[string]any) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	required := make([]string, 0, len(props))
	for k := range props {
		required = append(required, k)
	}
	// choose is the only tool whose arguments all matter; the read tools'
	// arguments all have sensible defaults, so requiring them would make the
	// model spell out a limit it does not care about.
	if name != "choose" {
		required = nil
	}
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": about,
			"parameters": map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             required,
				"additionalProperties": false,
			},
		},
	}
}

// Decision is what the model handed over.
//
// The same shape the picker's own answer has, because it has to be
// interchangeable with it: everything downstream renders one of these and does
// not know or care which produced it.
type Decision struct {
	Kind    string
	RefID   int64
	Text    string
	Because string
}

// decidePreamble is the whole of what the model is told about the job.
//
// The last two lines are the ones that matter. "Only something a tool
// returned" is enforced in code as well, and saying it here is what keeps the
// model from wasting a round trip on an answer that will be thrown away.
// "Prefer the fixed time" is the product's own rule 1, stated so the model
// agrees with the floor rather than fighting it.
const decidePreamble = `You are Buddy. You hand one person with ADHD exactly one thing to do
now, and you say why in one clause.

Look at what is open, what is coming, and how long things take. Then call
choose with one of them.

Only ever choose something a tool returned in this conversation.
Prefer something at a fixed time. Then something short. On a low-capacity
day prefer something bodily or something under five minutes.
The clause is lower case, one clause, no full stop, and it explains the
choice rather than describing the thing.`

// Decide asks the model what to do now.
//
// Everything about it degrades to the picker: no coach, no budget, a loop that
// runs out of rounds, a malformed call, or an id the model made up all end in
// ErrUnavailable, and the caller uses PickNow(). That is not a fallback bolted
// on — it is the same six rules that were the whole answer before this
// existed, and they are still the whole answer whenever this does not work.
func (p *Provider) Decide(ctx context.Context, personID int64) (Decision, error) {
	if p.Facts == nil {
		return Decision{}, ErrUnavailable
	}
	now := p.now()
	if ok, spent := p.Budget.Allows(ctx, personID, now); !ok {
		slog.Info("the coach is over its budget for the month; the picker chooses",
			"spent_micros", spent, "ceiling_micros", p.Budget.CeilingMicros)
		return Decision{}, ErrUnavailable
	}

	// Everything a tool handed back, so that choose() can be checked against
	// it. This is the reason the model cannot invent a task.
	handed := map[string]Work{}

	msgs := []chatMessage{{Role: "system", Content: decidePreamble}}
	if clock, err := p.Facts.Clock(ctx, personID); err == nil {
		if line := Context(clock); line != "" {
			msgs = append(msgs, chatMessage{Role: "user", Content: line})
		}
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: "What should I do now?"})

	var inTotal, outTotal int
	defer func() {
		// Recorded whatever happened, including the calls that ended in the
		// picker answering instead. A loop that gave up still cost what it
		// cost, and a budget that only counted the successes would be wrong in
		// the direction that spends money.
		if inTotal+outTotal > 0 {
			if err := p.Budget.Record(ctx, personID, Answer{
				Kind: "decide", Model: p.Deep, Prompt: "what should I do now?",
				InTokens: inTotal, OutTokens: outTotal, Used: true, At: now,
			}); err != nil {
				slog.Error("recording what the coach said", "error", err)
			}
		}
	}()

	for round := 0; round < maxRounds; round++ {
		reply, calls, in, out, err := p.completionWithTools(ctx, p.Deep, msgs, toolSpecs)
		inTotal += in
		outTotal += out
		if err != nil {
			slog.Error("the coach deciding", "error", err)
			return Decision{}, err
		}
		if len(calls) == 0 {
			// It answered in prose instead of choosing. Nothing here renders
			// prose, so there is nothing to do with it.
			slog.Warn("the coach talked instead of choosing; the picker chooses",
				"said", strings.TrimSpace(reply))
			return Decision{}, ErrUnavailable
		}

		msgs = append(msgs, chatMessage{Role: "assistant", Calls: calls})

		for _, call := range calls {
			if call.Function.Name == "choose" {
				d, ok := chosen(call, handed)
				if !ok {
					return Decision{}, ErrUnavailable
				}
				return d, nil
			}
			msgs = append(msgs, chatMessage{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    p.answerTool(ctx, personID, call, handed),
			})
		}
	}

	slog.Warn("the coach did not choose in time; the picker chooses", "rounds", maxRounds)
	return Decision{}, ErrUnavailable
}

// chosen validates the model's answer against what it was actually shown.
//
// A moment is the one kind that is allowed through without having been handed
// back by a tool listing work, because next_fixed returns it separately and it
// carries no id of its own that the model could confuse with a task's.
func chosen(call toolCall, handed map[string]Work) (Decision, bool) {
	var args struct {
		Kind    string `json:"kind"`
		RefID   int64  `json:"ref_id"`
		Because string `json:"because"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		slog.Warn("the coach chose something unreadable; the picker chooses", "error", err)
		return Decision{}, false
	}

	w, ok := handed[key(args.Kind, args.RefID)]
	if !ok {
		// The check the whole design rests on. A model that names a task it
		// was never shown has invented one, and inventing work is the failure
		// this product cannot ship.
		slog.Warn("the coach chose something it was never shown; the picker chooses",
			"kind", args.Kind, "id", args.RefID)
		return Decision{}, false
	}

	because := strings.TrimSpace(args.Because)
	if because == "" {
		// An offer that cannot say why it is the offer is a demand. The
		// picker's own clause is better than none, and the picker is what
		// answers when this returns false.
		slog.Warn("the coach chose without saying why; the picker chooses")
		return Decision{}, false
	}
	return Decision{Kind: w.Kind, RefID: w.ID, Text: w.Text, Because: because}, true
}

func key(kind string, id int64) string { return kind + ":" + fmt.Sprint(id) }

// answerTool runs one read tool and renders its result as JSON.
//
// Every failure is an empty result rather than an error. A tool that cannot
// answer is a fact the model does not have, and the alternative — telling the
// model the database is unreachable — invites it to say so to a person who
// asked what to do next.
func (p *Provider) answerTool(ctx context.Context, personID int64, call toolCall, handed map[string]Work) string {
	var args struct {
		Limit int    `json:"limit"`
		ID    int64  `json:"id"`
		Label string `json:"label"`
	}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)

	switch call.Function.Name {
	case "now":
		clock, err := p.Facts.Clock(ctx, personID)
		if err != nil {
			return "{}"
		}
		return asJSON(clock)

	case "open_work":
		work, err := p.Facts.OpenWork(ctx, personID, capped(args.Limit, workCap))
		if err != nil {
			return "[]"
		}
		remember(handed, work...)
		return asJSON(work)

	case "next_fixed":
		fixed, found, err := p.Facts.NextFixed(ctx, personID)
		if err != nil || !found {
			return "{}"
		}
		// A moment is choosable, so it has to be in what was handed over. It
		// has no id the model could confuse with a task's, so zero is the one
		// it is shown and the one it must give back.
		remember(handed, Work{Kind: "moment", Text: fixed.Label})
		return asJSON(fixed)

	case "lately":
		lately, err := p.Facts.Lately(ctx, personID, capped(args.Limit, latelyCap))
		if err != nil {
			return "[]"
		}
		return asJSON(lately)

	case "item":
		w, found, err := p.Facts.Item(ctx, personID, args.ID)
		if err != nil || !found {
			return "{}"
		}
		remember(handed, w)
		return asJSON(w)

	case "typically":
		mins, found, err := p.Facts.Typically(ctx, personID, args.Label)
		if err != nil || !found {
			// Absent rather than zero. Zero is a measurement and this is the
			// absence of one, and a model told "0 minutes" will believe it.
			return "{}"
		}
		return asJSON(map[string]int{"minutes": mins})
	}

	return "{}"
}

func remember(handed map[string]Work, work ...Work) {
	for _, w := range work {
		handed[key(w.Kind, w.ID)] = w
	}
}

func asJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
