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
	// The words people actually reach for. A fortnight is not a new rhythm,
	// it is two weeks said in one word, and refusing it because the table did
	// not have it is the product being difficult about English.
	"fortnight": 14 * day, "fortnights": 14 * day,
	"quarter": 90 * day, "quarters": 90 * day,
	"year": 365 * day, "years": 365 * day,
	// A weekday rhythm is daily; which days it may be raised on is a separate
	// question, answered by Asking rather than by the interval.
	"weekday": day, "weekdays": day,
}

// Anchored at the start: "i vacuum every 2 weeks" is a note, not a definition.
// The colon is optional because requiring punctuation would be a command
// language to memorise, which the principles forbid. Case-insensitive (the
// (?i) flag) so it runs directly against the original string rather than a
// lowercased copy: strings.ToLower is not byte-length-preserving (it grows
// some runes and shrinks others), so a length measured on a lowercased copy
// and then used to index the original string can go out of range or land
// mid-rune.
var everyPattern = regexp.MustCompile(`(?i)^every\s+(?:(\d+)\s+)?(?:(other)\s+)?([a-z]+)\s*:?\s+(.+)$`)

// ParseEvery recognises a chore definition. ok is false for anything not
// confidently one — the caller then treats the message as a capture, which is
// the only safe direction to be wrong in.
func ParseEvery(s string) (string, time.Duration, bool) {
	name, every, _, ok := ParseEveryAsking(s)
	return name, every, ok
}

// ParseEveryAsking is ParseEvery plus the preference the same sentence
// sometimes carries.
//
// "Every other tuesday" is two facts, not one: a fortnightly rhythm, and a
// preference for which day to raise it on. Reading it as a rhythm alone would
// throw away the half the person cared enough to type, and reading it as a
// calendar entry would make it a deadline. So both are kept, apart, and only
// the first one decides when the chore is due.
func ParseEveryAsking(s string) (string, time.Duration, Asking, bool) {
	trimmed := strings.TrimSpace(s)
	m := everyPattern.FindStringSubmatch(trimmed)
	if m == nil {
		return "", 0, Asking{}, false
	}

	word := strings.ToLower(m[3])
	asking := Asking{}

	// "every tuesday" and "every other tuesday": weekly and fortnightly, both
	// raised on that day.
	unit, ok := unitDurations[word]
	if !ok {
		w, isDay := weekdayNames[word]
		if !isDay {
			return "", 0, Asking{}, false
		}
		unit, asking.Days = 7*day, OnlyOn(w)
	}
	if word == "weekday" || word == "weekdays" {
		asking.Days = Weekdays
	}

	count := 1
	// "other" is two, and it does not combine with a number: "every other 3
	// weeks" is not a thing anyone means.
	if m[2] != "" {
		if m[1] != "" {
			return "", 0, Asking{}, false
		}
		count = 2
	}
	if m[1] != "" {
		parsed, err := strconv.Atoi(m[1])
		if err != nil || parsed < 1 {
			return "", 0, Asking{}, false
		}
		// Guard the multiplication below against int64 overflow — not a
		// human-scale cap, just the arithmetic's own ceiling.
		if int64(parsed) > math.MaxInt64/int64(unit) {
			return "", 0, Asking{}, false
		}
		count = parsed
	}

	// m[3] is captured straight from trimmed (the pattern is matched
	// case-insensitively against it directly, never against a lowercased
	// copy), so it is already the name as first written, byte-for-byte.
	name := strings.TrimSpace(m[4])
	if name == "" {
		return "", 0, Asking{}, false
	}
	return name, time.Duration(count) * unit, asking, true
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
	IntentMoment   IntentKind = "moment"
)

type Intent struct {
	Kind IntentKind
	// Text is the capture text, with a leading "." stripped. Otherwise verbatim.
	Text string
	// Position is the line number a command addressed, or 0 for none.
	Position int
	Name     string
	Every    time.Duration
	// At is the fixed point a message turned out to be, when it did.
	At Moment
	// Ask is when a definition said it was worth raising, if it said.
	Ask Asking
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
	// "other" is the one word allowed between "every" and the unit, so that
	// "every other tuesday: bins out" is as deliberate as "every tuesday: bins
	// out". It does not make the bare form deliberate — "every other tuesday i
	// see my mum" is prose, and the colon is still what tells them apart.
	deliberateDefine = regexp.MustCompile(`^every\s+(?:\d+\s|(?:other\s+)?[a-z]+\s*:)`)
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

	if name, every, ask, ok := ParseEveryAsking(trimmed); ok && deliberateDefine.MatchString(lower) {
		return Intent{Kind: IntentDefine, Name: name, Every: every, Ask: ask}
	}

	// A time the world imposed. The bar is the same one the chore definition
	// above sets, and ParseMoment sets it high on purpose: "at" or "tomorrow"
	// in front of a real clock time. A bare "14:30 dentist" stays a note,
	// because someone writing a thought down should never have to escape it,
	// and the cost of being wrong in this direction is one command to say it
	// again rather than a warning that never comes.
	//
	// The clock comes from time.Now rather than from a parameter, which is the
	// one impurity in this function. Match is called on stored rows too — see
	// CapturesSince — and there it only ever asks whether something was a
	// capture, so the date it would resolve to is never read.
	if m, ok := ParseMoment(trimmed, time.Now()); ok {
		return Intent{Kind: IntentMoment, At: m}
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
