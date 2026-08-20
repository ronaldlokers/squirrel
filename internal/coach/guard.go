package coach

import (
	"strings"
	"unicode"
)

// The guard is about shape, not content.
//
// Principle 5 was opened on 20 August 2026: the coach may evaluate, compare,
// and mention counts and streaks. That decides *what* it may say, and nothing
// here second-guesses it — there is no word list, no sentiment check, no
// judgement about tone.
//
// What is still enforced is *form*, because form is where this product's whole
// failure mode lives. The thing being prevented is the twelve-step plan: a
// wall of numbered steps handed to someone who said they were overwhelmed. A
// model that returns one is not being unkind, it is being a chatbot, and the
// difference between those is a list.
//
// A cheaper model needs this more, not less.
//
// Anything failing is discarded and the deterministic answer used. There is no
// retry: a retry is a second chance to say something worse, and the fixed line
// underneath is already acceptable.

// maxReply is the ceiling in characters. Two sentences of plain English fit in
// far less; this is a backstop against a model that ignores the prompt
// entirely, not a target to write up to.
const maxReply = 400

// maxSentences allows a little slack over the two the prompt asks for. The
// prompt is where the intent lives; this only refuses an essay.
const maxSentences = 3

// Guard returns the reply to show, and whether it is usable at all.
//
// It trims, because trailing whitespace is not a violation of anything. It
// does not otherwise rewrite: a reply that has to be repaired to be acceptable
// is a reply the deterministic line should have handled.
func Guard(text string) (string, bool) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", false
	}
	if len([]rune(t)) > maxReply {
		return "", false
	}
	if strings.Contains(t, "```") {
		return "", false
	}
	// A blank line means paragraphs, and paragraphs mean a document.
	if strings.Contains(t, "\n\n") {
		return "", false
	}
	if countSentences(t) > maxSentences {
		return "", false
	}
	for _, line := range strings.Split(t, "\n") {
		if isListOrHeading(strings.TrimSpace(line)) {
			return "", false
		}
	}
	return t, true
}

// isListOrHeading reports the shapes that turn an answer into a plan: bullets,
// numbered steps, and markdown headings.
func isListOrHeading(line string) bool {
	if line == "" {
		return false
	}
	switch line[0] {
	case '#', '>':
		return true
	}
	// "- ", "* ", "• " — the marker plus a space, so a sentence opening with a
	// dash or an asterisk in ordinary prose is not mistaken for a bullet.
	for _, bullet := range []string{"- ", "* ", "+ ", "• ", "– "} {
		if strings.HasPrefix(line, bullet) {
			return true
		}
	}
	// "1. ", "2) " — a small number followed by punctuation and a space. Any
	// leading digit run is checked rather than only a single digit, because
	// "10. " is as much a list as "1. ".
	digits := 0
	for digits < len(line) && line[digits] >= '0' && line[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits+1 < len(line) &&
		(line[digits] == '.' || line[digits] == ')') && line[digits+1] == ' ' {
		return true
	}
	return false
}

// countSentences counts terminators followed by a space or the end of the
// text.
//
// Deliberately crude, and it errs toward under-counting: "e.g." and "5 p.m."
// would each read as a sentence end to a stricter reader, and refusing a good
// reply over an abbreviation is a worse failure than letting a third sentence
// through.
func countSentences(t string) int {
	n := 0
	runes := []rune(t)
	for i, r := range runes {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
			n++
		}
	}
	if n == 0 {
		// No terminator at all is still one thing said.
		return 1
	}
	return n
}
