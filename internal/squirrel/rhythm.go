package squirrel

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type Rhythm string

const (
	Daily  Rhythm = "daily"
	Weekly Rhythm = "weekly"
	Seldom Rhythm = "seldom"
)

const (
	dailyUpTo  = 2
	weeklyUpTo = 14
)

func RhythmOf(everyDays int) Rhythm {
	switch {
	case everyDays <= dailyUpTo:
		return Daily
	case everyDays <= weeklyUpTo:
		return Weekly
	default:
		return Seldom
	}
}

const fewestUsually = 3

type Usually struct {
	Weekday time.Weekday
	Part    DayPart
	Known   bool
}

func (u Usually) Words() string {
	if !u.Known {
		return ""
	}
	if u.Part == AnyPart {
		return "you usually do this on a " + u.Weekday.String()
	}
	return "you usually do this " + u.Weekday.String() + " " + string(u.Part)
}

func UsuallyFrom(when []time.Time) Usually {
	if len(when) < fewestUsually {
		return Usually{}
	}
	days := map[time.Weekday]int{}
	parts := map[DayPart]int{}
	for _, at := range when {
		days[at.Weekday()]++
		parts[PartOfDay(at)]++
	}
	return Usually{Weekday: commonest(days), Part: commonestPart(parts), Known: true}
}

func commonest(counted map[time.Weekday]int) time.Weekday {
	best, most := time.Sunday, -1
	for day, n := range counted {
		if n > most || (n == most && day < best) {
			best, most = day, n
		}
	}
	return best
}

func commonestPart(counted map[DayPart]int) DayPart {
	best, most := AnyPart, -1
	for part, n := range counted {
		if n > most || (n == most && part < best) {
			best, most = part, n
		}
	}
	return best
}

type Standing struct {
	Chore   Chore
	Usually Usually
	Because string
	rank    int
}

type Rack struct {
	Rhythm  Rhythm
	Waiting []Standing
	Resting int
}

func due(c Chore) bool { return c.SinceDays >= c.EveryDays }

func partHasCome(part DayPart, at time.Time) bool {
	hours, named := partHours[part]
	return named && at.Hour() >= hours[0]
}

func rankOf(c Chore, u Usually, at time.Time) (int, string) {
	if !c.EverDone {
		return 6, choreBecause(c)
	}
	today := u.Known && u.Weekday == at.Weekday()

	switch {
	case due(c) && today && partHasCome(u.Part, at):
		return 1, "it comes back today, and this is when you usually do it"
	case due(c) && today:
		return 2, "it comes back today, and you usually do it later on"
	case due(c):
		return 3, "it comes back today"
	case today:
		return 4, "not yet, but this is the day you usually do it"
	default:
		return 5, choreBecause(c)
	}
}

func RacksOf(chores []Chore, usually map[int64]Usually, at time.Time, quiet bool) []Rack {
	by := map[Rhythm][]Standing{}
	resting := map[Rhythm]int{}

	for _, c := range chores {
		u := usually[c.ID]
		rank, because := rankOf(c, u, at)
		rhythm := RhythmOf(c.EveryDays)
		if quiet && !due(c) {
			resting[rhythm]++
			continue
		}
		by[rhythm] = append(by[rhythm], Standing{Chore: c, Usually: u, Because: because, rank: rank})
	}

	out := make([]Rack, 0, 3)
	for _, rhythm := range []Rhythm{Daily, Weekly, Seldom} {
		standing := by[rhythm]
		sort.SliceStable(standing, func(i, j int) bool {
			if standing[i].rank != standing[j].rank {
				return standing[i].rank < standing[j].rank
			}
			return overdueBy(standing[i].Chore) > overdueBy(standing[j].Chore)
		})
		out = append(out, Rack{Rhythm: rhythm, Waiting: standing, Resting: resting[rhythm]})
	}
	return out
}

func overdueBy(c Chore) float64 {
	if c.EveryDays <= 0 {
		return 0
	}
	return float64(c.SinceDays) / float64(c.EveryDays)
}

const usuallyReadsBack = 12

func (s *Store) WhenYouUsuallyDo(ctx context.Context, personID int64) (map[int64]Usually, error) {
	rows, err := s.pool.Query(ctx, `
		select chore_id, occurred_at from (
		    select e.chore_id, e.occurred_at,
		           row_number() over (partition by e.chore_id order by e.occurred_at desc) as recency
		      from events e join chores c on c.id = e.chore_id
		     where c.person_id = $1 and c.active and e.retracted_at is null
		  ) recent
		 where recency <= $2`, personID, usuallyReadsBack)
	if err != nil {
		return nil, fmt.Errorf("reading when you usually do things: %w", err)
	}
	defer rows.Close()

	when := map[int64][]time.Time{}
	for rows.Next() {
		var choreID int64
		var at time.Time
		if err := rows.Scan(&choreID, &at); err != nil {
			return nil, fmt.Errorf("reading when you usually do things: %w", err)
		}
		when[choreID] = append(when[choreID], s.here(at))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading when you usually do things: %w", err)
	}

	out := make(map[int64]Usually, len(when))
	for choreID, times := range when {
		if u := UsuallyFrom(times); u.Known {
			out[choreID] = u
		}
	}
	return out, nil
}
