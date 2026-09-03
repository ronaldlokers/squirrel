package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// Noticing one thing about one thing.
//
// The board is read without anything having been said about most of it. What is
// worth saying is what a person cannot see by looking at a list — that three
// notes are about the same afternoon, that the number needed for one of them is
// written in another. Not a summary, not encouragement, and never a count.
//
// Bounded in three places, the same as knowing: here, in the tool's shape, and
// in what the caller will accept back.

// Thing is one row the model may say something about.
type Thing struct {
	Kind  string
	RefID int64
	Words string
}

// mostNoticed is how many lines a pass may produce.
//
// Two. A model asked for six observations about a rack will write six, and the
// last four will be restatements of the strip they are attached to — which is
// worse than silence, because a line that says nothing still has to be read.
const mostNoticed = 2

// noticingPreamble is what the model is told. Every line is a refusal.
const noticingPreamble = `You are reading somebody's board: the things they have written down and
not yet dealt with. You may write at most two short notes, each attached
to one of those things.

Only write a note when it says something the list does not already say.
The test is whether they could see it themselves by reading the list. If
they could, say nothing.

Good notes connect things: the detail one thing needs is written in
another, several of these are the same errand, this cannot be done until
that is. A note that restates its own thing is not a note.

Rules, all of them absolute:
- Never count anything. No numbers, no "always", "never", "again", "still".
- Never say anything about the person. Only about the things.
- Never tell them to do something, and never ask them a question.
- One plain sentence, no longer than a line.
- Nothing is better than something. Most boards deserve no notes at all.`

var noticingTool = []map[string]any{
	{
		"type": "function",
		"function": map[string]any{
			"name":        "notes",
			"description": "At most two notes, each about one thing on the board.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"notes": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"ref":   map[string]any{"type": "integer"},
								"words": map[string]any{"type": "string"},
							},
							"required":             []string{"ref", "words"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"notes"},
				"additionalProperties": false,
			},
		},
	},
}

// Notice reads the board and writes at most two lines about it, or none.
//
// None is the common answer and the preamble says so twice. Deep, and for the
// same reason Learn is: judgement is the whole output and nobody is waiting.
func (p *Provider) Notice(ctx context.Context, personID int64, things []Thing, refused []string) ([]Note, error) {
	if len(things) == 0 {
		return nil, ErrUnavailable
	}
	now := p.now()
	permit, err := p.Budget.Ask(ctx, personID, now, "reading the board")
	if err != nil {
		return nil, ErrUnavailable
	}
	defer permit.Release()

	var b strings.Builder
	b.WriteString("Here is the board.\n\n")
	for _, one := range things {
		fmt.Fprintf(&b, "%d (%s): %s\n", one.RefID, one.Kind, one.Words)
	}
	b.WriteString(refusedBefore(refused))

	_, calls, in, out, err := p.completionWithTools(ctx, permit, p.Deep, []chatMessage{
		{Role: "system", Content: noticingPreamble},
		{Role: "user", Content: b.String()},
	}, noticingTool)

	if in+out > 0 {
		if err := p.Budget.Record(ctx, personID, Answer{
			Kind: "noticing", Model: p.Deep, Prompt: "reading the board",
			InTokens: in, OutTokens: out, Used: err == nil, At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}
	if err != nil {
		slog.Error("the coach reading the board", "error", err)
		return nil, err
	}
	return noticedIn(calls, things), nil
}

// Note is one line about one thing.
type Note struct {
	Kind  string
	RefID int64
	Words string
}

// refusedBefore is what this person has already said was not useful.
//
// The lines themselves rather than a rule about them: "do not write notes like
// these" with the notes attached is checkable, where a rule derived from them
// is a guess about why they were refused.
func refusedBefore(refused []string) string {
	if len(refused) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nNotes like these were not useful. Do not write anything like them again:\n")
	for _, one := range refused {
		b.WriteString("- " + strings.TrimSpace(one) + "\n")
	}
	return b.String()
}

// noticedIn reads the notes out of the tool call and holds them to their shape.
//
// A note about something that was not on the board is dropped rather than
// kept against whatever happens to have that id: the model is answering about
// what it was shown, and an id it invented belongs to nothing.
func noticedIn(calls []toolCall, things []Thing) []Note {
	kinds := map[int64]string{}
	for _, one := range things {
		kinds[one.RefID] = one.Kind
	}
	for _, call := range calls {
		if call.Function.Name != "notes" {
			continue
		}
		var args struct {
			Notes []struct {
				Ref   int64  `json:"ref"`
				Words string `json:"words"`
			} `json:"notes"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return nil
		}
		out := make([]Note, 0, mostNoticed)
		for _, one := range args.Notes {
			words := strings.TrimSpace(one.Words)
			kind, known := kinds[one.Ref]
			if words == "" || !known || isListOrHeading(words) {
				continue
			}
			out = append(out, Note{Kind: kind, RefID: one.Ref, Words: words})
		}
		if len(out) > mostNoticed {
			out = out[:mostNoticed]
		}
		return out
	}
	return nil
}
