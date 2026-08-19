package squirrel

import (
	"fmt"
	"strings"
	"time"
)

// When a chore is worth asking about — which is not the same question as when
// it is due, and keeping them apart is the point.
//
// A chore has a rhythm: its clock starts when you do it. Adding a calendar
// anchor ("the first of the month") would turn that rhythm into a deadline,
// and a deadline is a thing you can be late for — which is the shape this
// product exists without. So none of what follows moves a chore's baseline. It
// only decides when interrupting you about it is worth doing.
//
// Missing a window is therefore not a miss. The chore is exactly as due as it
// was; the asking waits for the next one.

// DayPart is a coarse time of day. Coarse on purpose: a chore that wants the
// evening does not want 17:00, and minutes are a precision tax charged every
// time you make a chore.
type DayPart string

const (
	AnyPart   DayPart = ""
	Morning   DayPart = "morning"
	Afternoon DayPart = "afternoon"
	Evening   DayPart = "evening"
)

// The hours each part covers, as [from, to). Nothing asks between 22:00 and
// 06:00 in any part, because a chore is not worth waking up for.
var partHours = map[DayPart][2]int{
	Morning:   {6, 12},
	Afternoon: {12, 17},
	Evening:   {17, 22},
}

// PartWords is what each part is called, for the screen and the chat.
var PartWords = map[DayPart]string{
	AnyPart:   "any time",
	Morning:   "mornings",
	Afternoon: "afternoons",
	Evening:   "evenings",
}

// ParseDayPart reads one of the four, or reports that it is not one.
func ParseDayPart(s string) (DayPart, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "any", "any time", "anytime":
		return AnyPart, true
	case "morning", "mornings":
		return Morning, true
	case "afternoon", "afternoons":
		return Afternoon, true
	case "evening", "evenings", "night", "nights":
		return Evening, true
	}
	return "", false
}

// Days is which days of the week a chore may be raised on, as a bitmask with
// bit 0 for Sunday — the same numbering as time.Weekday and as Postgres's
// extract(dow), so nothing has to translate.
type Days uint8

// AnyDay is the zero value and means what it says: no preference.
const AnyDay Days = 0

// Weekdays is Monday to Friday, for "every weekday".
const Weekdays Days = 1<<time.Monday | 1<<time.Tuesday | 1<<time.Wednesday |
	1<<time.Thursday | 1<<time.Friday

// OnlyOn is a single day.
func OnlyOn(d time.Weekday) Days { return 1 << d }

// Has reports whether this day is one of them. AnyDay has every day.
func (d Days) Has(w time.Weekday) bool {
	return d == AnyDay || d&(1<<w) != 0
}

var weekdayNames = map[string]time.Weekday{
	"sunday": time.Sunday, "sundays": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mondays": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tuesdays": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wednesdays": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thursdays": time.Thursday, "thu": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fridays": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "saturdays": time.Saturday, "sat": time.Saturday,
}

// DaysWords says a day preference the way a person would.
func (d Days) Words() string {
	switch d {
	case AnyDay:
		return ""
	case Weekdays:
		return "weekdays"
	}
	names := []string{}
	for w := time.Sunday; w <= time.Saturday; w++ {
		if d&(1<<w) != 0 {
			names = append(names, strings.ToLower(w.String())+"s")
		}
	}
	return strings.Join(names, " and ")
}

// Asking is the pair, and it is what a chore carries beside its rhythm.
type Asking struct {
	Days Days
	Part DayPart
}

// Open reports whether now is a moment worth raising this chore in.
//
// Both halves are preferences rather than deadlines, so an empty one is
// permission rather than a restriction.
func (a Asking) Open(now time.Time) bool {
	if !a.Days.Has(now.Weekday()) {
		return false
	}
	if a.Part == AnyPart {
		return true
	}
	h := partHours[a.Part]
	return now.Hour() >= h[0] && now.Hour() < h[1]
}

// Words says the whole preference in a phrase, or nothing at all when there is
// no preference to say. Nothing is the common case and it should read as
// silence rather than as "any time, any day".
func (a Asking) Words() string {
	switch {
	case a.Days == AnyDay && a.Part == AnyPart:
		return ""
	case a.Part == AnyPart:
		return a.Days.Words()
	case a.Days == AnyDay:
		return PartWords[a.Part]
	}
	return fmt.Sprintf("%s, %s", a.Days.Words(), PartWords[a.Part])
}
