package squirrel

import (
	"hash/fnv"
	"time"
)

// Habituation is the enemy: a surface that looks identical every time stops being
// seen within about a week.
//
// Scope, from the roadmap: novelty in art and phrasing only. The sentences met
// most often have more than one wording, the stamp lands at a different angle,
// and the room's light falls from a different place. Nothing else moves — every
// control keeps its label, because a button that renames itself is one you have
// to read again.

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

	// The acknowledgements, met several times a sitting now the app is one
	// conversation. A conversation whose every reply is the same word is a
	// conversation with a machine.

	// SayingDid is what Buddy says when you finished something.
	SayingDid Saying = "did"
	// SayingKept is a note put on the shelf rather than ended.
	SayingKept Saying = "kept"
	// SayingDropped is a note thrown away. Quieter than the rest: this one is
	// never congratulated, because throwing a thought away is not an
	// achievement and saying so would make it one.
	SayingDropped Saying = "dropped"
	// SayingDecided is a note that became a task.
	SayingDecided Saying = "decided"
	// SayingHere is Buddy handing you a note out of the pile, which is the
	// single most repeated sentence in the product.
	SayingHere Saying = "here"
	// SayingLater is skipping one without deciding.
	SayingLater Saying = "later"
	// SayingHeard is a reply that landed badly, taken.
	SayingHeard Saying = "heard"
)

// sayings are the wordings, all meaning what the original did. The first of each
// is what shipped, so a day that picks index zero is the product as it was. None
// is cleverer than the thing it replaces: a line you notice for being clever is a
// line you notice instead of the note.
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
		// "that was a session" was here from #129 until 26 August 2026. The stopping
		// screen's own test has banned the word since #73, and the phrase was added later
		// without anyone noticing — a saying is chosen from the day, so the screen failed
		// its own test roughly one day in six.
		//
		// Removed rather than un-banned: naming what just happened as "a session" makes
		// it a unit of work.
		"you did some",
		"leave the rest",
	},
	SayingDid: {
		"Good.",
		"That is done.",
		"Off the list.",
		"Right, done.",
		"Good — that one is finished.",
		"Done.",
	},
	SayingKept: {
		"Kept.",
		"On the shelf.",
		"Put away.",
		"Kept — it is on the shelf.",
		"Filed.",
	},
	SayingDropped: {
		"Gone.",
		"Thrown away.",
		"Dropped.",
		"That one is gone.",
	},
	SayingDecided: {
		"On the list.",
		"That is a task now.",
		"Decided.",
		"Right — on the list.",
	},
	SayingHere: {
		"This one.",
		"Here is one.",
		"Next.",
		"How about this one.",
		"And this.",
		"Here.",
	},
	SayingLater: {
		"Fine.",
		"Later, then.",
		"Leave it.",
		"All right.",
	},
	SayingHeard: {
		"Noted.",
		"Taken.",
		"Understood.",
		"Right — noted.",
	},
}

// Say picks the wording for a day, deterministic on the date rather than random:
// both viewports agree, a reload is not a slot machine, and the words are the
// product's own rather than a model's.
//
// The day is the unit because habituation is measured in days.
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

// pick is the same day-seeded choice Say makes, over a range of numbers.
//
// Same hash, same unit, same reasons — one implementation so the stamp and the
// sentences cannot drift into disagreeing about what day it is.
func pick(salt string, on time.Time, from, to int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(salt))
	_, _ = h.Write([]byte(on.Format("2006-01-02")))
	return from + int(h.Sum32()%uint32(to-from+1))
}

// TiltRange is how far the stamp can lean, in degrees. Exported so the test
// that holds this to "a few degrees" reads the number rather than repeating it.
const (
	TiltFrom = -11
	TiltTo   = -3
)

// Tilt is the angle the stamp lands at today. Eight degrees around the -7 it
// shipped at: past about a dozen a stamp reads as crooked rather than slapped on.
// Negative throughout, so it always leans the same way.
func Tilt(on time.Time) int { return pick("tilt", on, TiltFrom, TiltTo) }

// LightFrom and LightTo bound where the room's light falls, across the field.
const (
	LightFrom = 8
	LightTo   = 26
)

// Light is where the field's highlight sits today, as a percentage across.
//
// Across only. The vertical and the alpha stay, because .35 is a measured
// contrast result — cream on the lit centre reads 4.8:1 and failed at .5. Sliding
// sideways cannot quietly undo that measurement.
func Light(on time.Time) int { return pick("light", on, LightFrom, LightTo) }
