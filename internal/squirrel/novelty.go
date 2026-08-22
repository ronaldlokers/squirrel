package squirrel

import (
	"hash/fnv"
	"time"
)

// Habituation is the enemy, and the stack is not the only thing that can move.
//
// PRODUCT.md's own accessibility premise: a surface that looks identical every
// time stops being seen within about a week. The card stack has been
// randomised since it was drawn, for exactly that reason — and it was the only
// thing in the product that was. "One frame", which made every screen more
// uniform on purpose, started a clock on the rest.
//
// The roadmap sanctions this as an experiment, in four words that are the
// whole of its scope: novelty in **art and phrasing only**.
//
// So: the sentences you meet most often have more than one wording. Nothing
// else moves. Every control keeps its label, because muscle memory is what
// Principle 6's "the same every time" protects, and a button that renames
// itself is a button you have to read again.

// Saying is one of the places a sentence varies.
type Saying string

const (
	// SayingSlot is the empty capture box, met more often than anything else
	// in the product.
	SayingSlot Saying = "slot"
	// SayingOffer is the label over the one thing Squirrel is holding out.
	SayingOffer Saying = "offer"
	// SayingStop is the way out of the deck, which has to read as an invitation
	// rather than as an exit sign.
	SayingStop Saying = "stop"
	// SayingEnough is the stopping screen's own line — the product's signature
	// beat, and the one it can least afford to have stopped being read.
	SayingEnough Saying = "enough"
)

// sayings are the wordings, all of which mean exactly what the original did.
//
// The first of each is the wording that shipped, so a day that picks index
// zero is the product as it was. None of them asks for anything, none counts
// anything, and none is cleverer than the thing it replaces — a line you
// notice for being clever is a line you notice instead of the note.
var sayings = map[Saying][]string{
	SayingSlot: {
		"tell me a thing",
		"say it here",
		"what is it",
		"go on then",
		"put it down here",
		"tell me",
		"what have you got",
		"anything at all",
	},
	SayingOffer: {
		"RIGHT NOW",
		"THIS ONE",
		"THE ONE THING",
		"HOW ABOUT THIS",
		"THIS, MAYBE",
		"START HERE",
	},
	SayingStop: {
		"stop whenever you like",
		"stop when you want to",
		"you can stop here",
		"leave it whenever",
		"stopping is fine",
		"that is enough whenever you say",
	},
	SayingEnough: {
		"that will do",
		"that is enough",
		"good enough",
		"that was a session",
		"you did some",
		"leave the rest",
	},
}

// Say picks the wording for a day.
//
// Deterministic on the date, not random, and that is the whole design:
//
//   - Both viewports agree. The phone and the desktop are one product and a
//     line that differs between them is a bug rather than a variation.
//   - A reload is not a slot machine. Text that changes while you are reading
//     it is worse than text that never changes.
//   - It is Squirrel's, not a model's. Principle 8 draws the line at
//     authorship: the rules produced these words, so they are the product's
//     own voice, and the deterministic floor never needs a key to speak.
//
// The day is the unit because habituation is measured in days. An hour would
// be a fidget; a week is longer than the thing it treats.
func Say(what Saying, on time.Time) string {
	pool := sayings[what]
	if len(pool) == 0 {
		return ""
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(string(what)))
	_, _ = h.Write([]byte(on.Format("2006-01-02")))
	return pool[int(h.Sum32()%uint32(len(pool)))]
}

// Sayings is every wording of one thing, for a test that has to check them all
// against the rules they are held to.
func Sayings(what Saying) []string { return sayings[what] }
