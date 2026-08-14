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

func Respond(w http.ResponseWriter, o squirrel.Outcome) {
	switch o {
	case squirrel.Stored:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "🐿️")
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

func NewCampfire(cfg squirrel.CampfireConfig) Transport {
	t := Transport{Name: CampfireName}

	t.Start = func(_ context.Context, sink Sink, mount Mount) (func(context.Context) error, error) {
		mount.Post(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
			if err != nil {
				slog.Error("campfire: reading body", "error", err)
				Respond(w, squirrel.Failed)
				return
			}
			Respond(w, sink.Accept(r.Context(), CaptureFrom(body, time.Now().UTC())))
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
