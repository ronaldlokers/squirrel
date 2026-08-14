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
	Port          int
	Transports    []string
	OwnerHandle   string
	SpoolDir      string
	DrainInterval time.Duration
	Postgres      PostgresConfig
	Campfire      *CampfireConfig
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
	if botKey == "" {
		baseURL = ""
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
