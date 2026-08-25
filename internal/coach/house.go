package coach

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// The model in the house.
//
// Every capture typed into the box was being sent abroad to be read, which is
// the architecture this product states and does not follow: "rules narrow, and
// the model answers the few that survive" — the argument the interruption
// pre-filter and the splitter are both built on. Reading every thought with a
// hosted model is the opposite of that, and it shipped anyway on 25 August
// 2026.
//
// So the reading is split by what each thing is actually good at.
//
// The judgement — is this a thought or a question — is a small, high-volume,
// low-stakes classification that runs on everything typed. It happens in the
// house, on the cluster, on a model small enough to answer in the time a press
// takes. Being wrong costs a note in the pile, which is the failure this whole
// design already accepts.
//
// The answer — what to actually say to a question — is rare, and it is the one
// place quality is the whole point. That stays abroad.
//
// And underneath both is the rule. `LooksLikeAQuestion` needs no model, no
// network and no cluster, and it is what answers when the house is asleep. The
// three of them are the same ladder every other seam in this product has:
// rules, then something cheap, then something good, and never the expensive
// one for a job the cheap one does.

// houseTimeout is how long the model in the house may take.
//
// Two seconds. This runs while somebody is watching a press land, and the
// thing underneath it is a regex that answers instantly — so a house model
// that is thinking is a house model that has already lost to the rule it was
// meant to improve on. Generous for a one-word classification on a small model
// and mean for anything else, which is the right way round.
const houseTimeout = 2 * time.Second

// House is a model running on the cluster, or the zero value for none.
//
// An OpenAI-shaped endpoint, because that is what ollama and llama.cpp both
// serve and neither needs a key. The zero value is a supported configuration
// and the one the product shipped with: no house, and the rule answers.
type House struct {
	Client  *http.Client
	BaseURL string
	Model   string
}

// NewHouse builds one, or nil when there is no address.
//
// Nil rather than a struct that cannot work, so the caller's own nil check is
// the whole of the configuration — the same shape Photos and Push use, and for
// the same reason: a thing that draws a control it cannot honour is worse than
// one that was never built.
func NewHouse(baseURL, model string) *House {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(model) == "" {
		return nil
	}
	return &House{
		Client:  &http.Client{Timeout: houseTimeout},
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
	}
}

// housePreamble is what the small model is told.
//
// Shorter and blunter than anything sent abroad, and that is deliberate: a
// model this size follows one instruction well and four badly. It is asked for
// one word, so there is nothing to parse and nothing to guard beyond which
// word came back.
const housePreamble = `Answer with one word and nothing else.

If the text is a question the person is asking you, answer: QUESTION
Otherwise answer: THOUGHT

Anything they are telling you to remember is a THOUGHT, including
reminders, worries, numbers and half-finished sentences. When unsure,
answer THOUGHT.`

// AskedAQuestion says whether the words are a question, using the model in the
// house.
//
// The second return is whether the house answered at all. False means the
// caller falls back to the rule, which is the floor and needs nothing running.
//
// No budget and no accounting. This costs electricity in a cupboard rather
// than money abroad, and the budget exists to bound a bill.
func (h *House) AskedAQuestion(ctx context.Context, said string) (bool, bool) {
	if h == nil {
		return false, false
	}
	body, err := json.Marshal(map[string]any{
		"model": h.Model,
		"messages": []map[string]string{
			{"role": "system", "content": housePreamble},
			{"role": "user", "content": strings.TrimSpace(said)},
		},
		// One word is the whole answer, and a cap is what stops a small model
		// explaining itself for four hundred tokens on a Raspberry Pi.
		"max_tokens":  4,
		"temperature": 0,
		"stream":      false,
	})
	if err != nil {
		return false, false
	}

	ctx, cancel := context.WithTimeout(ctx, houseTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.BaseURL+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return false, false
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := h.Client.Do(req)
	if err != nil {
		// Asleep, unreachable, or thinking for longer than a press takes. The
		// rule answers, which is what happens when there is no house at all.
		slog.Debug("the house did not answer", "error", err)
		return false, false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slog.Debug("the house refused", "status", res.StatusCode)
		return false, false
	}

	var out chatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil || len(out.Choices) == 0 {
		return false, false
	}

	// One word, and it has to be one of the two. A small model that answered
	// with a sentence has not done what it was asked, and anything other than
	// the two words is treated as no answer rather than as a guess — the rule
	// underneath is better than a coin toss.
	switch strings.ToUpper(strings.TrimSpace(strings.Trim(out.Choices[0].Message.Content, ".\"'"))) {
	case "QUESTION":
		return true, true
	case "THOUGHT":
		return false, true
	}
	return false, false
}
