package squirrel

import (
	"fmt"
	"strings"
)

// RenderDigest is the daily message. Facts only: "19 days, usually 14" is the
// entire editorial position, and it does not change as the number grows. No
// streaks, no counts of times missed, no escalation. Shame is not a feature.
//
// Returns "" when there is nothing to say, because a daily "nothing to report"
// is how you teach someone to skip the message.
func RenderDigest(due []Chore, captures []string) string {
	var b strings.Builder

	if len(due) > 0 {
		b.WriteString("Due\n")
		for i, c := range due {
			fmt.Fprintf(&b, " %d. %s\n", i+1, choreLine(c))
		}
	}

	if len(captures) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Since yesterday\n")
		for _, c := range captures {
			fmt.Fprintf(&b, " · %s\n", c)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func choreLine(c Chore) string {
	return fmt.Sprintf("%s — %s, usually %d", c.Name, plural(c.SinceDays, "day"), c.EveryDays)
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

func RenderList(chores []Chore) string {
	if len(chores) == 0 {
		return "No chores yet. Say something like: every 2 weeks vacuum"
	}

	var b strings.Builder
	for i, c := range chores {
		fmt.Fprintf(&b, " %d. %s\n", i+1, choreLine(c))
	}
	return strings.TrimRight(b.String(), "\n")
}

// RenderDefined confirms a definition and offers the undo in the same breath,
// because the matcher will sometimes turn a note into a chore and the
// correction has to cost one word.
func RenderDefined(c Chore) string {
	return fmt.Sprintf("%s, every %d days. First nudge in %d days.\nnvm if you meant that as a note.",
		c.Name, c.EveryDays, c.EveryDays)
}

// The rendered text is unchanged from phase 2 — buttons are a second path to
// the same intent, never a replacement for the numbers. Someone reading a
// notification without opening the app still types `done 2`.

func actionsForChores(chores []Chore, kind, emoji string) []Action {
	out := make([]Action, 0, len(chores))
	for i, c := range chores {
		out = append(out, Action{
			Label: c.Name,
			Value: fmt.Sprintf("%s:%d", kind, i+1),
			Emoji: emoji,
		})
	}
	return out
}

func DigestMessage(due []Chore, captures []string) Message {
	m := Message{Text: RenderDigest(due, captures)}
	if len(due) > 0 {
		m.SelectionMode = "multiple"
		m.Actions = actionsForChores(due, "done", "✅")
	}
	return m.Capped()
}

func ListMessage(chores []Chore) Message {
	m := Message{Text: RenderList(chores)}
	if len(chores) > 0 {
		m.SelectionMode = "multiple"
		m.Actions = actionsForChores(chores, "done", "✅")
	}
	return m.Capped()
}

// One button, and it is the correction. There is no confirm: doing nothing
// already means right, and a confirm button is a decision you would have to
// make every time you spoke to it.
func DefinedMessage(c Chore) Message {
	return Message{
		Text:          RenderDefined(c),
		SelectionMode: "single",
		Actions: []Action{{
			Label: "make it a note",
			Value: "undefine:1",
			Emoji: "📝",
		}},
	}
}
