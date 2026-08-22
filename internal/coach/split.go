package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Four thoughts captured as one note.
//
// Capture is sacred and stays exactly as it was: the raw text is stored
// verbatim, the moment it arrives, before anything looks at it. This is a
// triage step that runs long afterwards and only when asked — and what it
// produces is a **proposal**, never a rewrite. Nothing here can change a word
// you typed.
//
// Which is decision 8 stated as a property rather than an intention: Split
// returns strings to a caller that must render them and wait for a press. It
// has no way to write anything.

// Whether a note is worth asking about is not a model's question. Overwhelmed()
// already answers it — a brain dump and an overwhelm turn are the same shape,
// which is the point — so the same rule gates both and there is one definition
// of "this is several things" in the product.

// mostPieces is the ceiling. Four, because a note that is really six things is
// a note where the splitting is the smaller problem, and because four presses
// is already the most triage anyone will do in one go.
const mostPieces = 4

// pieceLongest is how long one piece may be. A piece longer than the note it
// came out of is not a piece; a piece near it is the model having reworded
// rather than split.
const pieceLongest = 120

// splitPreamble is what the model is told.
//
// "Use their words" is the load-bearing line, and it is why this is a cheap
// model's job rather than an expensive one's. The task is not to improve
// anything. It is to find the seams.
const splitPreamble = `Someone with ADHD typed several things at once. Separate them.

Use their own words. Do not tidy, expand, correct, or add anything.
Do not invent a thing they did not say. If it is really one thought,
return nothing.

Two to four pieces, each one thing.`

var splitTool = []map[string]any{
	{
		"type": "function",
		"function": map[string]any{
			"name":        "pieces",
			"description": "The separate things, in the order they were said. Empty if it is really one thought.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pieces": map[string]any{
						"type":     "array",
						"items":    map[string]any{"type": "string"},
						"maxItems": mostPieces,
					},
				},
				"required":             []string{"pieces"},
				"additionalProperties": false,
			},
		},
	},
}

// Split proposes the separate things in one note, or says it cannot.
//
// Luna, not Terra. Finding the seams in a sentence someone already wrote is
// not a judgement call — the judgement was theirs, when they typed it — and
// the routing matrix says so at the call site rather than through a classifier.
func (p *Provider) Split(ctx context.Context, personID int64, text string) ([]string, error) {
	text = strings.TrimSpace(text)
	if !Overwhelmed(text) {
		// The same rule that recognises the overwhelm turn, because a brain
		// dump and an overwhelm turn are the same shape. Free, and it is what
		// keeps this off the capture path's cost curve entirely.
		return nil, ErrUnavailable
	}
	now := p.now()
	permit, err := p.Budget.Ask(ctx, personID, now, "the note stays as it is")
	if err != nil {
		return nil, ErrUnavailable
	}
	// The gate is given back whichever way this returns. `defer` rather
	// than a release at the end, because the end is not the only exit.
	defer permit.Release()

	_, calls, in, out, err := p.completionWithTools(ctx, permit, p.Fast, []chatMessage{
		{Role: "system", Content: splitPreamble},
		{Role: "user", Content: text},
	}, splitTool)

	if in+out > 0 {
		if err := p.Budget.Record(ctx, personID, Answer{
			Kind: "split", Model: p.Fast, Prompt: text,
			InTokens: in, OutTokens: out, Used: err == nil, At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}
	if err != nil {
		slog.Error("the coach splitting a note", "error", err)
		return nil, err
	}

	pieces := piecesIn(calls, text)
	if len(pieces) < 2 {
		return nil, ErrUnavailable
	}
	return pieces, nil
}

// piecesIn reads the proposal and holds it to the one rule that matters: the
// pieces have to be made of the note.
//
// Checked rather than trusted, because "use their own words" is exactly the
// instruction a model is most willing to improve on. A piece whose words are
// not in the original is the model having written something — and something a
// model wrote must never be proposed as a thing you said.
func piecesIn(calls []toolCall, original string) []string {
	for _, call := range calls {
		if call.Function.Name != "pieces" {
			continue
		}
		var args struct {
			Pieces []string `json:"pieces"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return nil
		}

		lower := strings.ToLower(original)
		out := make([]string, 0, mostPieces)
		for _, piece := range args.Pieces {
			s := strings.TrimSpace(piece)
			if s == "" || len([]rune(s)) > pieceLongest {
				return nil
			}
			if !madeOf(lower, s) {
				return nil
			}
			out = append(out, s)
		}
		if len(out) > mostPieces {
			out = out[:mostPieces]
		}
		return out
	}
	return nil
}

// madeOf reports whether a piece is drawn from the note rather than written
// fresh.
//
// Word by word rather than as a substring, because splitting reorders nothing
// but it does drop the joins — "the bins and the vet" becomes "the bins" and
// "the vet", and neither is a substring once the "and" is gone. Every word has
// to appear in the original, which allows dropping and forbids inventing.
//
// Deliberately generous about punctuation and case: someone typing in a hurry
// does not capitalise, and a model tidying a full stop onto the end of a piece
// is not the failure being guarded against.
func madeOf(lowerOriginal, piece string) bool {
	for _, word := range strings.Fields(strings.ToLower(piece)) {
		word = strings.Trim(word, ".,;:!?\"'()")
		if word == "" {
			continue
		}
		if !strings.Contains(lowerOriginal, word) {
			return false
		}
	}
	return true
}
