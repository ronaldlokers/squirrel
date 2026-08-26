package squirrel

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// NotesMessage renders a numbered list of notes — the pile, or search results.
//
// `more` is a bool rather than a count: the number never reaches this function,
// so it could not render a total.
//
// No buttons. The pile is answered by typing a number, and a twenty-line list
// cannot carry a button per line — actions are capped at twelve.
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

// TasksMessage is what you decided, numbered so `done 2` can name one. The
// numbers point at a task; the list is capped and says "…and more" rather than
// reporting a total.
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

// HelpMessage is the vocabulary. The capture rule comes first: if you remember
// nothing else, typing a thought stores the thought.
var screenURL string

// SetScreenURL is called once at boot.
func SetScreenURL(u string) { screenURL = u }

// coachHere is whether a model is reachable. Help does not list `!buddy` when it
// is not. Package-level and set once at boot, like screenURL.
var coachHere bool

// stuckHelp is named because the coach's line is inserted directly after it,
// and a help list that quietly stops matching itself is how a line ends up in
// the wrong place.
const stuckHelp = "!stuck — I can't start. Four answers, and one of them helps"

// SetCoachHere is called once at boot.
func SetCoachHere(here bool) { coachHere = here }

func HelpMessage() Message {
	lines := []string{
		"Anything you type is a note. That is the default and it always wins.",
		"",
		"!now — one thing, chosen. !now anyway ignores a low day",
		stuckHelp,
		"at 14:30 dentist, 20 minutes away — a time the world imposed",
		"!bring keys, wallet — what to take to it",
		"!leaving — you went, or it is off",
		"!notes — the pile, newest first",
		"!find <text> — search everything you have told me",
		"!chores — what is due (same as ?)",
		"!chore <n> every <interval> — turn note n into a chore",
		"!task <n> — a note is something you decided to do",
		"!task <words> — decide something outright",
		"!tasks — what you decided, newest first",
		"!untask <n> — back in the pile, undecided",
		"!waiting <n> on <who> · !blocked <n> on <what> · !someday <n>",
		"!waiting — everything you set aside, and what would move it",
		"every other tuesday: bins out — a rhythm, and when to raise it",
		"!moods — what you said before, when you ask. Nothing else reads it",
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
	}

	if coachHere {
		// Next to !stuck, because it is the same moment reached a different
		// way: the ladder when you can name what is in the way, this when you
		// cannot.
		lines = slices.Insert(lines, slices.Index(lines, stuckHelp)+1,
			"!buddy <words> — say what is going on, in your own words",
			"!next — the step after the one you just did")
	}

	return Message{Text: strings.Join(lines, "\n") + screenLine()}
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

// EveningMessage is the once-a-day message: what you did, what you captured, and
// — only when no nudge fired earlier — the chore that would otherwise arrive as a
// second notification.
//
// When nothing was completed the section is absent rather than empty.
func EveningMessage(handled Handled, captures []string, nudge *Chore, kept string) Message {
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

	// One kept note, sometimes, riding along with a message that was going out
	// anyway. Never its own message and never more than one. Last, and unprompted:
	// nothing is being asked of you.
	if kept != "" && b.Len() > 0 {
		fmt.Fprintf(&b, "\nYou kept this: %s\n", kept)
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

// NowMessage is the one thing, in chat. One line and never a list. The clause is
// on its own line because it answers "why this".
//
// Two buttons, matching the nudge's shape exactly.
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

// MomentKeptMessage answers with the leaving time rather than the start: the
// start is what you already knew. The offer to say what to take rides along.
func MomentKeptMessage(m Moment) Message {
	return Message{Text: fmt.Sprintf("%s %s.\nI will say something at %s.\n!bring keys, wallet if there is something to take.",
		m.Label, LeaveWords(m), m.WarnAt().Format("15:04"))}
}

// LeaveMessage is the one thing a fixed point says, at the moment it matters.
//
// No buttons, structurally: a numbered line points at a chore or an item and the
// database enforces exactly one of the two. `!leaving` says the same in a word.
//
// There is deliberately no "in five minutes": a fixed point cannot be moved by
// pressing something.
func LeaveMessage(m Moment) Message {
	text := fmt.Sprintf("%s %s.", m.Label, LeaveWords(m))
	if m.Bring != "" {
		text += "\nTake: " + m.Bring
	}
	return Message{Text: text + "\n!leaving when you go."}
}

// StuckQuestion asks what is in the way — four answers, one line, no follow-up.
// Asked once and never twice.
func StuckQuestion() Message {
	words := make([]string, 0, len(Blockers))
	for _, b := range Blockers {
		words = append(words, BlockerWords[b])
	}
	return Message{Text: "What is in the way?\n" + strings.Join(words, " · ") +
		"\n\nSay !stuck and one of those."}
}

// StepMessage is one step, never the sequence, and says nothing about how many
// there are. On the last one it says so, because being left waiting for a step
// that never comes is its own failure.
func StepMessage(st Step) Message {
	if st.Last {
		return Message{Text: st.Body + "\nThat is the last one. Say !next when it is done."}
	}
	return Message{Text: st.Body + "\nSay !next when it is done."}
}

// StepsFinishedMessage is the end of a sequence.
//
// Nothing is celebrated and nothing is totalled. Finishing a breakdown is a
// normal ending, and a reward here would be a counter wearing a different hat.
func StepsFinishedMessage(label string) Message {
	if label == "" {
		return Message{Text: "That is all of them."}
	}
	return Message{Text: "That is all of them for " + label + "."}
}

// HeldMessage says what was set aside and what would bring it back.
//
// One line, and no encouragement. Setting something aside is an ordinary thing
// to do with a thing you cannot do — not a failure to be softened, and not a
// decision to be congratulated.
func HeldMessage(h HeldItem) Message {
	return Message{Text: h.Text + " — " + h.Words() + "."}
}

// HeldListMessage is everything you set aside, grouped by which of the three:
// waiting has somebody to chase, blocked has something to arrive, someday has
// neither.
//
// No count in either direction. `more` says the list was capped and never by how
// much.
func HeldListMessage(held []HeldItem, more bool) Message {
	if len(held) == 0 {
		return Message{Text: "Nothing set aside."}
	}

	lines := make([]string, 0, len(held)+len(Held))
	for _, state := range Held {
		var group []string
		for _, h := range held {
			if h.State != state {
				continue
			}
			line := h.Text
			if h.Because != "" {
				line += " — " + h.Because
			}
			group = append(group, line)
		}
		if len(group) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.ToUpper(HeldWords[state]))
		lines = append(lines, group...)
	}

	if more {
		lines = append(lines, "", "There is more.")
	}
	return Message{Text: strings.Join(lines, "\n")}
}

// MoodsMessage is the readings and their days, and nothing else. No average, no
// streak. Grouped by day: two answers on one Tuesday is a Tuesday you checked in
// twice.
func MoodsMessage(readings []Checkin, now time.Time) Message {
	if len(readings) == 0 {
		return Message{Text: "You have not said how you are lately."}
	}

	var b strings.Builder
	day := ""
	for _, c := range readings {
		if d := moodDay(c.SaidAt, now); d != day {
			if day != "" {
				b.WriteString("\n")
			}
			b.WriteString(d)
			b.WriteString(": ")
			day = d
		} else {
			b.WriteString(", ")
		}
		b.WriteString(Words[c.Mood])
	}
	return Message{Text: b.String()}
}

// MoodDay is moodDay, for the screen. Both surfaces name a day identically or
// they are two products.
func MoodDay(at, now time.Time) string { return moodDay(at, now) }

// moodDay names a day the way you would say it. Two names and then the date:
// past "yesterday" a weekday alone is ambiguous within a fortnight.
func moodDay(at, now time.Time) string {
	today := startOfDay(now)
	switch startOfDay(at) {
	case today:
		return "today"
	case today.AddDate(0, 0, -1):
		return "yesterday"
	}
	return strings.ToLower(at.Format("Monday 2 January"))
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

// handledLines is what happened today. Chores and tasks are named; notes are
// counted, because naming a dozen cleared notes buries the lines above them.
//
// Never "nothing today": an absent section says nothing about you.
func handledLines(h Handled) []string {
	lines := make([]string, 0, len(h.Chores)+len(h.Tasks)+1)
	lines = append(lines, h.Chores...)
	lines = append(lines, h.Tasks...)
	if h.Notes > 0 {
		lines = append(lines, plural(h.Notes, "note")+" cleared")
	}
	return lines
}

// CheckinQuestion is the five faces as words, since chat has no pictures. Asked
// about now, which is why the reading goes stale after a few hours.
//
// The five are not a scale and are never numbered.
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
// OnTheScreen is what chat says when Buddy suggested something chat cannot offer
// a press for. Chat's buttons resolve against recorded lines and cannot carry a
// proposal's four fields, so the honest answer names the place that can.
//
// Empty with no screen configured, which removes the sentence.
func OnTheScreen() string {
	if screenURL == "" {
		return ""
	}
	return " There is something to say yes to on the screen: " + screenURL
}

func screenLine() string {
	if screenURL == "" {
		return ""
	}
	return "\n\nThe same pile, to look at: " + screenURL
}
