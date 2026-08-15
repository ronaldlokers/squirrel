package squirrel_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestURLForIPv4Host(t *testing.T) {
	got := squirrel.URLFor(squirrel.PostgresConfig{
		Host:     "host",
		Port:     5432,
		Database: "db",
		User:     "user",
		Password: "pass",
	})
	require.Equal(t, "postgres://user:pass@host:5432/db", got)
}

func TestURLForBracketsALiteralIPv6Host(t *testing.T) {
	got := squirrel.URLFor(squirrel.PostgresConfig{
		Host:     "::1",
		Port:     5432,
		Database: "db",
		User:     "user",
		Password: "pass",
	})

	require.Contains(t, got, "[::1]:5432")

	parsed, err := url.Parse(got)
	require.NoError(t, err)
	require.Equal(t, "::1", parsed.Hostname())
}

func TestURLForIPv4HostParsesCleanlyAndKeepsHostname(t *testing.T) {
	got := squirrel.URLFor(squirrel.PostgresConfig{
		Host:     "db.example.com",
		Port:     5432,
		Database: "db",
		User:     "user",
		Password: "pass",
	})

	parsed, err := url.Parse(got)
	require.NoError(t, err)
	require.Equal(t, "db.example.com", parsed.Hostname())
}

func TestURLForRoundTripsAPasswordWithSpecialCharacters(t *testing.T) {
	got := squirrel.URLFor(squirrel.PostgresConfig{
		Host:     "host",
		Port:     5432,
		Database: "db",
		User:     "user",
		Password: "p/a@ss",
	})

	parsed, err := url.Parse(got)
	require.NoError(t, err)

	password, ok := parsed.User.Password()
	require.True(t, ok)
	require.Equal(t, "p/a@ss", password)
}
