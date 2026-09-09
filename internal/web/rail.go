package web

import (
	"sort"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type railView struct {
	Head  *stripView
	Hung  []stripView
	Off   []stripView
	Ahead []apptView
}

func railFor(racks []rackView, coming *comingView, byID map[int64]squirrel.Usually) *railView {
	rail := &railView{}

	type placed struct {
		row  stripView
		hour int
	}
	var hung []placed
	for _, rack := range racks {
		if rack.Key == "now" {
			continue
		}
		for _, row := range rack.Strips {
			hour, ok := squirrel.PartStarts(byID[row.ID].Part)
			if rack.Key == "once" || !row.Wants || !ok {
				rail.Off = append(rail.Off, row)
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
	if rail.Head == nil && len(rail.Off) > 0 {
		rail.Head, rail.Off = &rail.Off[0], rail.Off[1:]
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
