package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
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

// spooledReceipt says the spool write and its fsync completed, so the thought
// survives a crash. The ✅ that follows comes from the applier once the drain has
// reached Postgres; the gap between the two is the window this architecture is
// built around.
const spooledReceipt = "👀"

// The Campfire adapter. Every quirk below is Campfire's, not ours:
//
// It treats the HTTP response body as the bot's reply. A 200 carrying a
// Content-Type is posted into the room; a non-200 carrying one is uploaded as an
// attachment; no Content-Type at all is the only silence. A hard seven-second
// deadline follows, after which Campfire posts its own failure notice over the
// top.
//
// There is no signature, no shared secret and no timestamp. The caller's
// identity is the callback URL, guarded by NetworkPolicy; the clock is ours.

type campfirePayload struct {
	Type string `json:"type"`
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
	Action *struct {
		Value    string `json:"value"`
		Selected bool   `json:"selected"`
	} `json:"action"`
}

func derefOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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

	// An action is input like anything else: spooled, acknowledged, applied after the
	// drain. Its text is a stable encoding rather than raw JSON so the matcher has
	// one thing to recognise and CapturesSince can filter it out via ParseAction.
	//
	// The external id carries the receive instant because the payload has no event id
	// and no timestamp: without it, tapping a button off and on again collides with
	// the first tap and is dropped by InsertItem's conflict clause.
	if p.Type == "action" && p.Message != nil && p.Action != nil {
		id := fmt.Sprintf("action:%s:%s:%s:%t:%d",
			derefOr(identifier(p.Message.ID)), derefOr(c.SenderID),
			p.Action.Value, p.Action.Selected, receivedAt.UnixNano())
		c.ExternalID = squirrel.Ptr(id)
		c.Text = fmt.Sprintf("!action %s %s %t",
			derefOr(identifier(p.Message.ID)), p.Action.Value, p.Action.Selected)
	}
	return c
}

// accept converts a panic in somebody else's Sink into squirrel.Failed.
//
// Nothing may escape the handler: the server's own recover replies with a bare
// 500 and no Content-Type, which Campfire treats as silence, so the sender never
// learns their thought was dropped.
func accept(ctx context.Context, sink Sink, c squirrel.Capture) (o squirrel.Outcome) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("campfire: sink panicked", "panic", r)
			o = squirrel.Failed
		}
	}()
	return sink.Accept(ctx, c)
}

// Respond says nothing unless something went wrong.
//
// The Content-Type is what speaks, not the status: carrying one posts the body
// into the room, and omitting one posts nothing. So a stored or ignored capture
// sets no header and writes no bytes — Go sniffs a type the moment there are
// any — and only a failure says so out loud. The receipt for a stored capture is
// a boost on the message, fired separately.
//
// A non-200 carrying a Content-Type would be uploaded as an attachment, which is
// why the failure is still a 200.
func Respond(w http.ResponseWriter, o squirrel.Outcome) {
	switch o {
	case squirrel.Stored, squirrel.Ignored:
		w.WriteHeader(http.StatusOK)
	case squirrel.Failed:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "⚠️ couldn't save that — please resend")
	}
}

// asRichText prepares a body for Campfire, whose Message declares
// `has_rich_text :body` — so whatever arrives is treated as HTML whatever the
// content type says, and a newline collapses the way any whitespace does inside
// an HTML block. That is how a three-line digest arrived as one run-on sentence.
//
// Escaping happens first and is not optional: the digest carries captured text
// back verbatim, so a note containing "<b>" would turn words into markup.
//
// Campfire's quirk, so it lives in the transport. render.go stays plain text.
func asRichText(text string) string {
	return strings.ReplaceAll(html.EscapeString(text), "\n", "<br>")
}

