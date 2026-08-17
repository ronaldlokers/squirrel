package squirrel

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrConfig wraps every configuration failure so callers can match on it
// without matching on message text.
var ErrConfig = errors.New("configuration")

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// CampfireConfig carries the transport's settings. BaseURL and BotKey are
// empty unless outbound is configured, which is how "this bot cannot start a
// conversation" is said honestly rather than discovered at send time.
type CampfireConfig struct {
	Path           string
	ConversationID string
	SenderID       string
	BaseURL        string
	BotKey         string
}

type Config struct {
	Port           int
	Transports     []string
	OwnerHandle    string
	SpoolDir       string
	DrainInterval  time.Duration
	Postgres       PostgresConfig
	Campfire       *CampfireConfig
	DigestAt       time.Duration
	DigestLocation *time.Location
	// PresenceSecret authenticates the arrival webhook. Empty means the route
	// is not mounted at all — the same way an absent bot key leaves Send nil
	// rather than half-working, MountPresence itself refuses to mount with an
	// empty secret rather than serve an effectively open endpoint.
	PresenceSecret string
	// PresencePath is where the arrival webhook is mounted.
	PresencePath string
	// PresenceDelay is how long an arrival waits before nudging — "you have
	// a coat on" — see PresenceOptions' own doc comment. Configurable rather
	// than a boot.go constant because production and the integration suite
	// genuinely want different values here: a couple of minutes is the
	// point for a real arrival, but that would blow any test budget built to
	// wait one out over a real socket.
	PresenceDelay time.Duration
}

var knownTransports = map[string]bool{"campfire": true}

func required(env map[string]string, name string) (string, error) {
	if v := env[name]; v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%w: %s is required", ErrConfig, name)
}

func optional(env map[string]string, name, fallback string) string {
	if v := env[name]; v != "" {
		return v
	}
	return fallback
}

func number(env map[string]string, name string, fallback int) (int, error) {
	v := env[name]
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%w: %s must be a non-negative integer, got %q", ErrConfig, name, v)
	}
	return n, nil
}

// clockTime parses "08:00" into a duration since local midnight.
func clockTime(env map[string]string, name string, fallback time.Duration) (time.Duration, error) {
	v := env[name]
	if v == "" {
		return fallback, nil
	}
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be HH:MM, got %q", ErrConfig, name, v)
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
}

// duration parses a Go duration string like "2m" or "500ms".
func duration(env map[string]string, name string, fallback time.Duration) (time.Duration, error) {
	v := env[name]
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be a duration like \"2m\", got %q", ErrConfig, name, v)
	}
	return d, nil
}

func transportsFrom(env map[string]string) ([]string, error) {
	raw, ok := env["TRANSPORTS"]
	if !ok {
		raw = "campfire"
	}

	names := []string{}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !knownTransports[name] {
			return nil, fmt.Errorf("%w: TRANSPORTS names unknown transport %q", ErrConfig, name)
		}
		names = append(names, name)
	}
	return names, nil
}

func campfireFrom(env map[string]string) (*CampfireConfig, error) {
	conversationID, err := required(env, "CAMPFIRE_CONVERSATION_ID")
	if err != nil {
		return nil, err
	}
	senderID, err := required(env, "CAMPFIRE_SENDER_ID")
	if err != nil {
		return nil, err
	}

	baseURL, botKey := env["CAMPFIRE_BASE_URL"], env["CAMPFIRE_BOT_KEY"]
	if botKey != "" && baseURL == "" {
		return nil, fmt.Errorf("%w: CAMPFIRE_BOT_KEY requires CAMPFIRE_BASE_URL", ErrConfig)
	}

	return &CampfireConfig{
		Path:           optional(env, "CAMPFIRE_PATH", "/transports/campfire"),
		ConversationID: conversationID,
		SenderID:       senderID,
		BaseURL:        baseURL,
		BotKey:         botKey,
	}, nil
}

func LoadConfig(env map[string]string) (Config, error) {
	transports, err := transportsFrom(env)
	if err != nil {
		return Config{}, err
	}

	port, err := number(env, "PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	drainMS, err := number(env, "DRAIN_INTERVAL_MS", 1000)
	if err != nil {
		return Config{}, err
	}
	if drainMS == 0 {
		// Interval == 0 never grows under Drain.Run's doubling backoff, so a
		// deferred pass would hammer Postgres and the spool directory as fast
		// as the CPU allows rather than backing off.
		return Config{}, fmt.Errorf("%w: DRAIN_INTERVAL_MS must be greater than zero, got %q", ErrConfig, env["DRAIN_INTERVAL_MS"])
	}
	pgPort, err := number(env, "POSTGRES_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	pgHost, err := required(env, "POSTGRES_SERVER")
	if err != nil {
		return Config{}, err
	}
	pgDatabase, err := required(env, "POSTGRES_DB")
	if err != nil {
		return Config{}, err
	}
	pgUser, err := required(env, "POSTGRES_USER")
	if err != nil {
		return Config{}, err
	}
	pgPassword, err := required(env, "POSTGRES_PASSWORD")
	if err != nil {
		return Config{}, err
	}

	digestAt, err := clockTime(env, "DIGEST_AT", 8*time.Hour)
	if err != nil {
		return Config{}, err
	}
	location, err := time.LoadLocation(optional(env, "DIGEST_TZ", "Europe/Amsterdam"))
	if err != nil {
		return Config{}, fmt.Errorf("%w: DIGEST_TZ: %v", ErrConfig, err)
	}
	presenceDelay, err := duration(env, "PRESENCE_DELAY", 2*time.Minute)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Port:          port,
		Transports:    transports,
		OwnerHandle:   optional(env, "OWNER_HANDLE", "owner"),
		SpoolDir:      optional(env, "SPOOL_DIR", "/var/spool/squirrel"),
		DrainInterval: time.Duration(drainMS) * time.Millisecond,
		Postgres: PostgresConfig{
			Host: pgHost, Port: pgPort, Database: pgDatabase,
			User: pgUser, Password: pgPassword,
		},
		DigestAt:       digestAt,
		DigestLocation: location,
		PresenceSecret: env["PRESENCE_SECRET"],
		PresencePath:   optional(env, "PRESENCE_PATH", "/hooks/home"),
		PresenceDelay:  presenceDelay,
	}

	if slicesContains(transports, "campfire") {
		if config.Campfire, err = campfireFrom(env); err != nil {
			return Config{}, err
		}
	}
	return config, nil
}

func slicesContains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
