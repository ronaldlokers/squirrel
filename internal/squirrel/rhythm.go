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

// Usually is when you tend to do a thing. The two halves are claimed
// separately and either may be absent: a chore you do every day has no usual
// weekday and a chore you do at any hour has no usual part of it.
type Usually struct {
	Weekday time.Weekday
	OnADay  bool
	Part    DayPart
}

func (u Usually) Known() bool { return u.OnADay || u.Part != AnyPart }

func (u Usually) Words() string {
	switch {
	case u.OnADay && u.Part != AnyPart:
		return "you usually do this " + u.Weekday.String() + " " + string(u.Part)
	case u.OnADay:
		return "you usually do this on a " + u.Weekday.String()
	case u.Part != AnyPart:
		return "you usually do this in the " + string(u.Part)
	default:
		return ""
	}
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
	u := Usually{}
	if day, most := commonest(days); mostly(most, len(when)) {
		u.Weekday, u.OnADay = day, true
	}
	if part, most := commonestPart(parts); mostly(most, len(when)) {
		u.Part = part
	}
	return u
}

// mostly is what turns a tally into a claim. More than half, so a thing split
// evenly between Saturday and Sunday has no usual day rather than whichever
// one the tie-break reached for.
func mostly(count, of int) bool { return count*2 > of }

func commonest(counted map[time.Weekday]int) (time.Weekday, int) {
	best, most := time.Sunday, -1
	for day, n := range counted {
		if n > most || (n == most && day < best) {
			best, most = day, n
		}
	}
	return best, most
}

func commonestPart(counted map[DayPart]int) (DayPart, int) {
	best, most := AnyPart, -1
	for part, n := range counted {
		if n > most || (n == most && part < best) {
			best, most = part, n
		}
	}
	return best, most
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
	today := u.OnADay && u.Weekday == at.Weekday()

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
		// Nothing to say. Not "last done a week ago" — the rack has refused
		// to print that since it was built, because how long a thing has been
		// waiting is a fact about you and its rhythm is a fact about it. What
		// a row here carries instead is when you usually do it, which the
		// screen draws from Usually.
		return 5, ""
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
		if u := UsuallyFrom(times); u.Known() {
			out[choreID] = u
		}
	}
	return out, nil
}

// Whichever rack they sit in, these are the ones with something to say for
// themselves today: it comes back today, or today is the day you usually do
// it. Rank 5 and 6 are the rest of your life and are not that.
const standingNow = 4

// Now is the racks flattened back into one order, for a screen too small to
// show three of them. Same ranks, same reasons, so the phone and the desk
// cannot disagree about what is in front of you.
func Now(racks []Rack) []Standing {
	out := []Standing{}
	for _, rack := range racks {
		for _, s := range rack.Waiting {
			if s.rank <= standingNow {
				out = append(out, s)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].rank != out[j].rank {
			return out[i].rank < out[j].rank
		}
		return overdueBy(out[i].Chore) > overdueBy(out[j].Chore)
	})
	return out
}
