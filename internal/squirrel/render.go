package squirrel

import (
	"fmt"
	"strings"
)

// NotesMessage renders a numbered list of notes — the pile, or the results of
// a search.
//
// `more` is a bool rather than a count, and that is the no-counting rule
// expressed in the signature: this function could not render a total if a later
// author wanted one, because the number never reaches it. A count of untriaged
// notes beside an implied target of zero is the accumulating mechanism this
// project bans.
//
// No buttons. The pile is answered by typing a number — `done 2`, `keep 2`,
// `drop 2` — and a twenty-line list cannot carry a button per line anyway;
// phase 3 capped actions at twelve for exactly that reason.
func NotesMessage(items []Item, more bool) Message {
	if len(items) == 0 {
		return Message{Text: "Nothing in the pile."}
	}

	var b strings.Builder
	for i, it := range items {
		fmt.Fprintf(&b, " %d. %s\n", i+1, it.RawText)
	}
	if more {
		b.WriteString("…and more.")
	}
	return Message{Text: strings.TrimRight(b.String(), "\n")}
}

// HelpMessage is the vocabulary. Until now it existed nowhere, so the only way
// to learn what Squirrel understood was to have written it.
//
// The capture rule comes first because it is the one that matters: if you
// remember nothing else, typing a thought stores the thought.
func HelpMessage() Message {
	return Message{Text: strings.Join([]string{
		"Anything you type is a note. That is the default and it always wins.",
		"",
		"!notes — the pile, newest first",
		"!find <text> — search everything you have told me",
		"!chores — what is due (same as ?)",
		"!chore <n> every <interval> — turn note n into a chore",
		"",
		"done <n> · keep <n> · drop <n> — clear line n",
		"done — the one thing outstanding",
		"nvm — undo a chore I just made from a note",
		"",
		"Start with a dot to store something I would otherwise read as a command.",
	}, "\n")}
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

// choreSentence is choreLine without the list numbering, for when a chore is
// named on its own rather than as item N of a list.
func choreSentence(c Chore) string {
	return fmt.Sprintf("%s — %s, usually %d", c.Name, plural(c.SinceDays, "day"), c.EveryDays)
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
	return fmt.Sprintf("%s, every %s. First nudge in %s.\nnvm if you meant that as a note.",
		c.Name, plural(c.EveryDays, "day"), plural(c.EveryDays, "day"))
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

// EveningMessage is the once-a-day message that always runs: what you did,
// what you captured, and — only when no nudge fired earlier — the chore that
// would otherwise have arrived as a second notification a second apart.
//
// When nothing was completed the section is absent rather than empty. An empty
// list is a scoreboard reading nil; an absent section says nothing about you.
func EveningMessage(completed []string, captures []string, nudge *Chore) Message {
	var b strings.Builder

	if nudge != nil {
		b.WriteString(choreSentence(*nudge))
		b.WriteString("\n")
	}
	if len(completed) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Today\n")
		for _, name := range completed {
			fmt.Fprintf(&b, " · %s\n", name)
		}
	}
	if len(captures) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Since yesterday\n")
		for _, text := range captures {
			fmt.Fprintf(&b, " · %s\n", text)
		}
	}

	m := Message{Text: strings.TrimRight(b.String(), "\n")}
	if nudge != nil {
		m.SelectionMode = "single"
		m.Actions = []Action{{Label: nudge.Name, Value: "done:1", Emoji: "✅"}}
	}
	return m
}
