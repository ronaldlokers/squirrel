package transport_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/transport"
)

type sentRequest struct {
	method string
	path   string
	body   map[string]any
}

func chatStub(t *testing.T, location string) (string, *[]sentRequest) {
	t.Helper()
	var got []sentRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		json.Unmarshal(raw, &body)
		got = append(got, sentRequest{method: r.Method, path: r.URL.Path, body: body})
		if location != "" {
			w.Header().Set("Location", location)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &got
}

func TestSendPostsActionsAsJSON(t *testing.T) {
	base, got := chatStub(t, "https://chat.example.com/rooms/9/messages/451")

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"
	chat := transport.NewCampfire(cfg).Chat

	id, err := chat.Send(context.Background(), "9", squirrel.Message{
		Text:          "Due",
		SelectionMode: "multiple",
		Actions: []squirrel.Action{
			{Label: "bin day", Value: "done:1", Emoji: "✅"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "451", id, "the id is parsed out of the Location header")

	require.Len(t, *got, 1)
	req := (*got)[0]
	require.Equal(t, http.MethodPost, req.method)
	require.Equal(t, "Due", req.body["body"])
	require.Equal(t, "multiple", req.body["selection_mode"])

	actions := req.body["actions"].([]any)
	require.Len(t, actions, 1)
	first := actions[0].(map[string]any)
	require.Equal(t, "bin day", first["label"])
	require.Equal(t, "done:1", first["value"])
	require.Equal(t, "✅", first["emoji"])
}

// DefinedMessage and a due DigestMessage always carry a button, which means
// they are always sent as JSON — but the spec requires phase 3 to degrade to
// phase 2 behaviour against an unforked Campfire, whose bot endpoint treats
// the raw request body as the message text. Without a fallback, the room
// would receive a literal "{"body":"…","actions":[…]}" instead of the
// message. A stub that rejects JSON with a 4xx and accepts plain text is
// what an unforked instance looks like from here, so a successful Send
// against it — carrying the plain text, not the JSON envelope — is what
// makes the degrade true by construction.
func TestSendDowngradesToPlainTextWhenJSONIsRejected(t *testing.T) {
	var mu []sentRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu = append(mu, sentRequest{
			method: r.Method,
			path:   r.URL.Path,
			body:   map[string]any{"raw": string(raw), "contentType": r.Header.Get("Content-Type")},
		})
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	cfg := config()
	cfg.BaseURL, cfg.BotKey = srv.URL, "3-abc"
	chat := transport.NewCampfire(cfg).Chat

	_, err := chat.Send(context.Background(), "9", squirrel.Message{
		Text:          "Due\n 1. bin day — 8 days, usually 7",
		SelectionMode: "multiple",
		Actions:       []squirrel.Action{{Label: "bin day", Value: "done:1", Emoji: "✅"}},
	})
	require.NoError(t, err, "the retry must make the call succeed, not merely attempted")

	require.Len(t, mu, 2, "one JSON attempt, then one plain-text retry")
	require.True(t, strings.HasPrefix(mu[0].body["contentType"].(string), "application/json"))
	require.True(t, strings.HasPrefix(mu[1].body["contentType"].(string), "text/plain"),
		"the retry must fall back to plain text, not JSON again")
	require.Equal(t, "Due\n 1. bin day — 8 days, usually 7", mu[1].body["raw"],
		"the plain-text retry carries the message's own Text")
}

// A message with no actions stays a plain text post, exactly as phase 2 sent
// it. Sending JSON for everything would change the shape of every existing
// reply for no gain.
func TestSendWithoutActionsStaysPlainText(t *testing.T) {
	base, got := chatStub(t, "")

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"
	chat := transport.NewCampfire(cfg).Chat

	_, err := chat.Send(context.Background(), "9", squirrel.Message{Text: "Stopped vacuum."})
	require.NoError(t, err)
	require.Len(t, *got, 1)
	require.Empty(t, (*got)[0].body, "not JSON")
}

func TestUpdateDisablesActions(t *testing.T) {
	base, got := chatStub(t, "")

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"
	chat := transport.NewCampfire(cfg).Chat

	require.NoError(t, chat.Update(context.Background(), "9", "451", squirrel.Message{
		Text:          "Due",
		SelectionMode: "multiple",
		Actions:       []squirrel.Action{{Label: "bin day", Value: "done:1", Emoji: "✅"}},
	}))

	require.Len(t, *got, 1)
	req := (*got)[0]
	require.Equal(t, http.MethodPatch, req.method)
	require.Contains(t, req.path, "/messages/451")

	first := req.body["actions"].([]any)[0].(map[string]any)
	require.Equal(t, true, first["disabled"], "Update is only ever used to close a surface")
}

// The fork's controller only touches keys actually present in the request
// (ActionController::Parameters#permit), so omitting "body" is how an update
// that only closes buttons leaves the message's existing text untouched.
// Marshalling an explicit "" would instead wipe it.
func TestUpdateWithNoTextOmitsTheBody(t *testing.T) {
	base, got := chatStub(t, "")

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"
	chat := transport.NewCampfire(cfg).Chat

	require.NoError(t, chat.Update(context.Background(), "9", "451", squirrel.Message{
		Actions: []squirrel.Action{{Label: "bin day", Value: "done:1", Emoji: "✅"}},
	}))

	require.Len(t, *got, 1)
	_, hasBody := (*got)[0].body["body"]
	require.False(t, hasBody, "no body key at all, not an empty string")
}

// The key is a path segment, so any error carrying the URL carries the key.
func TestChatFailureDoesNotLeakTheBotKey(t *testing.T) {
	cfg := config()
	cfg.BaseURL, cfg.BotKey = "http://127.0.0.1:1", "3-test"
	chat := transport.NewCampfire(cfg).Chat

	_, err := chat.Send(context.Background(), "9", squirrel.Message{Text: "hi"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "3-test")
}

func TestChatIsAbsentWithoutABotKey(t *testing.T) {
	cfg := config()
	cfg.BaseURL, cfg.BotKey = "http://example.invalid", ""
	require.Nil(t, transport.NewCampfire(cfg).Chat.Send)
}
