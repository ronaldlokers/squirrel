package squirrel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Chore struct {
	ID        int64
	PersonID  int64
	Name      string
	Every     time.Duration
	Tolerance time.Duration
	Active    bool
	// SinceDays is days since the baseline — the last completion, or creation
	// if there has never been one. EveryDays is the interval. Both are what the
	// renderer prints and neither is stored.
	SinceDays int
	EveryDays int
	// EverDone says whether the baseline is a completion or merely the
	// chore's own birthday, which the screen needs in order to keep quiet:
	// "last done 3 weeks ago" about a chore nobody has ever done is a
	// sentence about the person, not about the chore.
	EverDone bool
}

// DefaultTolerance is used when a definition does not carry one: a quarter of
// the interval, never less than a day. A weekly chore then reappears every
// other day once due; a quarterly one, every three weeks.
func DefaultTolerance(every time.Duration) time.Duration {
	t := every / 4
	if t < 24*time.Hour {
		return 24 * time.Hour
	}
	return t
}

func (s *Store) UpsertChore(ctx context.Context, personID int64, name string, every, tolerance time.Duration) (Chore, error) {
	const q = `
		insert into chores (person_id, name, interval_seconds, tolerance_seconds)
		values ($1, $2, $3, $4)
		on conflict (person_id, lower(name)) do update
		  set interval_seconds = excluded.interval_seconds,
		      tolerance_seconds = excluded.tolerance_seconds,
		      active = true,
		      updated_at = now()
		returning id, person_id, name, interval_seconds, tolerance_seconds, active`

	var c Chore
	var everySec, tolSec int64
	err := s.pool.QueryRow(ctx, q, personID, name,
		int64(every.Seconds()), int64(tolerance.Seconds()),
	).Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec, &c.Active)
	if err != nil {
		return Chore{}, fmt.Errorf("upserting chore %q: %w", name, err)
	}
	c.Every = time.Duration(everySec) * time.Second
	c.Tolerance = time.Duration(tolSec) * time.Second
	c.EveryDays = int(c.Every.Hours() / 24)
	return c, nil
}

func (s *Store) DeactivateChore(ctx context.Context, choreID int64) error {
	_, err := s.pool.Exec(ctx,
		`update chores set active = false, updated_at = now() where id = $1`, choreID)
	return err
}

// baselineCTE computes, per chore, the moment its clock last started: the most
// recent completion event, or the chore's creation if it has never been
// completed. Deriving it rather than storing it is what makes a sensor-written
// event reset the clock with no extra code.
//
// last_shown is filtered to p.kind in ('digest', 'nudge'). `?` (IntentQuery
// in apply.go) records a prompt_line for every active chore too, due or not,
// so without this filter asking "?" marked every chore as shown and the
// tolerance gate below hid all of them until their tolerance window passed
// again — up to a week of silence for one keystroke. Only an actual nudge
// (or, before the rename, a digest) may reset how recently a chore was
// shown.
//
// 'nudge' is the kind that actually names a chore now — 'digest' stays for
// rows that predate the split. 'evening' is deliberately excluded: on a
// nudge day the evening message shows nothing new about the nudged chore
// (nudgeFor already recorded it), and on a quiet day the evening message
// carries no chore lines at all. Without 'nudge' here, last_shown was
// permanently null once nothing wrote 'digest' any more, and the tolerance
// gate below — "last_shown is null or ..." — always took its null branch:
// PickChore's overdue weighting exists specifically to stop the same most-
// overdue chore from being nudged every single day, and a dead gate let it
// happen anyway.
const baselineCTE = `
	with baseline as (
	  select c.id,
	         coalesce((select max(e.occurred_at) from events e
	                    where e.chore_id = c.id and e.retracted_at is null),
	                  c.created_at) as since,
	         -- Only a digest or nudge that actually reached the room counts as
	         -- having shown a chore. The row is committed before the send, so a
	         -- failed send would otherwise start the tolerance clock on a
	         -- message nobody ever saw, hiding an overdue chore for its whole
	         -- tolerance window — the same silence a query used to cause.
	         (select max(p.sent_at)
	            from prompt_lines l join prompts p on p.id = l.prompt_id
	           where l.chore_id = c.id
	             and p.kind in ('digest', 'nudge')
	             and p.delivered_at is not null) as last_shown
	    from chores c
	   where c.person_id = $1 and c.active
	)`

