package squirrel_test

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func minimalEnv(overrides map[string]string) map[string]string {
	env := map[string]string{
		"CAMPFIRE_CONVERSATION_ID": "7",
		"CAMPFIRE_SENDER_ID":       "1",
		"POSTGRES_SERVER":          "db.example",
		"POSTGRES_DB":              "squirrel",
		"POSTGRES_USER":            "squirrel",
		"POSTGRES_PASSWORD":        "hunter2",
	}
	maps.Copy(env, overrides)
	return env
}

func TestLoadConfigDefaults(t *testing.T) {
	got, err := squirrel.LoadConfig(minimalEnv(nil))
	require.NoError(t, err)
	require.Equal(t, 8080, got.Port)
	require.Equal(t, []string{"campfire"}, got.Transports)
	require.Equal(t, "owner", got.OwnerHandle)
	require.Equal(t, "/var/spool/squirrel", got.SpoolDir)
	require.Equal(t, time.Second, got.DrainInterval)
	require.Equal(t, 5432, got.Postgres.Port)
	require.Equal(t, "/transports/campfire", got.Campfire.Path)
}

func TestLoadConfigLeavesSendUnconfigured(t *testing.T) {
	got, err := squirrel.LoadConfig(minimalEnv(nil))
	require.NoError(t, err)
	require.Empty(t, got.Campfire.BotKey)
}

func TestLoadConfigCarriesBotKeyWithBaseURL(t *testing.T) {
	got, err := squirrel.LoadConfig(minimalEnv(map[string]string{
		"CAMPFIRE_BASE_URL": "https://campfire.example",
		"CAMPFIRE_BOT_KEY":  "3-abc",
	}))
	require.NoError(t, err)
	require.Equal(t, "https://campfire.example", got.Campfire.BaseURL)
	require.Equal(t, "3-abc", got.Campfire.BotKey)
}

// Half-configured outbound would exist and fail at the moment it is needed.
// Absent outbound is at least honest.
func TestLoadConfigRejectsBotKeyWithoutBaseURL(t *testing.T) {
	_, err := squirrel.LoadConfig(minimalEnv(map[string]string{"CAMPFIRE_BOT_KEY": "3-abc"}))
	require.ErrorIs(t, err, squirrel.ErrConfig)
	require.ErrorContains(t, err, "CAMPFIRE_BASE_URL")
}

func TestLoadConfigNamesTheMissingVariable(t *testing.T) {
	env := minimalEnv(nil)
	delete(env, "POSTGRES_PASSWORD")
	_, err := squirrel.LoadConfig(env)
	require.ErrorContains(t, err, "POSTGRES_PASSWORD")
}

func TestLoadConfigRequiresCampfireWhenEnabled(t *testing.T) {
	env := minimalEnv(nil)
	delete(env, "CAMPFIRE_CONVERSATION_ID")
	_, err := squirrel.LoadConfig(env)
	require.ErrorContains(t, err, "CAMPFIRE_CONVERSATION_ID")
}

func TestLoadConfigAllowsNoTransports(t *testing.T) {
	env := minimalEnv(map[string]string{"TRANSPORTS": ""})
	delete(env, "CAMPFIRE_CONVERSATION_ID")
	delete(env, "CAMPFIRE_SENDER_ID")

	got, err := squirrel.LoadConfig(env)
	require.NoError(t, err)
	require.Empty(t, got.Transports)
	require.Nil(t, got.Campfire)
}

// The TypeScript build shipped without this and booted happily with zero
// transports, answering every webhook with an attachment-producing 404.
func TestLoadConfigRejectsUnknownTransport(t *testing.T) {
	_, err := squirrel.LoadConfig(minimalEnv(map[string]string{"TRANSPORTS": "campfre"}))
	require.ErrorIs(t, err, squirrel.ErrConfig)
	require.ErrorContains(t, err, "campfre")
}

func TestLoadConfigRejectsNonNumericPort(t *testing.T) {
	_, err := squirrel.LoadConfig(minimalEnv(map[string]string{"PORT": "eight"}))
	require.ErrorContains(t, err, "PORT")
}

func TestLoadConfigRejectsNegativeInterval(t *testing.T) {
	_, err := squirrel.LoadConfig(minimalEnv(map[string]string{"DRAIN_INTERVAL_MS": "-1"}))
	require.ErrorContains(t, err, "DRAIN_INTERVAL_MS")
}

// Interval == 0 never grows under Drain.Run's doubling backoff (0 * 2 stays
// 0), so a deferred pass would hammer Postgres and the spool directory as
// fast as the CPU allows instead of backing off.
func TestLoadConfigRejectsZeroInterval(t *testing.T) {
	_, err := squirrel.LoadConfig(minimalEnv(map[string]string{"DRAIN_INTERVAL_MS": "0"}))
	require.ErrorIs(t, err, squirrel.ErrConfig)
	require.ErrorContains(t, err, "DRAIN_INTERVAL_MS")
}

func TestLoadConfigEveningDefaults(t *testing.T) {
	got, err := squirrel.LoadConfig(minimalEnv(nil))
	require.NoError(t, err)
	require.Equal(t, 19*time.Hour, got.EveningAt,
		"19:00 is load-bearing twice — the clock trigger and the evening capture slot")
	require.Equal(t, "Europe/Amsterdam", got.DigestLocation.String())
}

func TestLoadConfigRejectsABadEveningTime(t *testing.T) {
	_, err := squirrel.LoadConfig(minimalEnv(map[string]string{"EVENING_AT": "8am"}))
	require.ErrorContains(t, err, "EVENING_AT")
}

func TestWebDefaults(t *testing.T) {
	cfg, err := squirrel.LoadConfig(minimalEnv(nil))
	require.NoError(t, err)
	require.Equal(t, "X-Authentik-Username", cfg.WebIdentityHeader)
	require.Empty(t, cfg.WebIdentity, "no identity means the screen is not mounted")
}

func TestWebIdentityIsTakenFromTheEnvironment(t *testing.T) {
	cfg, err := squirrel.LoadConfig(minimalEnv(map[string]string{
		"WEB_IDENTITY":        "ronald",
		"WEB_IDENTITY_HEADER": "X-Forwarded-User",
	}))
	require.NoError(t, err)
	require.Equal(t, "ronald", cfg.WebIdentity)
	require.Equal(t, "X-Forwarded-User", cfg.WebIdentityHeader)
}
