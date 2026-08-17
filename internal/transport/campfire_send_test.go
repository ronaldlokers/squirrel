package transport_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/transport"
)

type received struct {
	path string
	body string
}

func stubCampfire(t *testing.T, status int) (string, *[]received) {
	t.Helper()
	got := &[]received{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*got = append(*got, received{path: r.URL.Path, body: string(body)})
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, got
}

func TestSendIsNilWithoutABotKey(t *testing.T) {
	require.Nil(t, transport.NewCampfire(config()).Send)
}

func TestSendPostsToTheBotMessagesEndpoint(t *testing.T) {
	base, got := stubCampfire(t, http.StatusCreated)

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"
	send := transport.NewCampfire(cfg).Send
	require.NotNil(t, send)

	require.NoError(t, send(context.Background(), "7", "time to vacuum"))
	require.Len(t, *got, 1)
	require.Equal(t, "/rooms/7/3-abc/messages", (*got)[0].path)
	require.Equal(t, "time to vacuum", (*got)[0].body)
}

func TestSendFailsOnA500(t *testing.T) {
	base, _ := stubCampfire(t, http.StatusInternalServerError)

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"

	err := transport.NewCampfire(cfg).Send(context.Background(), "7", "hello")
	require.ErrorContains(t, err, "500")
}

// response.ok in the TypeScript version treated 4xx as failure too. So must this.
func TestSendFailsOnA404(t *testing.T) {
	base, _ := stubCampfire(t, http.StatusNotFound)

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"

	err := transport.NewCampfire(cfg).Send(context.Background(), "7", "hello")
	require.ErrorContains(t, err, "404")
}

func TestSendToleratesTrailingSlashes(t *testing.T) {
	base, got := stubCampfire(t, http.StatusCreated)

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base+"//", "3-abc"

	require.NoError(t, transport.NewCampfire(cfg).Send(context.Background(), "7", "hello"))
	require.Equal(t, "/rooms/7/3-abc/messages", (*got)[0].path)
}

// client.Do wraps a transport failure in a *url.Error whose Error() method
// embeds the full request URL, and the Campfire URL carries the bot key as a
// path segment: "Post \"http://127.0.0.1:1/rooms/7/3-test/messages\": dial
// tcp ...". That string is exactly what would otherwise reach slog.Error in
// internal/boot/boot.go on every outbound failure — precisely during an
// outage, exactly when logs get shipped and read. "3-test" stands in for a
// real bot key; never a real credential in a fixture.
func TestSendFailureDoesNotLeakTheBotKey(t *testing.T) {
	cfg := config()
	cfg.BaseURL, cfg.BotKey = "http://127.0.0.1:1", "3-test"

	err := transport.NewCampfire(cfg).Send(context.Background(), "7", "hello")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "3-test")
}

func TestSendHonoursContextCancellation(t *testing.T) {
	base, _ := stubCampfire(t, http.StatusCreated)

	cfg := config()
	cfg.BaseURL, cfg.BotKey = base, "3-abc"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := transport.NewCampfire(cfg).Send(ctx, "7", "hello")
	require.ErrorIs(t, err, context.Canceled)
}
