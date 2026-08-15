package squirrel

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const day = 24 * time.Hour

var unitDurations = map[string]time.Duration{
	"day": day, "days": day,
	"week": 7 * day, "weeks": 7 * day,
	// A month is 30 days. This is a nudge, not a calendar.
	"month": 30 * day, "months": 30 * day,
}

// Anchored at the start: "i vacuum every 2 weeks" is a note, not a definition.
// The colon is optional because requiring punctuation would be a command
// language to memorise, which the principles forbid.
var everyPattern = regexp.MustCompile(`^every\s+(?:(\d+)\s+)?([a-z]+)\s*:?\s+(.+)$`)

// ParseEvery recognises a chore definition. ok is false for anything not
// confidently one — the caller then treats the message as a capture, which is
// the only safe direction to be wrong in.
func ParseEvery(s string) (string, time.Duration, bool) {
	trimmed := strings.TrimSpace(s)
	m := everyPattern.FindStringSubmatch(strings.ToLower(trimmed))
	if m == nil {
		return "", 0, false
	}

	unit, ok := unitDurations[m[2]]
	if !ok {
		return "", 0, false
	}

	count := 1
	if m[1] != "" {
		parsed, err := strconv.Atoi(m[1])
		if err != nil || parsed < 1 {
			return "", 0, false
		}
		count = parsed
	}

	// Take the name from the original string rather than the lowercased copy,
	// so it is stored as first written.
	name := strings.TrimSpace(trimmed[len(trimmed)-len(m[3]):])
	if name == "" {
		return "", 0, false
	}
	return name, time.Duration(count) * unit, true
}
