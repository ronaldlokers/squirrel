package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
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
			return fmt.Errorf("campfire: building send request: %w", stripURL(err))
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")

		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("campfire: send failed: %w", stripURL(err))
		}
		defer res.Body.Close()
		io.Copy(io.Discard, res.Body)

		if res.StatusCode < 200 || res.StatusCode > 299 {
			return fmt.Errorf("campfire: send failed with %d", res.StatusCode)
		}
		return nil
	}
}

type campfireAction struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Emoji    string `json:"emoji,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type campfireMessage struct {
	Body          string           `json:"body"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Actions       []campfireAction `json:"actions,omitempty"`
}

func actionsFor(m squirrel.Message, disabled bool) []campfireAction {
	out := make([]campfireAction, 0, len(m.Actions))
	for _, a := range m.Actions {
		out = append(out, campfireAction{
			Label: a.Label, Value: a.Value, Emoji: a.Emoji, Disabled: disabled,
		})
	}
	return out
}

// messageIDFrom pulls the id out of the Location header of a create response.
// Campfire returns the message's own URL there; nothing else in the response
// names it.
func messageIDFrom(res *http.Response) string {
	loc := res.Header.Get("Location")
	if loc == "" {
		return ""
	}
	trimmed := strings.TrimRight(loc, "/")
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		return trimmed[i+1:]
	}
	return ""
}

// chatVia builds the outbound surface. A message with no actions is posted as
// plain text, byte for byte what phase 2 sent — only a message that needs
// buttons becomes JSON.
func chatVia(baseURL, botKey string) squirrel.Chat {
	base := strings.TrimRight(baseURL, "/")
	client := &http.Client{Timeout: 10 * time.Second}

	do := func(ctx context.Context, method, dest string, m squirrel.Message, disabled bool) (*http.Response, error) {
		var (
			body        io.Reader
			contentType = "text/plain; charset=utf-8"
		)
		if len(m.Actions) > 0 {
			encoded, err := json.Marshal(campfireMessage{
				Body:          m.Text,
				SelectionMode: m.SelectionMode,
				Actions:       actionsFor(m, disabled),
			})
			if err != nil {
				return nil, fmt.Errorf("campfire: encoding message: %w", err)
			}
			body, contentType = bytes.NewReader(encoded), "application/json"
		} else {
			body = strings.NewReader(m.Text)
		}

		req, err := http.NewRequestWithContext(ctx, method, dest, body)
		if err != nil {
			return nil, fmt.Errorf("campfire: building request: %w", stripURL(err))
		}
		req.Header.Set("Content-Type", contentType)

		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("campfire: request failed: %w", stripURL(err))
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			return nil, fmt.Errorf("campfire: request failed with %d", res.StatusCode)
		}
		return res, nil
	}

	return squirrel.Chat{
		Send: func(ctx context.Context, conversationID string, m squirrel.Message) (string, error) {
			dest := fmt.Sprintf("%s/rooms/%s/%s/messages", base, conversationID, botKey)
			res, err := do(ctx, http.MethodPost, dest, m, false)
			if err != nil {
				return "", err
			}
			defer res.Body.Close()
			io.Copy(io.Discard, res.Body)
			return messageIDFrom(res), nil
		},

		// Update is only ever used to close a surface, so it always disables.
		// A general-purpose edit would need a reason to exist first.
		Update: func(ctx context.Context, conversationID, messageID string, m squirrel.Message) error {
			dest := fmt.Sprintf("%s/rooms/%s/%s/messages/%s", base, conversationID, botKey, messageID)
			res, err := do(ctx, http.MethodPatch, dest, m, true)
			if err != nil {
				return err
			}
			io.Copy(io.Discard, res.Body)
			return res.Body.Close()
		},

		Boost: func(ctx context.Context, conversationID, messageID, content string) error {
			dest := fmt.Sprintf("%s/rooms/%s/%s/messages/%s/boosts", base, conversationID, botKey, messageID)
			return boost(ctx, client, dest, content)
		},
	}
}

