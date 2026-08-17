package squirrel_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

// strings.ToLower is not byte-length-preserving: Ⱥ (U+023A, 2 bytes) lowercases
// to ⱥ (U+2C65, 3 bytes), growing the string. A name derived by measuring a
// length on the lowercased copy and then indexing the original with it used to
// walk past the end of the original string and panic. Thirteen repeats push
// the miscomputed index comfortably negative rather than leaving it one byte
// off, so the failure cannot be masked by an accidental alignment.
func TestParseEveryHandlesAGrowingRune(t *testing.T) {
	name := strings.Repeat("Ⱥ", 13)
	got, every, ok := squirrel.ParseEvery("every 1 day " + name)
	require.True(t, ok)
	require.Equal(t, name, got)
	require.Equal(t, 24*time.Hour, every)
}

// ẞ (U+1E9E, 3 bytes) lowercases to ß (U+00DF, 2 bytes) — the shrinking
// direction. Before the fix, indexing the original string with a length
// measured on the lowercased copy did not panic here (the index stayed
// non-negative) but landed mid-rune, producing an invalid-UTF-8, truncated
// name instead of the name exactly as typed.
func TestParseEveryHandlesAShrinkingRune(t *testing.T) {
	got, every, ok := squirrel.ParseEvery("every 2 weeks ẞnbox")
	require.True(t, ok)
	require.True(t, utf8.ValidString(got), "name must be valid UTF-8")
	require.Equal(t, "ẞnbox", got)
	require.Equal(t, 14*24*time.Hour, every)
}

// The whole risk of this phase lives in this test.
func TestParseEveryRejects(t *testing.T) {
	for _, in := range []string{
		"",
		"every",
		"every 2 weeks",                  // no name
		"every 2 weeks:",                 // no name
		"every fortnight vacuum",         // unit not supported
		"every 2 lightyears x",           // unit not supported
		"everything is fine",             // must not match on a prefix
		"i vacuum every 2 weeks",         // must be the start of the message
		"every 0 weeks vacuum",           // zero interval
		"every -2 weeks vacuum",          // negative
		"every 200000 days water plants", // overflows a day-unit duration
		"every 4000 months water plants", // overflows a month-unit duration
		"buy milk",
		"done",
	} {
		_, _, ok := squirrel.ParseEvery(in)
		require.False(t, ok, in)
	}
}
