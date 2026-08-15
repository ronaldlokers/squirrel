package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

const CampfireName = "campfire"

// The Campfire adapter, and every quirk below stays inside this file.
//
// Campfire treats the HTTP response body as the bot's reply. A 200 with a
// Content-Type is posted into the room; a non-200 that still carries one is
// uploaded as an attachment; no Content-Type at all is the only silence. There
// is a hard seven-second deadline after which Campfire posts its own failure
// notice over the top. None of that is ours to choose.
//
// There is also no signature, no shared secret and no timestamp. The caller's
// identity is the callback URL, guarded by NetworkPolicy; the clock is ours.
type campfirePayload struct {
	User *struct {
		ID *json.Number `json:"id"`
	} `json:"user"`
	Room *struct {
		ID *json.Number `json:"id"`
	} `json:"room"`
	Message *struct {
		ID   *json.Number `json:"id"`
		Body *struct {
			Plain string `json:"plain"`
		} `json:"body"`
	} `json:"message"`
}

func identifier(n *json.Number) *string {
	if n == nil {
		return nil
	}
	return squirrel.Ptr(n.String())
}

// CaptureFrom fails open. An envelope we cannot read still becomes a capture
// with nil ids, because the alternative — dropping it — is how a payload
// change upstream silently empties the inbox for a week.
func CaptureFrom(body []byte, receivedAt time.Time) squirrel.Capture {
	unreadable := func() squirrel.Capture {
		wrapped, _ := json.Marshal(map[string]string{"unparseable": string(body)})
		return squirrel.Capture{
			Transport:  CampfireName,
			Text:       string(body),
			ReceivedAt: receivedAt,
			Payload:    wrapped,
		}
	}

	// A body of `null` unmarshals without error, leaving every field nil — the
	// TypeScript version crashed on exactly this case. A body of `[]` fails to
	// unmarshal into a struct and takes the unreadable path. Both end up as a
	// capture with nil ids, which is the fail-open behaviour required.
	var p campfirePayload
	if err := json.Unmarshal(body, &p); err != nil {
		return unreadable()
	}

	c := squirrel.Capture{
		Transport:  CampfireName,
		ReceivedAt: receivedAt,
		Payload:    append(json.RawMessage(nil), body...),
	}
	if p.Message != nil {
		c.ExternalID = identifier(p.Message.ID)
		if p.Message.Body != nil {
			c.Text = p.Message.Body.Plain
		}
	}
	if p.Room != nil {
		c.ConversationID = identifier(p.Room.ID)
	}
	if p.User != nil {
		c.SenderID = identifier(p.User.ID)
	}
	return c
}

// accept calls the sink and converts a panic into squirrel.Failed. Nothing
// may escape the handler: Sink is an interface supplied by the caller of
// NewCampfire, not written by this package, and a panic in someone else's
// implementation must still answer Campfire rather than unwind into the
// server's own recover — which replies with a bare 500 and no Content-Type,
// the one response Campfire treats as silence, so the sender never learns
// their thought was dropped.
func accept(ctx context.Context, sink Sink, c squirrel.Capture) (o squirrel.Outcome) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("campfire: sink panicked", "panic", r)
			o = squirrel.Failed
		}
	}()
	return sink.Accept(ctx, c)
}

func Respond(w http.ResponseWriter, o squirrel.Outcome) {
	switch o {
	case squirrel.Stored:
		// The receipt is a boost on the message itself, fired separately. A 200
		// with no Content-Type is Campfire's "post nothing", same as Ignored.
		w.WriteHeader(http.StatusOK)
	case squirrel.Ignored:
		// No body and no Content-Type: the one path that says nothing at all.
		// Never call Write here — Go sniffs a type when there are bytes.
		w.WriteHeader(http.StatusOK)
	case squirrel.Failed:
		// Still a 200. A non-200 carrying a Content-Type becomes an attachment.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "⚠️ couldn't save that — please resend")
	}
}