// stripURL removes the request URL from a *url.Error before it is logged or
// otherwise surfaced. client.Do wraps a transport failure in a *url.Error
// whose Error() method embeds the full request URL, and every outbound
// Campfire URL — send and boost alike — carries the bot key as a path
// segment: "Post \"http://.../rooms/7/<bot-key>/messages\": dial tcp ...".
// The key is the most sensitive thing in this whole namespace, and this path
// is exercised precisely during an outage, exactly when logs get shipped and
// read. Only the operation and the underlying error survive; anything that
// is not a *url.Error passes through unchanged.
func stripURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}

// safeRoomPath and safeMessageID bound what a boost URL is built from. Both
// values ride in on the webhook payload, not from configuration, so they are
// attacker-controlled input rather than a trusted constant — even though in
// practice room.path comes from Campfire's own Rails route helper.
var (
	safeRoomPath  = regexp.MustCompile(`^/[A-Za-z0-9._~/-]+$`)
	safeMessageID = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

// BoostURL builds the reaction endpoint from what the payload already carries.
// room.path is "/rooms/:id/:bot_key/messages", so the bot key needs no
// configuration — it arrives with every message. Phase 1 rejected reusing
// room.path for *initiating* a conversation, because outbound would then only
// reach rooms we had recently heard from. Reacting to the message that just
// arrived is the opposite case: the payload is the context.
//
// room.path is untrusted, so it is validated in two layers rather than
// trusted and interpolated. First, the shape: the allowed character class
// excludes "@" (and every other URL-special character) outright, so no
// whitespace, no "//" (which would parse as a scheme-relative host), no ".."
// segment (traversal), and the path must start with "/". Second — because a
// regex is easy to loosen later without noticing what it re-admits — the
// composed URL is re-parsed with a real URL parser and checked to still
// address the configured scheme and host with no userinfo. That second check
// is the one that would catch the classic bare "@" SSRF even if the character
// class above stopped excluding it: a roomPath of "@evil.com/rooms" against
// baseURL "http://campfire.internal" composes to
// "http://campfire.internal@evil.com/rooms/42/boosts", which parses with host
// evil.com and userinfo campfire.internal — a different destination entirely.
//
// A rejected input returns ok == false. That must mean no boost, silently —
// exactly like a missing room.path already does — never a dropped capture
// and never a request sent somewhere else.
func BoostURL(baseURL, roomPath, messageID string) (built string, ok bool) {
	if !safeRoomPath.MatchString(roomPath) || strings.Contains(roomPath, "//") {
		return "", false
	}
	for _, segment := range strings.Split(roomPath, "/") {
		if segment == ".." {
			return "", false
		}
	}
	if !safeMessageID.MatchString(messageID) {
		return "", false
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", false
	}

	built = fmt.Sprintf("%s%s/%s/boosts", strings.TrimRight(baseURL, "/"), roomPath, messageID)
	composed, err := url.Parse(built)
	if err != nil {
		return "", false
	}
	if composed.Scheme != base.Scheme || composed.Host != base.Host || composed.User != nil {
		return "", false
	}
	return built, true
}

// boost reacts to a message. It is fired after the capture is durable and never
// blocks the response: Campfire is waiting on us with a seven-second deadline,
// and making it wait on a second call to Campfire is a shape to avoid.
//
// Two retries, then give up. A missing receipt is cosmetic — the capture is on
// disk either way, and the daily digest lists it regardless.
func boost(ctx context.Context, client *http.Client, dest, content string) error {
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, dest, strings.NewReader(content))
		if err != nil {
			return stripURL(err)
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")

		res, err := client.Do(req)
		if err != nil {
			lastErr = stripURL(err)
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

	dest, ok := BoostURL(cfg.BaseURL, p.Room.Path, *capture.ExternalID)
	if !ok {
		// room.path embeds the bot key, so it never goes into a log field.
		slog.Warn("campfire: rejecting boost with an unsafe room path")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := boost(ctx, client, dest, "🐿️"); err != nil {
			// dest carries the bot key, so it never goes into a log field —
			// err is already stripped of it by stripURL inside boost.
			slog.Error("campfire: boost failed", "error", err)
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
		t.Chat = chatVia(cfg.BaseURL, cfg.BotKey)
	}

	return t
}
