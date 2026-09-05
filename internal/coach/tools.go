package coach

import (
	"context"
	"encoding/json"
	"fmt"
)

// toolSpecs is the wire description. It is written out rather than generated
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
}

// requiredArgs is empty for every tool but choose, whose arguments all matter.
// The read tools' arguments have sensible defaults, so requiring them would make
// the model spell out a limit it does not care about.
//
// Empty and never nil: a nil slice marshals to `null`, and the API rejects the
// whole request with `None is not of type 'array'`. That took down every call
// that offered tools, and the deterministic floor kept working underneath — which
// is why it looked like a quiet model rather than a broken one.
func requiredArgs(name string, props map[string]any) []string {
	required := []string{}
	if name != "choose" {
		return required
	}
	for k := range props {
		required = append(required, k)
	}
	return required
}

func spec(name, about string, props map[string]any) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	required := requiredArgs(name, props)
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

func key(kind string, id int64) string { return kind + ":" + fmt.Sprint(id) }

// answerTool runs one read tool and renders its result as JSON.
//
// Every failure is an empty result rather than an error. A tool that cannot
// answer is a fact the model does not have, and the alternative — telling the
// model the database is unreachable — invites it to say so to a person who
// asked what to do next.
func (p *Provider) answerTool(ctx context.Context, personID int64, room string, call toolCall, handed map[string]Work) string {
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
		// The room's own kind. Without this the chores ask what is open and
		// are handed a task, which tells the model the task exists — the fact
		// the room was drawn to keep out.
		work = onlyKind(room, work)
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
