package coach

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// The wire format, which is the OpenAI chat completions shape.
//
// Chat completions rather than anything newer on purpose: it is the shape every
// gateway and every compatible provider implements, which is what makes
// COACH_BASE_URL a real escape hatch rather than a decorative one. Nothing in
// this product needs a feature that only the newer endpoint has.
//
// Hand-rolled against net/http, like the push sender and the Campfire client
// before it. A provider SDK would be a third dependency, a release cadence, and
// a surface far larger than the four fields actually used.

// maxOutput caps what the model may produce. The guard refuses anything over
// 400 characters anyway; capping here means an ignored prompt costs a fraction
// of a cent rather than a full completion that is then thrown away.
const maxOutput = 200

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	// MaxCompletionTokens rather than the older max_tokens: the older name is
	// rejected outright by current models rather than merely deprecated.
	MaxCompletionTokens int `json:"max_completion_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// completion is one call. It returns what was said and what it cost in tokens.
//
// Every failure is wrapped in ErrUnavailable, because that is the only
// distinction a caller makes: either there is something usable to show, or the
// deterministic answer is used. A taxonomy of failures here would be a
// taxonomy nobody branches on.
//
// The detail is kept in the wrapped message all the same, for the log. "No
// coach available" in a log line at three in the morning is not an answer to
// anything.
func (p *Provider) completion(ctx context.Context, model string, messages []chatMessage) (string, int, int, error) {
	body, err := json.Marshal(chatRequest{
		Model:               model,
		Messages:            messages,
		MaxCompletionTokens: maxOutput,
	})
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w: building the request: %w", ErrUnavailable, err)
	}

	url := strings.TrimSuffix(p.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w: building the request: %w", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	res, err := p.Client.Do(req)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer res.Body.Close()

	// Capped: a proxy returning an HTML error page must not be read into
	// memory in full to be discarded.
	raw, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w: reading the reply: %w", ErrUnavailable, err)
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// The status is included because a body that will not parse is nearly
		// always a proxy or an auth failure rather than the model, and the
		// status says which. The body itself is not logged: it may carry back
		// whatever was sent, and what was sent is the person's own words.
		return "", 0, 0, fmt.Errorf("%w: unreadable reply (status %d)", ErrUnavailable, res.StatusCode)
	}

	if res.StatusCode != http.StatusOK {
		detail := ""
		if parsed.Error != nil {
			detail = ": " + parsed.Error.Message
		}
		return "", 0, 0, fmt.Errorf("%w: status %d%s", ErrUnavailable, res.StatusCode, detail)
	}
	if len(parsed.Choices) == 0 {
		return "", 0, 0, fmt.Errorf("%w: no reply in the response", ErrUnavailable)
	}

	// Tokens are returned even on the paths that then fail the guard, because
	// a rejected reply was still paid for and the budget has to hear about it.
	return parsed.Choices[0].Message.Content,
		parsed.Usage.PromptTokens,
		parsed.Usage.CompletionTokens,
		nil
}
