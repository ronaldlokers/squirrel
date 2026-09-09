package squirrel

import (
	"context"
	"strings"
)

// I can't start.
//
// Knowing what to do is not being able to do it. This is the ladder, and it is
// deterministic: four answers, each producing one line and at most one control. A
// model may later replace the "too big" sentence with a generated first step, and
// the sentence stays underneath as the fallback.
//
// Two options are deliberately absent. "No energy" is what the check-in already
// says, and "anxious" invites a therapeutic response this product should not
// attempt — its useful action, make the thing smaller, is already first.

// Blocker is what is in the way. Four, and never more: a list of reasons is
// itself a decision, and this is being read by someone who has just said they
// cannot make one.
type Blocker string

const (
	BlockerBig      Blocker = "big"
	BlockerHow      Blocker = "how"
	BlockerBoring   Blocker = "boring"
	BlockerNotToday Blocker = "not today"
)

// Blockers is the order they are offered in, everywhere. One order, so the two
// surfaces cannot disagree about which sits where.
var Blockers = []Blocker{BlockerBig, BlockerHow, BlockerBoring, BlockerNotToday}

// BlockerWords is what each is called. The words are the question's answers
// rather than labels for a category: someone says "it's too big", not "size".
var BlockerWords = map[Blocker]string{
	BlockerBig:      "too big",
	BlockerHow:      "don't know how",
	BlockerBoring:   "boring",
	BlockerNotToday: "not today",
}

// ParseBlocker reads what was typed or pressed generously, because this arrives
// from someone who is stuck: "too big", "big" and "TOO BIG" are the same answer.
func ParseBlocker(s string) (Blocker, bool) {
	t := strings.ToLower(strings.TrimSpace(s))
	switch {
	case t == "":
		return "", false
	case strings.Contains(t, "big"), strings.Contains(t, "huge"), strings.Contains(t, "much"):
		return BlockerBig, true
	case strings.Contains(t, "how"), strings.Contains(t, "know"), strings.Contains(t, "start where"):
		return BlockerHow, true
	case strings.Contains(t, "bor"), strings.Contains(t, "dull"):
		return BlockerBoring, true
	case strings.Contains(t, "not today"), strings.Contains(t, "later"), strings.Contains(t, "nah"):
		return BlockerNotToday, true
	}
	return "", false
}

// Unstuck is one answer: one sentence, and nowhere to put a second step. It
// offered a timer beside the sentence until the body double was retired on
// 9 September 2026; what is left is the sentence, which was always the part
// that did the work.
type Unstuck struct {
	// Line is what Squirrel says. One sentence, lower case, no exclamation.
	Line string
	// Ask means the answer is a question, and the reply becomes a note. It is
	// the only branch that captures, and it captures because "what would I
	// have to find out first" is a thought, and thoughts go in the pile.
	Ask bool
	// Refuse means this was not an obstacle but a no. The caller turns the
	// offer down and says nothing further.
	Refuse bool
}

// Breaker is the seam a model breaks a thing into steps through, or nil. It
// reports false for everything — no coach, no budget, a model that numbered its
// steps — and false means the fixed line stands.
type Breaker func(ctx context.Context, personID int64, task, blocker string) ([]string, bool)

// BreakingHelps reports whether a breakdown answers this blocker. Only "too big":
// "don't know how" ends in a question whose answer is a thought, "boring" ends in
// a timer, and "not today" is not an obstacle.
func BreakingHelps(b Blocker) bool { return b == BlockerBig }

// UnstuckFor is the ladder. Every branch ends in something smaller than the
// thing that could not be started, and none of them ends in a question about
// why you could not start it.
func UnstuckFor(b Blocker) Unstuck {
	switch b {
	case BlockerBig:
		// Not "break it down", which is a second job. The smallest visible
		// piece is a thing you can see from where you are standing, and seeing
		// it is most of starting it.
		return Unstuck{Line: "forget the rest of it. just do the smallest piece you can see."}
	case BlockerHow:
		// The answer to not knowing how is not an instruction, it is the
		// question underneath — and that question is a thought, so it goes
		// where thoughts go rather than into a field that throws it away.
		return Unstuck{
			Line: "what is the first thing you would have to find out?",
			Ask:  true,
		}
	case BlockerBoring:
		// Boring is the one an alarm used to answer. Without it the sentence
		// has to carry it alone, and what it says is the same thing: a short
		// go, ended by you rather than by a bell.
		return Unstuck{Line: "give it a short go and stop wherever you get to."}
	case BlockerNotToday:
		return Unstuck{Refuse: true}
	}
	return Unstuck{}
}
