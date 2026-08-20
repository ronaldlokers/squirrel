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

// TasksMessage is what you decided, numbered so `done 2` can name one.
//
// The numbers are how you point at a task and nothing else. The last of them
// happens to equal how many there are, which is true of any numbered list and
// is why the list is capped and says "…and more" rather than reporting a
// total: what is refused is a number that means "how much is outstanding".
//
// Nothing decided is stated plainly and nothing is suggested. An empty task
// list is a normal state, not a failure to set up.
func TasksMessage(items []Item, more bool) Message {
	if len(items) == 0 {
		return Message{Text: "Nothing decided."}
	}

	var b strings.Builder
	b.WriteString("What you decided\n")
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
// screenURL is where the screen lives, or empty when nobody has said. It is a
// package-level value rather than a parameter because the help message and the
// evening message both want it and neither has any other reason to know about
// configuration.
//
// Empty means chat says nothing about the screen. A link built from a guess is
// a link that 404s, and a bot that confidently sends you nowhere is worse than
// one that stays quiet.
var screenURL string

// SetScreenURL is called once at boot.
func SetScreenURL(u string) { screenURL = u }

func HelpMessage() Message {
	return Message{Text: strings.Join([]string{
		"Anything you type is a note. That is the default and it always wins.",
		"",
		"!notes — the pile, newest first",
		"!find <text> — search everything you have told me",
		"!chores — what is due (same as ?)",
		"!chore <n> every <interval> — turn note n into a chore",
		"!task <n> — a note is something you decided to do",
		"!task <words> — decide something outright",
		"!tasks — what you decided, newest first",
		"!untask <n> — back in the pile, undecided",
		"every other tuesday: bins out — a rhythm, and when to raise it",
		"!did <chore> — a chore is done, by name",
		"!retire <chore> — stop a chore coming back",
		"!snooze <chore> for <how long> — stop asking for a while",
		"",
		"done <n> · keep <n> · drop <n> — clear line n",
		"done — the one thing outstanding",
		"!fix <n> <words> — say line n differently",
		"!undo — put the last note I cleared back in the pile",
		"nvm — undo a chore I just made from a note",
		"",
		"Start with a dot to store something I would otherwise read as a command.",
	}, "\n") + screenLine()}
}

// choreLine is a chore as a line of a list. The words are ChoreWords', because
// the screen says the same thing about the same chore.
func choreLine(c Chore) string {
	return ChoreWords(c)
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
	return ChoreWords(c)
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
		// The same pair the nudge carries. The evening message is the other
		// place a chore is raised, and a chore raised in two places that can
		// be answered two different ways is two views disagreeing.
		m.Actions = []Action{
			{Label: nudge.Name, Value: "done:1", Emoji: "✅"},
			{Label: "not today", Value: "snooze:1", Emoji: "🌙"},
		}
	}
	return m
}

// CheckinQuestion is the five faces as words, since chat has no pictures.
//
// "How do you feel?" — asked, and answered, about now. The reading goes stale
// after a few hours for that reason: a day is a thing you can have had a bad
// one of, and the useful answer is about the minute you are in.
//
// The five are not a scale and are never numbered. Low and frazzled are
// different states wanting different answers, which is exactly what a
// one-to-five row cannot say and the reason this is worth asking at all.
func CheckinQuestion() Message {
	actions := make([]Action, 0, len(Moods))
	for _, m := range Moods {
		actions = append(actions, Action{
			Label: Words[m],
			Value: "mood:" + string(m),
			Emoji: MoodEmoji[m],
		})
	}
	return Message{
		Text:          "How do you feel?",
		SelectionMode: "single",
		Actions:       actions,
	}
}

// MoodEmoji is the chat's version of the drawn faces. The drawings are the
// screen's; these are as close as a room of text gets.
var MoodEmoji = map[Mood]string{
	MoodGood:     "😄",
	MoodCalm:     "🙂",
	MoodLow:      "😔",
	MoodFrazzled: "😵",
	MoodWiped:    "😴",
}

// screenLine is the way in, when there is one to give.
func screenLine() string {
	if screenURL == "" {
		return ""
	}
	return "\n\nThe same pile, to look at: " + screenURL
}
