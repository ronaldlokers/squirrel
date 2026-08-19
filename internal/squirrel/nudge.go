package squirrel

import "fmt"

// PickChore chooses the one chore a nudge will name.
//
// Weighted random, not simply the most overdue, for two reasons. The most
// overdue chore is the one you have been avoiding longest, which usually means
// it is the aversive or vague or large one — so naming it every day leads with
// the thing you least want to see. And more sharply: it is by definition the
// one nudging has already failed to shift, so a fourth week of naming it is not
// the intervention.
//
// Weighting by how overdue keeps the urgent thing usually surfacing while
// leaving the nudge unpredictable, which is the same mechanism that fights
// habituation.
//
// draw comes from the caller so this stays a pure function with deterministic
// tests; production passes rand.Float64().
func PickChore(due []Chore, draw float64) (Chore, bool) {
	if len(due) == 0 {
		return Chore{}, false
	}

	weights := make([]float64, len(due))
	var total float64
	for i, c := range due {
		// A chore exactly at its interval weighs 1, one at three intervals
		// weighs 3. The floor of 1 matters: a chore that is due but not yet
		// past its interval would otherwise weigh 0 and never be reachable.
		w := 1.0
		if c.EveryDays > 0 && c.SinceDays > c.EveryDays {
			w = float64(c.SinceDays) / float64(c.EveryDays)
		}
		weights[i] = w
		total += w
	}

	target := draw * total
	var running float64
	for i, w := range weights {
		running += w
		if target < running {
			return due[i], true
		}
	}
	// Only reachable on a draw of exactly 1.0 or floating-point drift.
	return due[len(due)-1], true
}

// NudgeReason is which trigger produced this nudge. It changes only the
// framing — varied phrasing is the same anti-habituation mechanism as varied
// timing, and it costs nothing.
//
// Two values, not three: a nudge only ever reaches a real send by riding back
// on a message (NudgeFromMessage) or off the presence webhook
// (NudgeFromArrival). The quiet-day fallback is EveningMessage, which builds
// its own line from choreSentence and never calls NudgeMessage — so there is
// no third trigger here to give words to.
type NudgeReason string

const (
	NudgeFromMessage NudgeReason = "message"
	NudgeFromArrival NudgeReason = "arrival"
)

// NudgeMessage is one chore and two buttons. Never a list: self-regulation
// draws on a depletable pool and every decision spends it, so six due chores
// is six decisions charged to the resource that is already short.
//
// Two buttons and not one, because the nudge arrives at the moment you are
// least able to decide anything, and until now the only thing it could hear
// was yes. "Not today" is the answer that was missing — !snooze existed, but
// it wanted a command typed at exactly the wrong moment. The chore's clock
// keeps running while it is quiet, so nothing about when it is next due
// changes; it is only the asking that stops.
//
// Tapping it again takes it back, the same way an unselected done retracts a
// completion. Nothing here is a decision you cannot reverse.
func NudgeMessage(c Chore, why NudgeReason) Message {
	text := choreSentence(c)
	if why == NudgeFromMessage {
		text = fmt.Sprintf("While you're here — %s.", ChoreWords(c))
	}

	return Message{
		Text:          text,
		SelectionMode: "single",
		Actions: []Action{{
			Label: c.Name,
			Value: "done:1",
			Emoji: "✅",
		}, {
			Label: "not today",
			Value: "snooze:1",
			Emoji: "🌙",
		}},
	}
}
