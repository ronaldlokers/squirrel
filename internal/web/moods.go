package web

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// How you have been, on the screen.
//
// This table was unreadable by construction until 20 August 2026 — the store
// returned one reading and no function could return more, which is stronger
// than a rule somebody has to remember. What replaced it was narrower: it was
// drawn in one place, on the page about you.
//
// Two places since 8 September 2026: six weeks there, and the week behind today
// on the board's dial. Both are drawn from the same readings by the same
// per-day rule, so the two cannot disagree about what a day was.

// howYouFeltBefore is what the way back to the readings is called, beside the
// answer you just gave.
const howYouFeltBefore = "how you felt before"

// latestPerDay is one reading per day, keyed by the date. What stands at the
// end of a day is what the day came to, so it is the last thing said on it —
// and the store hands them back newest first, which makes the first one seen
// for a day the one to keep.
func latestPerDay(readings []squirrel.Checkin) map[string]squirrel.Checkin {
	said := map[string]squirrel.Checkin{}
	for _, c := range readings {
		key := c.SaidAt.Format("2006-01-02")
		if _, seen := said[key]; !seen {
			said[key] = c
		}
	}
	return said
}

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

	said := latestPerDay(readings)

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

// dialDays is how far the dial looks back: the week behind today, which is the
// span over which "how have I been" is a question about now rather than about
// your year. The page about you still holds six weeks.
const dialDays = 7

// moodDaysBefore is the last seven days ending today, oldest first, for the
// dial on the board.
//
// Same cells as the grid on the page about you and drawn from the same
// readings, so the two cannot disagree about what a day was. A day you said
// nothing is drawn as nothing said, because the gaps are the honest part here
// exactly as they are there.
func moodDaysBefore(readings []squirrel.Checkin, at time.Time) []moodCellView {
	said := latestPerDay(readings)
	days := make([]moodCellView, 0, dialDays)
	for back := dialDays - 1; back >= 0; back-- {
		day := at.AddDate(0, 0, -back)
		cell := moodCellView{
			Day:    strings.ToLower(day.Format("Monday 2 January")),
			Short:  strings.ToUpper(day.Format("Mon")),
			Nought: true,
		}
		if c, ok := said[day.Format("2006-01-02")]; ok {
			cell.Mood, cell.Word, cell.Nought = string(c.Mood), squirrel.Words[c.Mood], false
		}
		days = append(days, cell)
	}
	return days
}

// The dial's ring: the seven days as seven arcs around today's face.
//
// A ring rather than the row of squares on the page about you, because these
// seven are a cycle you are inside rather than a stretch of past you are
// looking back over. Today is the last arc, and where it sits moves with the
// week.
const (
	dialRadius = 44
	dialGap    = 6.5
)

type ringSegView struct {
	Dash   string
	Offset string
	Class  string
	Label  string
	X, Y   float64
	Letter string
}

// moodRing lays the same seven cells out as arcs. The geometry is worked out
// here and not in the template: a template that can do trigonometry is a
// template nobody can read.
func moodRing(days []moodCellView) []ringSegView {
	if len(days) == 0 {
		return nil
	}
	round := 2 * math.Pi * dialRadius
	step := round / float64(len(days))

	out := make([]ringSegView, 0, len(days))
	for i, day := range days {
		// -90° puts the first arc at the top, and half a step further puts the
		// letter in the middle of its own arc rather than on the seam.
		mid := -math.Pi/2 + (float64(i)+0.5)*(2*math.Pi/float64(len(days)))
		seg := ringSegView{
			Dash:   fmt.Sprintf("%.2f %.2f", step-dialGap, round-step+dialGap),
			Offset: fmt.Sprintf("%.2f", -float64(i)*step),
			Class:  "nought",
			Label:  day.Day + ", nothing said",
			X:      math.Round((60+(dialRadius+13)*math.Cos(mid))*100) / 100,
			Y:      math.Round((60+(dialRadius+13)*math.Sin(mid))*100) / 100,
			Letter: day.Short,
		}
		if !day.Nought {
			seg.Class, seg.Label = "m"+day.Mood, day.Day+", "+day.Word
		}
		out = append(out, seg)
	}
	return out
}
