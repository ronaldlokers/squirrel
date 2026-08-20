package squirrel

import (
	"regexp"
	"strconv"
)

// ActionIntent is a tap, resolved as far as text alone can take it. The
// position still has to be looked up against a person-scoped prompt before it
// means a chore.
type ActionIntent struct {
	MessageID string
	// Kind is "done", "undefine", "snooze" or "later".
	Kind     string
	Position int
	// Mood is set for a "mood" tap and empty otherwise. It is a word rather
	// than a position because a check-in is not about a line of a list.
	Mood string
	// Selected is the resulting state Campfire reports, not a delta. false on a
	// "done" button means the completion is being taken back.
	Selected bool
}

// Anchored at both ends, like every other matcher in this system: a message
// that merely contains this shape is a thought about it, not a tap.
//
// `later` is the picker's refusal, and it is deliberately not `snooze`. A
// snooze silences a chore's asking for a day and belongs to the chore; a
// refusal says "not from you, today" to the picker and touches nothing about
// the thing itself. Sharing one word would let one press silence two surfaces.
var actionPattern = regexp.MustCompile(`^!action (\d+) (done|undefine|snooze|later):(\d{1,3}) (true|false)$`)

// A mood tap carries a word where the others carry a position. Its own pattern
// rather than a widened one: "mood:3" should not parse, because a mood is not
// the third of anything.
var moodPattern = regexp.MustCompile(`^!action (\d+) mood:([a-z]{1,12}) (true|false)$`)

// ParseAction recognises the encoding CaptureFrom writes for an action webhook.
// It is deliberately not part of Match: an action is not something a person can
// type into the room, and putting it in the intent matcher would make it one.
func ParseAction(text string) (ActionIntent, bool) {
	if m := moodPattern.FindStringSubmatch(text); m != nil {
		return ActionIntent{
			MessageID: m[1],
			Kind:      "mood",
			Mood:      m[2],
			Selected:  m[3] == "true",
		}, true
	}
	m := actionPattern.FindStringSubmatch(text)
	if m == nil {
		return ActionIntent{}, false
	}
	position, err := strconv.Atoi(m[3])
	if err != nil || position < 1 {
		return ActionIntent{}, false
	}
	return ActionIntent{
		MessageID: m[1],
		Kind:      m[2],
		Position:  position,
		Selected:  m[4] == "true",
	}, true
}
