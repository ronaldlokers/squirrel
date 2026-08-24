package squirrel

import (
	"context"
	"fmt"
	"time"
)

// Waiting is the number on each door.
//
// Principle 2 — nothing accrues that can be destroyed — was retired by the
// owner on 24 August 2026, and this is the first thing built on the retirement.
// What the rule protected against arrives in exactly this shape: a number
// beside a door with an implied target of zero, that grows while nobody is
// looking. The decision reverses cleanly because nothing here is stored —
// remove the call and the numbers are gone.
//
// Each field is what is *waiting for you*, not what the door holds, so an empty
// door reads as finished rather than as absent.
type Waiting struct {
	// Pile is notes not yet decided about, under exactly the pile's own rule.
	Pile int
	// Tasks is what you decided and have not done.
	Tasks int
	// Chores is what is due right now — not how many chores you keep. A number
	// that never drops is a number that has stopped saying anything.
	Chores int
	// Agenda is fixed points still ahead today. Not this week: a door counting
	// next Tuesday is a door you cannot empty.
	Agenda int
}

// Waiting reads all four.
//
// Three are count queries. Chores are counted by asking DueChores and taking
// the length, because due-ness is a CTE and a tolerance gate against the last
// delivered digest (see DueChores) — a count query would be a second definition
// of "due" that drifts from the first. Chores are few, so this is cheap and it
// cannot disagree.
func (s *Store) Waiting(ctx context.Context, personID int64, now time.Time) (Waiting, error) {
	var w Waiting

	const q = `
		select
		  (select count(*) from items
		    where person_id = $1 and kind = 'note' and state = 'open'
		      and ` + stillMine + `),
		  (select count(*) from items
		    where person_id = $1 and kind = 'task' and state = 'open'),
		  (select count(*) from moments
		    where person_id = $1 and done_at is null
		      and starts_at > $2 and starts_at < $3)`

	endOfDay := StartOfDay(now).Add(24 * time.Hour)
	if err := s.pool.QueryRow(ctx, q, personID, now, endOfDay).
		Scan(&w.Pile, &w.Tasks, &w.Agenda); err != nil {
		return Waiting{}, fmt.Errorf("reading what is waiting: %w", err)
	}

	due, err := s.DueChores(ctx, personID, now)
	if err != nil {
		return Waiting{}, fmt.Errorf("reading what is waiting: %w", err)
	}
	w.Chores = len(due)

	return w, nil
}
