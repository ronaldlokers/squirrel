package coach

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Provider is the coach with a model behind it.
//
// The order of operations is the whole design, and it is fixed here so that no
// caller can get it wrong:
//
//  1. Ask the budget. Over the ceiling means no call is made at all — a
//     ceiling checked after the call is not a ceiling.
//  2. Call the model.
//  3. Record what it cost, whether or not the answer survives. A reply the
//     guard throws away was still paid for.
//  4. Guard the shape. Failing means ErrUnavailable, and the caller uses its
//     deterministic answer.
//
// Step 3 before step 4 is deliberate and it is the step that is easy to get
// backwards.
type Provider struct {
	Client  *http.Client
	BaseURL string
	APIKey  string
	// Fast answers routine turns, Deep the ones where judgement matters. Which
	// is used is the caller's decision, carried on the Turn, because the caller
	// is the only thing that knows what kind of moment this is.
	Fast, Deep string
	Budget     Budget
	// Clock is time.Now unless a test says otherwise. The budget's month
	// boundary is the only thing that reads it.
	Clock func() time.Time
}

// Timeout is how long the coach may take before the deterministic answer is
// used instead.
//
// Fifteen seconds is generous for a two-sentence reply and mean for anything
// else, which is the right way round. The thing on the other side of this wait
// is a person who has already said they are stuck.
const Timeout = 15 * time.Second

// NewProvider builds a coach with its own HTTP client.
//
// Its own, rather than the default one: a model that hangs must not hold a
// screen open or a scheduler minute, and nothing else in this product should
// inherit a timeout chosen for a chat completion.
func NewProvider(baseURL, apiKey, fast, deep string, budget Budget) *Provider {
	return &Provider{
		Client:  &http.Client{Timeout: Timeout},
		BaseURL: baseURL,
		APIKey:  apiKey,
		Fast:    fast,
		Deep:    deep,
		Budget:  budget,
		Clock:   time.Now,
	}
}

func (p *Provider) now() time.Time {
	if p.Clock == nil {
		return time.Now()
	}
	return p.Clock()
}

// modelFor picks the tier. Two rather than one because the price between them
// differs tenfold and the need does too.
func (p *Provider) modelFor(t Turn) string {
	if t.Deep && p.Deep != "" {
		return p.Deep
	}
	return p.Fast
}

// Answer is a conversational turn.
func (p *Provider) Answer(ctx context.Context, t Turn) (Reply, error) {
	now := p.now()

	if ok, spent := p.Budget.Allows(ctx, t.PersonID, now); !ok {
		// Info, not error. Crossing the ceiling is the system working: the
		// picker chooses and the ladder answers for the rest of the month, and
		// nothing about the product stops.
		slog.Info("the coach is over its budget for the month; the deterministic answers take over",
			"spent_micros", spent, "ceiling_micros", p.Budget.CeilingMicros)
		return Reply{}, ErrUnavailable
	}

	model := p.modelFor(t)
	messages := p.messages(t)

	text, in, out, err := p.completion(ctx, model, messages)
	if err != nil {
		slog.Error("the coach", "kind", t.Kind, "model", model, "error", err)
		return Reply{}, err
	}

	guarded, usable := Guard(text)

	// Recorded before the guard's verdict is acted on, and recorded either
	// way. The prompt kept is what the person said rather than the whole
	// rendered conversation: the preamble is the same every time and the
	// window is already several rows of this same table.
	if err := p.Budget.Record(ctx, t.PersonID, Answer{
		Kind:      t.Kind,
		Model:     model,
		Prompt:    t.Said,
		Reply:     text,
		InTokens:  in,
		OutTokens: out,
		Used:      usable,
		At:        now,
	}); err != nil {
		// Not fatal to the turn. A reply that arrived is a reply worth showing
		// even if the log failed to take it; the cost of that is one call
		// missing from the month's sum, which errs toward under-counting and
		// is the direction a failure here should err in anyway, because a
		// failing database is about to take the next Allows call with it.
		slog.Error("recording what the coach said", "error", err)
	}

	if !usable {
		// The rejected text is logged, once, at the only moment anyone could
		// act on it. Without this the guard is a silent filter and there is no
		// way to tell "the model said something shaped wrong" from "the model
		// was never called".
		slog.Warn("the coach said something the wrong shape; using the fixed answer",
			"kind", t.Kind, "model", model, "said", text)
		return Reply{}, ErrUnavailable
	}

	return Reply{Text: guarded, Model: model, InTokens: in, OutTokens: out}, nil
}

// messages renders the turn.
//
// The order is fixed: system, then the window oldest first, then the state of
// now, then what was actually said. Now goes immediately before the question
// rather than in the system message on purpose — the system message is the
// part worth caching, and the clock changes every minute.
func (p *Provider) messages(t Turn) []chatMessage {
	msgs := make([]chatMessage, 0, 2*len(t.Recent)+3)
	msgs = append(msgs, chatMessage{Role: "system", Content: System(t.Now)})

	for _, e := range t.Recent {
		msgs = append(msgs, chatMessage{Role: "user", Content: e.Said})
		msgs = append(msgs, chatMessage{Role: "assistant", Content: e.Replied})
	}

	said := t.Said
	if line := Context(t.Now); line != "" {
		said = line + "\n\n" + said
	}
	if t.Subject != "" {
		// What is on screen, named as such. Without the framing the model
		// reads it as another thing the person said, and answers about the
		// wrong one.
		said = "On screen: " + t.Subject + "\n" + said
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: said})

	return msgs
}
