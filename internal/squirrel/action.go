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
	// Kind is "done", "undefine" or "snooze".
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
var actionPattern = regexp.MustCompile(`^!action (\d+) (done|undefine|snooze):(\d{1,3}) (true|false)$`)

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
