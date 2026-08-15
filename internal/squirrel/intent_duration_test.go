package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestParseEveryAccepts(t *testing.T) {
	cases := []struct {
		in    string
		name  string
		every time.Duration
	}{
		{"every 2 weeks: vacuum", "vacuum", 14 * 24 * time.Hour},
		{"every 2 weeks vacuum", "vacuum", 14 * 24 * time.Hour},
		{"every week bin day", "bin day", 7 * 24 * time.Hour},
		{"every 1 week bin day", "bin day", 7 * 24 * time.Hour},
		{"every day meds", "meds", 24 * time.Hour},
		{"every 3 days water plants", "water plants", 3 * 24 * time.Hour},
		{"every 3 months change the filters", "change the filters", 90 * 24 * time.Hour},
		{"EVERY 2 WEEKS: VACUUM", "VACUUM", 14 * 24 * time.Hour},
		{"  every 2 weeks:   vacuum  ", "vacuum", 14 * 24 * time.Hour},
	}
	for _, c := range cases {
		name, every, ok := squirrel.ParseEvery(c.in)
		require.True(t, ok, c.in)
		require.Equal(t, c.name, name, c.in)
		require.Equal(t, c.every, every, c.in)
	}
}

// The whole risk of this phase lives in this test.
func TestParseEveryRejects(t *testing.T) {
	for _, in := range []string{
		"",
		"every",
		"every 2 weeks",          // no name
		"every 2 weeks:",         // no name
		"every fortnight vacuum", // unit not supported
		"every 2 lightyears x",   // unit not supported
		"everything is fine",     // must not match on a prefix
		"i vacuum every 2 weeks", // must be the start of the message
		"every 0 weeks vacuum",   // zero interval
		"every -2 weeks vacuum",  // negative
		"buy milk",
		"done",
	} {
		_, _, ok := squirrel.ParseEvery(in)
		require.False(t, ok, in)
	}
}
