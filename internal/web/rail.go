package web

import (
	"sort"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// railView is the phone's whole screen: today, in the order today happens.
//
// Head is the one thing that wants you now, drawn whole. Hung is everything
// else today, in time order. Ahead is a fixed point that is not today. Folded
// is what has no hour and no claim on today — the things you decided to do
// once, and the notes — each a single row carrying a count.
type railView struct {
	Head   *stripView
	Hung   []stripView
	Ahead  []apptView
	Folded []foldView
}

// foldView is one of the two rows off the rail. A count and a way in, and
// nothing else: what is behind them is not today's business.
type foldView struct {
	Key   string
	Words string
	Count int
	Href  string
	Rows  []stripView
}

func railFor(racks []rackView, coming *comingView, byID map[int64]squirrel.Usually, notes int) *railView {
	rail := &railView{}

	type placed struct {
		row  stripView
		hour int
	}
	var hung []placed
	var once []stripView
	onceCount := 0
	for _, rack := range racks {
		if rack.Key == "now" {
			continue
		}
		if rack.Key == "once" {
			once = rack.Strips
			onceCount = len(once)
			continue
		}
		for _, row := range rack.Strips {
			if !row.Wants {
				continue
			}
			hour, ok := squirrel.PartStarts(byID[row.ID].Part)
			if !ok {
				// No hour to hang at, but it still wants you today, so it
				// hangs at the end of the day rather than being folded away.
				hung = append(hung, placed{row, 24})
				continue
			}
			row.When = squirrel.PartWords[byID[row.ID].Part]
			hung = append(hung, placed{row, hour})
		}
	}

	if coming != nil {
		for _, appt := range coming.Appts {
			if appt.Day != "" {
				rail.Ahead = append(rail.Ahead, appt)
				continue
			}
			hung = append(hung, placed{atRow(appt), hourOf(appt.Time)})
		}
	}

	sort.SliceStable(hung, func(i, j int) bool { return hung[i].hour < hung[j].hour })

	if coming != nil && coming.Hoisted != nil {
		lifted := atRow(*coming.Hoisted)
		rail.Head = &lifted
	}
	for _, one := range hung {
		row := one.row
		if rail.Head == nil {
			rail.Head = &row
			continue
		}
		rail.Hung = append(rail.Hung, row)
	}

	rail.Folded = []foldView{
		{Key: "once", Words: "things you decided", Count: onceCount, Rows: once},
		{Key: "notes", Words: "in the notes", Count: notes, Href: "/notes"},
	}
	return rail
}

func atRow(appt apptView) stripView {
	return stripView{
		ID: appt.ID, What: "at", Words: appt.Label, When: appt.Time,
		Why: appt.LeaveBy, Mark: appt.Repeats, Late: appt.Late, Wants: true,
	}
}

func hourOf(clock string) int {
	if len(clock) < 2 {
		return 0
	}
	h := int(clock[0]-'0')*10 + int(clock[1]-'0')
	if h < 0 || h > 23 {
		return 0
	}
	return h
}
