package squirrel

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
