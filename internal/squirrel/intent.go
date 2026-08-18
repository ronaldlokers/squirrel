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
// language to memorise, which the principles forbid. Case-insensitive (the
// (?i) flag) so it runs directly against the original string rather than a
// lowercased copy: strings.ToLower is not byte-length-preserving (it grows
// some runes and shrinks others), so a length measured on a lowercased copy
// and then used to index the original string can go out of range or land
// mid-rune.
var everyPattern = regexp.MustCompile(`(?i)^every\s+(?:(\d+)\s+)?([a-z]+)\s*:?\s+(.+)$`)

// ParseEvery recognises a chore definition. ok is false for anything not
// confidently one — the caller then treats the message as a capture, which is
// the only safe direction to be wrong in.
func ParseEvery(s string) (string, time.Duration, bool) {
	trimmed := strings.TrimSpace(s)
	m := everyPattern.FindStringSubmatch(trimmed)
	if m == nil {
		return "", 0, false
	}

	unit, ok := unitDurations[strings.ToLower(m[2])]
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

	// m[3] is captured straight from trimmed (the pattern is matched
	// case-insensitively against it directly, never against a lowercased
	// copy), so it is already the name as first written, byte-for-byte.
	name := strings.TrimSpace(m[3])
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
	IntentCommand  IntentKind = "command"
	IntentKeep     IntentKind = "keep"
)

type Intent struct {
	Kind IntentKind
	// Text is the capture text, with a leading "." stripped. Otherwise verbatim.
	Text string
	// Position is the line number a command addressed, or 0 for none.
	Position int
	Name     string
	Every    time.Duration
	// Command is the word after "!", lowercased — commands are typed, and a
	// capitalised "!Find" is the same request as "!find".
	Command string
	// Arg is everything after the command word, trimmed but otherwise
	// verbatim. Its case is the user's: it is matched against text they wrote,
	// and it becomes a chore's name on the promotion path.
	Arg string
}

var (
	bareNumber = regexp.MustCompile(`^(\d{1,3})$`)
	doneNumber = regexp.MustCompile(`^done\s+(\d{1,3})$`)
	stopNumber = regexp.MustCompile(`^stop\s+(\d{1,3})$`)
	keepNumber = regexp.MustCompile(`^keep\s+(\d{1,3})$`)
	dropNumber = regexp.MustCompile(`^drop\s+(\d{1,3})$`)

	// A command name is a word. Without this, "!!!" is a command called "!!"
	// and "!?" is one called "?" — punctuation someone typed, answered with a
	// help message instead of being remembered.
	commandName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

	// ParseEvery answers "does this have the shape"; Match decides policy, and
	// Match is its only caller. ParseEvery deliberately makes both the count and
	// the colon optional, which leaves `every <unit> <rest>` matching ordinary
	// prose — "every day i think about leaving" parses as a daily chore named "i
	// think about leaving". A count or a colon is the mark of a deliberate
	// definition, so Match requires one. The cost is that a bare "every day meds"
	// is filed as a note; the alternative is a chore that nags forever with a
	// sentence that was only ever a thought.
	deliberateDefine = regexp.MustCompile(`^every\s+(?:\d+\s|[a-z]+\s*:)`)
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

	// A tap's own text begins with "!" as well: phase 3 encodes one as
	// "!action <message id> <value> <selected>". Text of that shape reaches
	// Match only when the payload proved it was NOT a genuine tap — someone
	// typed it by hand — and phase 3 settled that such text is a thought.
	// Without this it becomes an unknown command and is answered with a help
	// message instead of being remembered, which is losing a thought: the one
	// failure this system exists to prevent.
	//
	// CapturesSince runs Match over stored rows for exactly this reason, so the
	// two paths have to agree here or a typed tap vanishes from the evening
	// list as well.
	if _, isAction := ParseAction(trimmed); isAction {
		return Intent{Kind: IntentCapture, Text: raw}
	}

	// `!` is a prefix rather than a keyword, and that is load-bearing. Every
	// bare word is a capture by design, so a keyword-triggered command would
	// eat "find my keys" and "notes to self about the boiler" — both thoughts,
	// and losing a thought is the failure this system exists to prevent. Phase
	// 2 met the same trap with `every day i think about leaving` and answered
	// it the same way: make the deliberate form unambiguous rather than guess
	// at intent.
	//
	// ".!find boiler" is still literal text, which is how a thought shaped like
	// a command gets captured. That does not depend on the order of the two
	// checks: ".!find" does not start with "!", so only one prefix can ever
	// match. Moving this block above the escape hatch would change nothing,
	// which is worth saying because the placement looks load-bearing and is not.
	//
	// An unknown command stays IntentCommand rather than falling through to
	// capture. A typo answered with 👀 would be filed as a note, silently, and
	// the correction with it.
	if after, found := strings.CutPrefix(trimmed, "!"); found {
		name, arg, _ := strings.Cut(strings.TrimSpace(after), " ")
		// The name has to look like a word, or "!!!" parses as a command
		// called "!!" and "!" as one called "". Those are punctuation someone
		// typed, and the rule when in doubt is capture.
		if !commandName.MatchString(name) {
			return Intent{Kind: IntentCapture, Text: raw}
		}
		return Intent{
			Kind:    IntentCommand,
			Command: strings.ToLower(name),
			Arg:     strings.TrimSpace(arg),
		}
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
	if m := keepNumber.FindStringSubmatch(lower); m != nil {
		return Intent{Kind: IntentKeep, Position: atoi(m[1])}
	}
	// `drop 2` and a bare `nvm` are both IntentDrop, told apart by Position.
	// No collision: the bare forms are exactly "nvm", "forget it" and "never
	// mind", none of which carries a number, and "drop" is not among them.
	if m := dropNumber.FindStringSubmatch(lower); m != nil {
		return Intent{Kind: IntentDrop, Position: atoi(m[1])}
	}

	if name, every, ok := ParseEvery(trimmed); ok && deliberateDefine.MatchString(lower) {
		return Intent{Kind: IntentDefine, Name: name, Every: every}
	}

	// Verbatim: not the trimmed copy. The raw text is the record.
	return Intent{Kind: IntentCapture, Text: raw}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// matchFn indirects every call Applier.Apply and CapturesSince make to Match,
// so a test can substitute a stand-in that panics — otherwise there is no way
// to exercise the recover added around Applier.Apply and Scheduler.Once,
// since Match itself has no reachable panic once the byte-length bug above is
// fixed. Production code never reassigns it.
var matchFn = Match