// sendVia is outbound, used when the system initiates rather than answers.
//
// Reusing room.path from a stored payload would need no credential at all,
// since that path already embeds a bot key. It is rejected on purpose:
// outbound would then only reach rooms Squirrel had recently heard from, and a
// morning nudge would depend on the capture history. That works in testing and
// fails on a quiet Monday.
func sendVia(baseURL, botKey string) func(context.Context, string, string) error {
	base := strings.TrimRight(baseURL, "/")
	client := &http.Client{Timeout: 10 * time.Second}

	return func(ctx context.Context, conversationID, text string) error {
		url := fmt.Sprintf("%s/rooms/%s/%s/messages", base, conversationID, botKey)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(text))
		if err != nil {
			return fmt.Errorf("campfire: building send request: %w", err)
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")

		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("campfire: send failed: %w", err)
		}
		defer res.Body.Close()
		io.Copy(io.Discard, res.Body)

		if res.StatusCode < 200 || res.StatusCode > 299 {
			return fmt.Errorf("campfire: send failed with %d", res.StatusCode)
		}
		return nil
	}
}

// BoostURL builds the reaction endpoint from what the payload already carries.
// room.path is "/rooms/:id/:bot_key/messages", so the bot key needs no
// configuration — it arrives with every message. Phase 1 rejected reusing
// room.path for *initiating* a conversation, because outbound would then only
// reach rooms we had recently heard from. Reacting to the message that just
// arrived is the opposite case: the payload is the context.
func BoostURL(baseURL, roomPath, messageID string) string {
	return fmt.Sprintf("%s%s/%s/boosts",
		strings.TrimRight(baseURL, "/"), roomPath, messageID)
}

// boost reacts to a message. It is fired after the capture is durable and never
// blocks the response: Campfire is waiting on us with a seven-second deadline,
// and making it wait on a second call to Campfire is a shape to avoid.
//
// Two retries, then give up. A missing receipt is cosmetic — the capture is on
// disk either way, and the daily digest lists it regardless.
func boost(ctx context.Context, client *http.Client, url, content string) error {
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(content))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")

		res, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()

		if res.StatusCode >= 200 && res.StatusCode <= 299 {
			return nil
		}
		lastErr = fmt.Errorf("boost failed with %d", res.StatusCode)
	}
	return lastErr
}

// fireBoost reacts in the background. Everything it needs is in the payload, so
// a missing room path or message id simply means no receipt — never a dropped
// capture.
func fireBoost(cfg squirrel.CampfireConfig, client *http.Client, body []byte, capture squirrel.Capture) {
	if cfg.BaseURL == "" || capture.ExternalID == nil {
		return
	}
	var p struct {
		Room *struct {
			Path string `json:"path"`
		} `json:"room"`
	}
	if err := json.Unmarshal(body, &p); err != nil || p.Room == nil || p.Room.Path == "" {
		return
	}

	url := BoostURL(cfg.BaseURL, p.Room.Path, *capture.ExternalID)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := boost(ctx, client, url, "🐿️"); err != nil {
			slog.Error("campfire: boost failed", "error", err, "url", url)
		}
	}()
}

func NewCampfire(cfg squirrel.CampfireConfig) Transport {
	t := Transport{Name: CampfireName}
	boostClient := &http.Client{Timeout: 5 * time.Second}

	t.Start = func(_ context.Context, sink Sink, mount Mount) (func(context.Context) error, error) {
		mount.Post(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
			if err != nil {
				slog.Error("campfire: reading body", "error", err)
				Respond(w, squirrel.Failed)
				return
			}

			capture := CaptureFrom(body, time.Now().UTC())
			outcome := accept(r.Context(), sink, capture)
			Respond(w, outcome)

			if outcome == squirrel.Stored {
				fireBoost(cfg, boostClient, body, capture)
			}
		})
		return func(context.Context) error { return nil }, nil
	}

	// Nil unless a bot key is configured. Half-working outbound would fail at
	// exactly the moment it is needed; absent outbound is at least honest.
	if cfg.BaseURL != "" && cfg.BotKey != "" {
		t.Send = sendVia(cfg.BaseURL, cfg.BotKey)
	}

	return t
}
