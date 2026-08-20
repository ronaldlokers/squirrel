package coach_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

func TestGuardKeepsAnOrdinaryReply(t *testing.T) {
	for _, ok := range []string{
		"Start with the envelope. It's the only one with a date on it.",
		"You've done this three days running — this is the fourth.",
		// One sentence, no terminator at all.
		"Two minutes on the top thing",
		// A dash in the middle of prose is not a bullet.
		"The bins - which you already did - are not the problem.",
		// A digit is not a numbered list without the punctuation after it.
		"15 minutes is enough for this one.",
	} {
		got, usable := coach.Guard(ok)
		require.True(t, usable, ok)
		require.Equal(t, ok, got)
	}
}

func TestGuardTrimsWhitespaceButDoesNotOtherwiseRewrite(t *testing.T) {
	got, usable := coach.Guard("  Pick the envelope.\n")
	require.True(t, usable)
	require.Equal(t, "Pick the envelope.", got)
}

// The twelve-step plan is the failure mode this exists to catch: a wall of
// steps handed to someone who said they were overwhelmed.
func TestGuardRefusesAPlan(t *testing.T) {
	for name, text := range map[string]string{
		"dash bullets":    "Here's the plan:\n- open the letter\n- read it\n- reply",
		"asterisks":       "Try this:\n* one thing\n* another thing",
		"bullet char":     "Steps:\n• first\n• second",
		"numbered":        "1. Open it\n2. Read it\n3. Reply",
		"double digits":   "10. Open it\n11. Read it",
		"paren numbered":  "1) Open it\n2) Read it",
		"heading":         "# What to do\nOpen the letter.",
		"blockquote":      "> Open the letter first.",
		"code fence":      "Run this:\n```\nls\n```",
		"paragraphs":      "Open the letter.\n\nThen reply to it.",
		"four sentences":  "Open it. Read it. Reply to it. Then rest.",
		"empty":           "",
		"only whitespace": "   \n  ",
	} {
		_, usable := coach.Guard(text)
		require.False(t, usable, name)
	}
}

func TestGuardRefusesAnEssay(t *testing.T) {
	// Under the sentence limit, over the length limit: the two checks catch
	// different failures and neither substitutes for the other.
	long := strings.Repeat("word ", 200) + "."
	_, usable := coach.Guard(long)
	require.False(t, usable)
}

// Three is the ceiling, not two: the prompt asks for two and the guard allows
// one over, so a good reply is never thrown away over a full stop.
func TestGuardAllowsThreeSentences(t *testing.T) {
	got, usable := coach.Guard("Open it. Read it. That's the whole thing.")
	require.True(t, usable)
	require.Equal(t, "Open it. Read it. That's the whole thing.", got)
}

// An abbreviation would read as a sentence end to a stricter counter. Erring
// toward letting one through beats refusing a good reply over "e.g.".
func TestGuardDoesNotCountAbbreviationsAsSentences(t *testing.T) {
	_, usable := coach.Guard("Leave at 5 p.m. so you're not rushing.")
	require.True(t, usable)
}
