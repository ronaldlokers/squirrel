//go:build integration

package boot_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/boot"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

const payload = `{
  "user":    {"id": 1, "name": "Ronald"},
  "room":    {"id": 7, "name": "Squirrel", "path": "/rooms/7/3-abc/messages"},
  "message": {"id": 42, "body": {"plain": "buy milk"}, "path": "/rooms/7/@42"}
}`

func envFor(t *testing.T, overrides map[string]string) map[string]string {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, raw, "TEST_DATABASE_URL is required — see docs/testing.md")

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	password, _ := parsed.User.Password()

	// campfireStub also hands back a requests func. Nothing here needs it —
	// a test that must inspect what was actually posted builds its own stub
	// via campfireStub(t) and overrides CAMPFIRE_BASE_URL with its URL.
	stubURL, _ := campfireStub(t)

	env := map[string]string{
		"PORT":                     "0",
		"SPOOL_DIR":                t.TempDir(),
		"DRAIN_INTERVAL_MS":        "10",
		"OWNER_HANDLE":             "ronald",
		"CAMPFIRE_CONVERSATION_ID": "7",
		"CAMPFIRE_SENDER_ID":       "1",
		"CAMPFIRE_BASE_URL":        stubURL,
		"CAMPFIRE_BOT_KEY":         "3-test",
		"POSTGRES_SERVER":          parsed.Hostname(),
		"POSTGRES_PORT":            parsed.Port(),
		"POSTGRES_DB":              strings.TrimPrefix(parsed.Path, "/"),
		"POSTGRES_USER":            parsed.User.Username(),
		"POSTGRES_PASSWORD":        password,
	}
	for k, v := range overrides {
		env[k] = v
	}
	return env
}

func boots(t *testing.T, env map[string]string) *boot.Squirrel {
	t.Helper()
	s, err := boot.Boot(context.Background(), env)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, s.Stop(ctx))
	})
	return s
}

func post(t *testing.T, s *boot.Squirrel, body string) *http.Response {
	t.Helper()
	res, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/transports/campfire", s.Port()),
		"application/json", strings.NewReader(body))
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestBootCapturesEndToEnd(t *testing.T) {
	store := withStore(t)
	s := boots(t, envFor(t, nil))

	res := post(t, s, payload)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	// Since the boost, the receipt is a reaction on the message rather than a
	// reply, so the response posts nothing.
	require.Empty(t, res.Header.Get("Content-Type"))
	require.Empty(t, string(body))

	require.Eventually(t, func() bool {
		var n int
		if err := store.Pool().QueryRow(context.Background(),
			`select count(*) from items where raw_text = 'buy milk'`).Scan(&n); err != nil {
			return false
		}
		return n == 1
	}, 5*time.Second, 20*time.Millisecond)

	var personID *int64
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select person_id from items limit 1`).Scan(&personID))
	require.NotNil(t, personID)
}

func TestBootSaysNothingToAnotherRoom(t *testing.T) {
	withStore(t)
	s := boots(t, envFor(t, nil))

	elsewhere := strings.Replace(payload, `"id": 7`, `"id": 8`, 1)
	res := post(t, s, elsewhere)

	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Empty(t, res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Empty(t, body)
}

func TestBootIsHealthy(t *testing.T) {
	withStore(t)
	s := boots(t, envFor(t, nil))

	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", s.Port()))
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
}

// The whole point of spooling. An unreachable database at boot must not stop a
// capture being accepted, because Campfire will not retry it.
func TestBootServesWithTheDatabaseUnreachable(t *testing.T) {
	withStore(t)
	s := boots(t, envFor(t, map[string]string{
		"POSTGRES_SERVER": "127.0.0.1", "POSTGRES_PORT": "1",
	}))

	res := post(t, s, payload)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	// Since the boost, the receipt is a reaction on the message rather than a
	// reply, so the response posts nothing.
	require.Empty(t, res.Header.Get("Content-Type"))
	require.Empty(t, string(body))

	health, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", s.Port()))
	require.NoError(t, err)
	defer health.Body.Close()
	healthBody, err := io.ReadAll(health.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(healthBody))
}

func TestBootRejectsBadConfiguration(t *testing.T) {
	_, err := boot.Boot(context.Background(), envFor(t, map[string]string{"TRANSPORTS": "campfre"}))
	require.ErrorIs(t, err, squirrel.ErrConfig)
}
