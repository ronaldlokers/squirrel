//go:build integration

package boot_test

import (
	"context"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/boot"
)

var retryInPattern = regexp.MustCompile(`retry_in=(\S+)`)

func TestMigrationRetryBacksOffRatherThanHammeringPostgres(t *testing.T) {
	var logs safeLog
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	s, err := boot.Boot(context.Background(), envFor(t, map[string]string{
		"POSTGRES_SERVER": "127.0.0.1", "POSTGRES_PORT": "1",
		"DRAIN_INTERVAL_MS": "20",
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, s.Stop(ctx))
	})

	var waits []time.Duration
	require.Eventually(t, func() bool {
		waits = retryWaits(t, logs.String())
		return len(waits) >= 4
	}, 3*time.Second, 10*time.Millisecond)

	require.Greater(t, waits[3], waits[0], "the fourth retry waited no longer than the first")
	for i := 1; i < len(waits); i++ {
		require.GreaterOrEqual(t, waits[i], waits[i-1], "retry %d waited less than retry %d before it", i, i-1)
	}
}

func retryWaits(t *testing.T, logged string) []time.Duration {
	t.Helper()
	var out []time.Duration
	for _, m := range retryInPattern.FindAllStringSubmatch(logged, -1) {
		d, err := time.ParseDuration(m[1])
		require.NoError(t, err)
		out = append(out, d)
	}
	return out
}
