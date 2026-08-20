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

// CoachConfig is the model layer's settings, gathered so that swapping
// provider is a configuration change rather than an edit spread across the
// codebase.
//
// BaseURL exists even though only one provider is used today. It is what makes
// "gateway-shaped internally" true rather than aspirational: pointing this at a
// gateway is a deployment change, not a code change.
type CoachConfig struct {
	APIKey  string
	BaseURL string
	// Fast answers routine turns; Deep answers the ones where judgement
	// matters. Two rather than one because the difference in price between
	// them is tenfold and the difference in need is real.
	Fast string
	Deep string
	// BudgetMicros is the monthly ceiling in micro-euros. Zero means no
	// ceiling in this process; the provider's own spend limit still applies and
	// is the one that guards against a stolen key.
	BudgetMicros int64
}

// Enabled reports whether there is enough here to build a coach at all.
func (c CoachConfig) Enabled() bool {
	return c.APIKey != "" && c.Fast != "" && c.Deep != ""
}

type Config struct {
	Port          int
	Transports    []string
	OwnerHandle   string
	SpoolDir      string
	DrainInterval time.Duration
	Postgres      PostgresConfig
	Campfire      *CampfireConfig
	// EveningAt is the time since local midnight the evening message and its
	// once-a-day nudge fallback fire at — see schedule.go's once(). Named for
	// what it now is: EVENING_AT is load-bearing twice, as both the clock
	// trigger and the evening capture slot, and the quiet-day merge in once()
	// exists because they are the same instant. It used to default to 08:00
	// under the name DIGEST_AT, left over from the phase 2/3 morning digest
	// this phase replaced with an evening message — deployed under that
	// default, the evening message fired at breakfast.
	EveningAt      time.Duration
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
	// WebIdentity is the value Authentik puts in WebIdentityHeader for the one
	// person who may read this. Empty means the screen is not mounted at all —
	// the same refusal PresenceSecret makes, and for a stronger reason: the
	// pile is every thought you have ever had at this bot, and a screen that
	// mounted without knowing who is allowed to read it would be open.
	WebIdentity string
	// WebIdentityHeader is the header Traefik's forward-auth middleware fills
	// from the Authentik outpost. Squirrel writes no authentication code and
	// holds no session; it compares one header to one configured value.
	WebIdentityHeader string
	// PhotoDir is where photographs are kept, or empty. Empty is a supported
	// state and the default: with nowhere to put them the screen never offers
	// a camera, exactly as it never offers to subscribe without a push key.
	PhotoDir string
	// Coach is what the model layer needs, or empty. Empty is the default and a
	// supported state: with no key the coach is never built, and the picker and
	// the ladder answer instead — the deterministic floor they were kept as.
	Coach CoachConfig
	// Push is the VAPID identity, or empty. Empty is a supported state and not
	// a degraded one: the leave-by message still reaches the room, and the
	// screen simply never offers to subscribe. The private key comes from the
	// secret store and must never appear in the repository.
	Push PushConfig
	// WebURL is where the screen can be reached from outside, so chat can
	// point at it. Empty means chat says nothing about the screen, which is
	// the honest answer when nobody has said where it is — a link built from a
	// guess is a link that 404s.
	WebURL string
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

	// 19:00, not 08:00: EVENING_AT is both the clock trigger and the evening
	// capture slot (see the Config.EveningAt doc comment), and the two must
	// agree for the quiet-day merge in schedule.go's once() to mean anything.
	eveningAt, err := clockTime(env, "EVENING_AT", 19*time.Hour)
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
	// Whole euros. A ceiling nobody can state out loud is a ceiling nobody
	// checks, and "ten euros a month" is the sentence this setting exists to
	// hold. Zero disables the in-process ceiling; the provider's own spend
	// limit is unaffected either way.
	budget, err := number(env, "COACH_BUDGET_EUR", 10)
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
		EveningAt:      eveningAt,
		DigestLocation: location,
		PhotoDir:       env["PHOTO_DIR"],
		PresenceSecret: env["PRESENCE_SECRET"],
		PresencePath:   optional(env, "PRESENCE_PATH", "/hooks/home"),
		PresenceDelay:  presenceDelay,

		Coach: CoachConfig{
			APIKey:  env["OPENAI_API_KEY"],
			BaseURL: optional(env, "COACH_BASE_URL", "https://api.openai.com/v1"),
			// Confirmed against GET /v1/models on 20 August 2026 rather than
			// copied from a pricing page.
			Fast:         optional(env, "COACH_MODEL_FAST", "gpt-5.6-luna"),
			Deep:         optional(env, "COACH_MODEL_DEEP", "gpt-5.6-terra"),
			BudgetMicros: int64(budget) * 1_000_000,
		},

		Push: PushConfig{
			PublicKey:  env["VAPID_PUBLIC_KEY"],
			PrivateKey: env["VAPID_PRIVATE_KEY"],
			Contact:    env["PUSH_CONTACT"],
		},

		WebIdentity:       env["WEB_IDENTITY"],
		WebIdentityHeader: optional(env, "WEB_IDENTITY_HEADER", "X-Authentik-Username"),
		WebURL:            strings.TrimRight(env["WEB_URL"], "/"),
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