func (s *Store) DueChores(ctx context.Context, personID int64, now time.Time) ([]Chore, error) {
	// The tolerance gate compares absolute instants, but the event it is
	// timed against — the daily digest — fires on a wall clock once a day,
	// not every 24 hours on the dot: ordinary scheduler tick jitter, and
	// twice a year the DST shift, both make consecutive mornings' sends land
	// anywhere from a little under to a little over 24 hours apart. Compared
	// against an untouched tolerance, a short morning silently pushes a chore
	// whose tolerance is a day or less out of the window it should have been
	// in, and it is quietly skipped that day. Slack subtracted here absorbs
	// both without risk of a second nudge the same day, because the digest
	// only ever fires once daily regardless.
	const q = baselineCTE + `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       extract(epoch from ($2::timestamptz - b.since))::bigint,
		       exists (select 1 from events e
		                where e.chore_id = c.id and e.retracted_at is null)
		  from chores c join baseline b on b.id = c.id
		 where c.person_id = $1 and c.active
		   and $2::timestamptz >= b.since + make_interval(secs => c.interval_seconds)
		   and (b.last_shown is null
		        or $2::timestamptz >= b.last_shown
		               + make_interval(secs => c.tolerance_seconds) - interval '2 hours')
		 order by extract(epoch from ($2::timestamptz - b.since)) / c.interval_seconds desc, c.name`

	return s.scanChores(ctx, q, personID, now)
}

func (s *Store) ActiveChores(ctx context.Context, personID int64) ([]Chore, error) {
	const q = baselineCTE + `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       extract(epoch from (now() - b.since))::bigint,
		       exists (select 1 from events e
		                where e.chore_id = c.id and e.retracted_at is null)
		  from chores c join baseline b on b.id = c.id
		 where c.person_id = $1 and c.active
		 order by c.name`

	return s.scanChores(ctx, q, personID)
}

func (s *Store) scanChores(ctx context.Context, q string, args ...any) ([]Chore, error) {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying chores: %w", err)
	}
	defer rows.Close()

	chores := []Chore{}
	for rows.Next() {
		var c Chore
		var everySec, tolSec, sinceSec int64
		if err := rows.Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec, &sinceSec, &c.EverDone); err != nil {
			return nil, fmt.Errorf("scanning chore: %w", err)
		}
		c.Active = true
		c.Every = time.Duration(everySec) * time.Second
		c.Tolerance = time.Duration(tolSec) * time.Second
		c.EveryDays = int(c.Every.Hours() / 24)
		c.SinceDays = int(time.Duration(sinceSec) * time.Second / (24 * time.Hour))
		chores = append(chores, c)
	}
	if err := rows.Err(); err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("reading chores: %w", err)
	}
	return chores, nil
}

// CapturesSince is the "Since yesterday" half of the digest — the raw text of
// what was captured, with commands filtered out so the list is thoughts rather
// than bookkeeping.
func (s *Store) CapturesSince(ctx context.Context, personID int64, since time.Time) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select raw_text, payload from items
		 where person_id = $1 and received_at >= $2 and raw_text <> ''
		 order by received_at`, personID, since)
	if err != nil {
		return nil, fmt.Errorf("querying captures: %w", err)
	}
	defer rows.Close()

	texts := []string{}
	for rows.Next() {
		var text string
		var payload json.RawMessage
		if err := rows.Scan(&text, &payload); err != nil {
			return nil, fmt.Errorf("scanning capture: %w", err)
		}
		// A tap is stored in items just like a genuine capture, encoded as
		// "!action <id> done:<n> <bool>". ParseAction matches on text alone,
		// so a person who types that same shape produces byte-identical text —
		// isActionPayload is the only thing that tells them apart, from the
		// payload the transport actually sent. apply's applyAction requires
		// both before treating something as a tap; requiring only ParseAction
		// here would give the digest a second, looser definition of "tap" and
		// silently drop a typed message that merely looked like one.
		if _, isTap := ParseAction(text); isTap && isActionPayload(payload) {
			continue
		}
		if matchFn(text).Kind == IntentCapture {
			texts = append(texts, text)
		}
	}
	return texts, rows.Err()
}

// CompletedToday names the chores completed since `since`, in the order they
// were done. Retracted events are excluded: a retraction means it did not
// happen, and reporting it back would contradict every other surface.
func (s *Store) CompletedToday(ctx context.Context, personID int64, since time.Time) ([]string, error) {
	const q = `
		select c.name from events e
		  join chores c on c.id = e.chore_id
		 where e.person_id = $1
		   and e.retracted_at is null
		   and e.occurred_at >= $2
		 order by e.occurred_at`

	rows, err := s.pool.Query(ctx, q, personID, since)
	if err != nil {
		return nil, fmt.Errorf("querying completions: %w", err)
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning completion: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
