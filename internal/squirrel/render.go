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
		"!now — one thing, chosen. !now anyway ignores a low day",
		"!stuck — I can't start. Four answers, and one of them helps",
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
func EveningMessage(handled Handled, captures []string, nudge *Chore) Message {
	var b strings.Builder

	if nudge != nil {
		b.WriteString(choreSentence(*nudge))
		b.WriteString("\n")
	}
	if lines := handledLines(handled); len(lines) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Today\n")
		for _, line := range lines {
			fmt.Fprintf(&b, " · %s\n", line)
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

// NowMessage is the one thing, in chat.
//
// One line and never a list — the same discipline the nudge keeps, and for the
// same reason: six things due is six decisions charged to the resource that is
// already short. The clause is on its own line rather than in brackets,
// because it is the answer to "why this" and that question deserves a sentence
// rather than an aside.
//
// Two buttons, matching the nudge's shape exactly. Anything that can be
// answered has to be answerable the same way everywhere, or the two surfaces
// have grown two vocabularies.
func NowMessage(o Offer) Message {
	m := Message{Text: fmt.Sprintf("%s\n%s.", o.Text, o.Because)}
	// A timer names no row, so there is nothing for a button to resolve
	// against — and nothing to press, either. You are already doing it.
	if o.Kind == OfferTimer {
		return m
	}
	// A breadcrumb names a label rather than a row, so it cannot be marked
	// done: Squirrel does not know what that label was. Picking it back up is
	// the whole of what it can offer, and in chat that is one line naming the
	// command rather than a button that would have to resolve against nothing.
	if o.Kind == OfferAgain {
		m.Text += fmt.Sprintf("\n!timer 10 %s to pick it up.", o.Text)
		return m
	}
	m.SelectionMode = "single"
	m.Actions = []Action{
		{Label: doneWord(o), Value: "done:1", Emoji: "✅"},
		{Label: "not now", Value: "later:1", Emoji: "🌙"},
	}
	return m
}

// doneWord is what the completing button says. A chore is named — the nudge
// already puts a chore's own name on its button and the two must agree — and a
// task says what was done to it, because a task's text is a whole sentence and
// a sentence on a button is unreadable.
func doneWord(o Offer) string {
	if o.Kind == OfferChore {
		return o.Text
	}
	return "did it"
}

// StuckQuestion asks what is in the way — four answers, one line, no
// follow-up.
//
// It is asked once and never twice. A product that answers "I can't start"
// with a second question has charged the person another decision at the moment
// they said they had none left.
func StuckQuestion() Message {
	words := make([]string, 0, len(Blockers))
	for _, b := range Blockers {
		words = append(words, BlockerWords[b])
	}
	return Message{Text: "What is in the way?\n" + strings.Join(words, " · ") +
		"\n\nSay !stuck and one of those."}
}

// StuckMessage is the answer, and it never grows into a plan.
func StuckMessage(u Unstuck, subject string) Message {
	if u.Ask {
		return Message{Text: u.Line + "\nTell me and I will keep it."}
	}
	m := Message{Text: u.Line}
	if u.Minutes > 0 && subject != "" {
		m.Text += fmt.Sprintf("\n!timer %d %s when you are ready.", u.Minutes, subject)
	}
	return m
}

// NothingNowMessage is what the picker says when it has nothing.
//
// Stated plainly, and nothing is suggested. Having nothing to be handed is a
// normal state rather than a setup failure, and a sentence encouraging you to
// go and find something would be the product deciding you ought to be busy.
func NothingNowMessage(capacity Capacity) Message {
	if capacity == CapacityLow {
		return Message{Text: "Nothing from me today. Say !now anyway if you want something."}
	}
	return Message{Text: "Nothing to hand you."}
}

// handledLines is what happened today, as lines, and nothing when nothing did.
//
// Chores and tasks are named because they are the things you set out to do.
// Notes are counted because naming a dozen cleared notes would bury the two
// lines above them in the bookkeeping — and because what matters about a
// cleared note is that it is no longer waiting, not what it said.
//
// Never "nothing today". An absent section says nothing about you; an empty
// one is a scoreboard reading nil.
func handledLines(h Handled) []string {
	lines := make([]string, 0, len(h.Chores)+len(h.Tasks)+1)
	lines = append(lines, h.Chores...)
	lines = append(lines, h.Tasks...)
	if h.Notes > 0 {
		lines = append(lines, plural(h.Notes, "note")+" cleared")
	}
	return lines
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
