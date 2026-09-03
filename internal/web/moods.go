package web

import (
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// How you have been, on the screen.
//
// This table was unreadable by construction until 20 August 2026 — the store
// returned one reading and no function could return more, which is stronger
// than a rule somebody has to remember. What replaced it is narrower: it is
// drawn in one place, on the page about you, and nothing else reads the
// readings at all.

// howYouFeltBefore is what the way back to the readings is called, beside the
// answer you just gave.
const howYouFeltBefore = "how you felt before"

// moodWeeks lays the readings out as six weeks by seven days.
//
// A grid rather than a list because the gaps are the honest part. You check in
// on some days and not others, and days you said nothing are most of what is
// there — a list of the days you answered hides that by only ever showing the
// days you answered, and a bar chart hides it more thoroughly still. Here they
// are drawn: an outline with nothing in it.
//
// Nothing is counted, averaged, compared or trended. There is no number on
// this page. You read it; the product does not.
//
// A day you answered more than once shows the last thing you said, because
// what stands at the end of a day is what the day came to. The earlier answers
// are still in the table and this page is not the place that reports them.
//
// Days after today are drawn as nothing at all, not as gaps: you have not
// failed to check in on Friday yet.
func moodWeeks(readings []squirrel.Checkin, at time.Time) []moodWeekView {
	// Nothing said at all draws no grid. Six weeks of empty outlines is a
	// picture of forty-two days you did not check in, which is the one
	// judgement this page could still make by accident.
	if len(readings) == 0 {
		return nil
	}

	said := map[string]squirrel.Checkin{}
	for _, c := range readings {
		key := c.SaidAt.Format("2006-01-02")
		// Newest first from the store, so the first one seen for a day is the
		// last one said on it.
		if _, seen := said[key]; !seen {
			said[key] = c
		}
	}

	start, today := squirrel.MoodCalendarStart(at), at.Format("2006-01-02")
	weeks := make([]moodWeekView, 0, squirrel.MoodCalendarWeeks)
	for w := 0; w < squirrel.MoodCalendarWeeks; w++ {
		monday := start.AddDate(0, 0, 7*w)
		week := moodWeekView{Week: weekNamed(monday, at), Days: make([]moodCellView, 0, 7)}
		for d := 0; d < 7; d++ {
			day := monday.AddDate(0, 0, d)
			key := day.Format("2006-01-02")
			cell := moodCellView{
				Day:    strings.ToLower(day.Format("Monday 2 January")),
				Ahead:  key > today,
				Nought: true,
			}
			if c, ok := said[key]; ok {
				cell.Mood, cell.Word, cell.Nought = string(c.Mood), squirrel.Words[c.Mood], false
			}
			week.Days = append(week.Days, cell)
		}
		weeks = append(weeks, week)
	}
	return weeks
}

// weekNamed labels a row. The week you are in is named rather than dated,
// because "this week" is where you are and a date makes you work that out.
func weekNamed(monday, at time.Time) string {
	if !monday.After(at) && monday.AddDate(0, 0, 7).After(at) {
		return "this week"
	}
	return strings.ToLower(monday.Format("2 Jan"))
}
