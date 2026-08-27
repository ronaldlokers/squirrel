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

// Unstuck is one answer: a line, and at most one thing to press. The shape of the
// struct is what makes a twelve-step productivity answer impossible — one
// sentence and one number, with nowhere to put a second step.
type Unstuck struct {
	// Line is what Squirrel says. One sentence, lower case, no exclamation.
	Line string
	// Minutes is a timer worth offering, or zero for none.
	Minutes int
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
		return Unstuck{
			Line:    "forget the rest of it. just do the smallest piece you can see.",
			Minutes: 5,
		}
	case BlockerHow:
		// The answer to not knowing how is not an instruction, it is the
		// question underneath — and that question is a thought, so it goes
		// where thoughts go rather than into a field that throws it away.
		return Unstuck{
			Line: "what is the first thing you would have to find out?",
			Ask:  true,
		}
	case BlockerBoring:
		// The only branch where a timer is the whole answer. Boring is what a
		// body double is for, and the going is the point.
		return Unstuck{
			Line:    "ten minutes, and I will say when. stop wherever you are.",
			Minutes: 10,
		}
	case BlockerNotToday:
		return Unstuck{Refuse: true}
	}
	return Unstuck{}
}