// sendVia is outbound, used when the system initiates. Reusing room.path from a
// stored payload would need no credential, since that path embeds a bot key — and
// is rejected on purpose: outbound would then only reach rooms Squirrel had
// recently heard from, which works in testing and fails on a quiet Monday.
func sendVia(baseURL, botKey string) func(context.Context, string, string) error {
	base := strings.TrimRight(baseURL, "/")
	client := &http.Client{Timeout: 10 * time.Second}

	return func(ctx context.Context, conversationID, text string) error {
		url := fmt.Sprintf("%s/rooms/%s/%s/messages", base, conversationID, botKey)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(asRichText(text)))
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
	// omitempty: the fork's controller only touches the keys actually present
	// in the request (ActionController::Parameters#permit), so a PATCH that
	// omits "body" leaves the room's existing text alone rather than
	// overwriting it with an empty string. That is what lets closePrevious
	// send an update carrying only Actions.
	Body          string           `json:"body,omitempty"`
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

	// roundTrip builds one request for m — JSON when it carries actions, plain
	// text otherwise — sends it, and hands back the response with its status
	// unexamined. do is what turns that into a result; keeping the status
	// check out of here is what lets do retry a rejected JSON attempt without
	// roundTrip knowing anything about retries.
	roundTrip := func(ctx context.Context, method, dest string, m squirrel.Message, disabled bool) (res *http.Response, isJSON bool, err error) {
		var (
			body        io.Reader
			contentType = "text/plain; charset=utf-8"
		)
		if len(m.Actions) > 0 {
			encoded, err := json.Marshal(campfireMessage{
				Body:          asRichText(m.Text),
				SelectionMode: m.SelectionMode,
				Actions:       actionsFor(m, disabled),
			})
			if err != nil {
				return nil, false, fmt.Errorf("campfire: encoding message: %w", err)
			}
			body, contentType, isJSON = bytes.NewReader(encoded), "application/json", true
		} else {
			body = strings.NewReader(asRichText(m.Text))
		}

		req, err := http.NewRequestWithContext(ctx, method, dest, body)
		if err != nil {
			return nil, isJSON, fmt.Errorf("campfire: building request: %w", stripURL(err))
		}
		req.Header.Set("Content-Type", contentType)

		res, err = client.Do(req)
		if err != nil {
			return nil, isJSON, fmt.Errorf("campfire: request failed: %w", stripURL(err))
		}
		return res, isJSON, nil
	}

	// unforkedCampfire reports a server that refused the buttons envelope.
	//
	// An unforked Campfire takes the raw request body as the message text, so it
	// would post the JSON itself into the room; a 4xx to a JSON attempt is what
	// that looks like from here. A 5xx or a network error is Campfire being
	// unavailable rather than a shape it refused, so neither retries.
	//
	// POST only. Update always sends Text empty so the PATCH omits "body" and the
	// room's existing text survives — retrying that as plain text would send an
	// explicit empty body and wipe the message, which is the bug the omission
	// exists to prevent, hidden by the retry's own 2xx.
	unforkedCampfire := func(method string, isJSON bool, res *http.Response) bool {
		return method == http.MethodPost && isJSON &&
			res.StatusCode >= 400 && res.StatusCode < 500
	}

	// do sends m, and falls back to plain text against a Campfire that cannot
	// take buttons.
	do := func(ctx context.Context, method, dest string, m squirrel.Message, disabled bool) (*http.Response, error) {
		res, isJSON, err := roundTrip(ctx, method, dest, m, disabled)
		if err != nil {
			return nil, err
		}

		if unforkedCampfire(method, isJSON, res) {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			slog.Warn("campfire: message with actions was rejected, retrying as plain text",
				"status", res.StatusCode)

			res, _, err = roundTrip(ctx, method, dest, squirrel.Message{Text: m.Text}, disabled)
			if err != nil {
				return nil, err
			}
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

// stripURL keeps the bot key out of the logs.
//
// client.Do wraps a transport failure in a *url.Error whose Error() embeds the
// full request URL, and every outbound Campfire URL carries the bot key as a
// path segment: "Post \"http://.../rooms/7/<bot-key>/messages\": dial tcp ...".
// This path runs during an outage, which is exactly when logs get shipped and
// read.
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

// shapedLikeAPath reports a room path safe to interpolate: absolute, no
// scheme-relative "//", no ".." to climb with, and none of the URL-special
// characters the class excludes.
func shapedLikeAPath(roomPath string) bool {
	if !safeRoomPath.MatchString(roomPath) || strings.Contains(roomPath, "//") {
		return false
	}
	for _, segment := range strings.Split(roomPath, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

// stillAddresses reports that the composed URL goes where baseURL goes.
//
// A regex is easy to loosen later without noticing what it re-admits, so the
// composed URL is re-parsed and checked with a real parser. This is the layer
// that catches the bare "@": a roomPath of "@evil.com/rooms" against
// "http://campfire.internal" composes to
// "http://campfire.internal@evil.com/rooms/42/boosts", which parses with host
// evil.com and campfire.internal as userinfo — a different destination entirely.
func stillAddresses(baseURL, built string) bool {
	base, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	composed, err := url.Parse(built)
	if err != nil {
		return false
	}
	return composed.Scheme == base.Scheme && composed.Host == base.Host && composed.User == nil
}

// BoostURL builds the reaction endpoint out of the payload, which carries
// room.path as "/rooms/:id/:bot_key/messages" — so the bot key needs no
// configuration. That path is attacker-controlled input and is validated in two
// layers before it is interpolated.
//
// A rejected input returns ok == false, which must mean no boost, silently —
// never a dropped capture and never a request sent somewhere else.
func BoostURL(baseURL, roomPath, messageID string) (built string, ok bool) {
	if !shapedLikeAPath(roomPath) || !safeMessageID.MatchString(messageID) {
		return "", false
	}
	built = fmt.Sprintf("%s%s/%s/boosts", strings.TrimRight(baseURL, "/"), roomPath, messageID)
	if !stillAddresses(baseURL, built) {
		return "", false
	}
	return built, true
}

// boost reacts to a message, fired after the capture is durable and never
// blocking the response: Campfire is waiting on us with a seven-second deadline.
//
// Two retries, then give up. A missing receipt is cosmetic.
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

	// Neither room.path nor dest ever goes into a log field: both embed the bot
	// key. Errors from boost are stripped of it by stripURL.
	dest, ok := BoostURL(cfg.BaseURL, p.Room.Path, *capture.ExternalID)
	if !ok {
		slog.Warn("campfire: rejecting boost with an unsafe room path")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := boost(ctx, client, dest, spooledReceipt); err != nil {
			slog.Error("campfire: boost failed", "error", err)
		}
	}()
}

func NewCampfire(cfg squirrel.CampfireConfig) Transport {
	t := Transport{Name: CampfireName}
	boostClient := &http.Client{Timeout: 5 * time.Second}
	limiter := newRateLimiter(campfireRateBurst, campfireRateRefillPerSecond)

	t.Start = func(_ context.Context, sink Sink, mount Mount) (func(context.Context) error, error) {
		mount.Post(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
			if !limiter.allow() {
				slog.Warn("campfire: rate limited")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
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
