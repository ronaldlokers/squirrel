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
// So: the sentences you meet most often have more than one wording, the stamp
// lands at a different angle, and the room's light falls from a different
// place. Nothing else moves. Every control keeps its label, because muscle
// memory is what Principle 6's "the same every time" protects, and a button
// that renames itself is a button you have to read again.
//
// The two that are not sentences are both *art* rather than layout, which is
// the other half of the roadmap's four words. Neither moves anything you aim
// at: the stamp is a result you read after the fact, and the light is the
// room. A control that shifted by a few degrees would be a different thing
// entirely, and is not on the table.

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

	// The acknowledgements. Every one of these is met several times in a
	// single sitting now that the app is one conversation, which is far more
	// often than anything above: the four wordings the slot has are read once
	// a visit, and "Good." was read every time anything at all happened.
	//
	// A conversation whose every reply is the same word is a conversation with
	// a machine, and this product's whole argument is that it is not one.

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
		// "that was a session" was here from #129 until 26 August 2026. The
		// stopping screen's own test has banned the word since #73 — it is on
		// a list with "cleared", "progress" and "streak" — and the phrase was
		// added later without anyone noticing, because a saying is chosen from
		// the day and this one only came up now and then. The screen failed
		// its own test roughly one day in six.
		//
		// Removed rather than un-banned: naming what just happened as "a
		// session" makes it a unit of work, which is the thing this screen
		// exists not to do.
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

// Tilt is the angle the stamp lands at today, in degrees.
//
// It shipped pinned at -7, and the band is eight degrees wide around it. The
// ceiling is not arbitrary: past about a dozen degrees a stamp stops reading
// as slapped on and starts reading as crooked, and the word inside it gets
// harder to read at a glance — which is the one job it has.
//
// Negative throughout, so it always leans the same way. A stamp that tipped
// left on Tuesday and right on Wednesday would be two different objects.
func Tilt(on time.Time) int { return pick("tilt", on, TiltFrom, TiltTo) }

// LightFrom and LightTo bound where the room's light falls, across the field.
const (
	LightFrom = 8
	LightTo   = 26
)

// Light is where the field's highlight sits today, as a percentage across.
//
// Across only. The vertical stays where it is and so does the alpha, because
// .35 is a measured contrast result rather than a taste one — cream on the lit
// centre reads 4.8:1 there and failed at .5. Sliding the centre sideways moves
// the same light without changing how bright it ever gets, which is the only
// version of this that cannot quietly undo that measurement.
func Light(on time.Time) int { return pick("light", on, LightFrom, LightTo) }
