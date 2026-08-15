package squirrel

import (
	"math"
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
		// Guard the multiplication below against int64 overflow — not a
		// human-scale cap, just the arithmetic's own ceiling.
		if int64(parsed) > math.MaxInt64/int64(unit) {
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

type IntentKind string

const (
	IntentCapture  IntentKind = "capture"
	IntentComplete IntentKind = "complete"
	IntentStop     IntentKind = "stop"
	IntentQuery    IntentKind = "query"
	IntentDrop     IntentKind = "drop"
	IntentDefine   IntentKind = "define"
)

type Intent struct {
	Kind IntentKind
	// Text is the capture text, with a leading "." stripped. Otherwise verbatim.
	Text string
	// Position is the line number a command addressed, or 0 for none.
	Position int
	Name     string
	Every    time.Duration
}

var (
	bareNumber = regexp.MustCompile(`^(\d{1,3})$`)
	doneNumber = regexp.MustCompile(`^done\s+(\d{1,3})$`)
	stopNumber = regexp.MustCompile(`^stop\s+(\d{1,3})$`)
)

// Match decides what a message is.
//
// An intent matches only if the ENTIRE trimmed message is one of its forms.
// "done with the flux migration" is a thought about a migration, not a
// completion, and losing it is the failure this whole system exists to prevent.
// When in doubt the answer is always capture.
func Match(raw string) Intent {
	trimmed := strings.TrimSpace(raw)

	// The escape hatch, and it wins over everything.
	if after, found := strings.CutPrefix(trimmed, "."); found {
		return Intent{Kind: IntentCapture, Text: strings.TrimSpace(after)}
	}

	lower := strings.ToLower(trimmed)

	switch lower {
	case "done", "did it", "✅", "✔️":
		return Intent{Kind: IntentComplete}
	case "?":
		return Intent{Kind: IntentQuery}
	case "nvm", "forget it", "never mind":
		return Intent{Kind: IntentDrop}
	}

	if m := doneNumber.FindStringSubmatch(lower); m != nil {
		return Intent{Kind: IntentComplete, Position: atoi(m[1])}
	}
	if m := bareNumber.FindStringSubmatch(lower); m != nil {
		return Intent{Kind: IntentComplete, Position: atoi(m[1])}
	}
	if m := stopNumber.FindStringSubmatch(lower); m != nil {
		return Intent{Kind: IntentStop, Position: atoi(m[1])}
	}

	// ParseEvery alone is too permissive for Match's purposes: its unit word
	// ("day", "week", ...) is common enough that plain sentences like "every
	// day i think about leaving" satisfy its pattern (colon and count are
	// both optional there by design). A digit somewhere in the message is
	// the signal that this is a deliberate "every N unit" definition and not
	// prose that happens to start with "every <unit-word>".
	if name, every, ok := ParseEvery(trimmed); ok && strings.ContainsAny(trimmed, "0123456789") {
		return Intent{Kind: IntentDefine, Name: name, Every: every}
	}

	// Verbatim: not the trimmed copy. The raw text is the record.
	return Intent{Kind: IntentCapture, Text: raw}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
