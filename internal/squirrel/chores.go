package squirrel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	// Ask is when raising it is worth doing. It never changes when the chore
	// is due — see asking.go for why those are deliberately two questions.
	Ask Asking
	// Weekday and Weeks are a chore that comes back on a day rather than after an
	// interval — alternating Thursdays, and the reason the bins never fitted.
	//
	// Weeks is 0 for the interval rhythm. When it is 1 or 2, Weekday is the day and
	// Every still carries the equivalent interval so everything that asks "how often"
	// keeps working.
	//
	// An interval measured from the last completion slides: do the bins a day late
	// once and every reminder after it is a day late too. A weekday does not.
	Weekday time.Weekday
	Weeks   int
}

// OnADay reports whether this chore comes back on a weekday rather than after
// an interval.
func (c Chore) OnADay() bool { return c.Weeks > 0 }

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
	return s.UpsertChoreAsking(ctx, personID, name, every, tolerance, Asking{})
}

// UpsertChoreAsking is UpsertChore plus when the chore is worth raising. An empty
// Asking leaves what is already there; clearing one is done by saying a different
// one.
func (s *Store) UpsertChoreAsking(ctx context.Context, personID int64, name string, every, tolerance time.Duration, ask Asking) (Chore, error) {
	const q = `
		insert into chores (person_id, name, interval_seconds, tolerance_seconds, ask_days, ask_part)
		values ($1, $2, $3, $4, $5, $6)
		on conflict (person_id, lower(name)) do update
		  set interval_seconds = excluded.interval_seconds,
		      tolerance_seconds = excluded.tolerance_seconds,
		      ask_days = coalesce(excluded.ask_days, chores.ask_days),
		      ask_part = coalesce(excluded.ask_part, chores.ask_part),
		      active = true,
		      updated_at = now()
		returning id, person_id, name, interval_seconds, tolerance_seconds, active`

	var days *int16
	if ask.Days != AnyDay {
		d := int16(ask.Days)
		days = &d
	}
	var part *string
	if ask.Part != AnyPart {
		p := string(ask.Part)
		part = &p
	}

	var c Chore
	var everySec, tolSec int64
	err := s.pool.QueryRow(ctx, q, personID, name,
		int64(every.Seconds()), int64(tolerance.Seconds()), days, part,
	).Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec, &c.Active)
	if err != nil {
		return Chore{}, fmt.Errorf("upserting chore %q: %w", name, err)
	}
	c.Every = time.Duration(everySec) * time.Second
	c.Tolerance = time.Duration(tolSec) * time.Second
	c.EveryDays = int(c.Every.Hours() / 24)
	c.Ask = ask
	return c, nil
}

// SnoozeChore silences a chore until a moment without pretending it was done. The
// baseline is untouched, so nothing about when it is next due changes — it is the
// asking that stops. A time in the past clears it.
func (s *Store) SnoozeChore(ctx context.Context, choreID, personID int64, until time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update chores set snoozed_until = $3, updated_at = now()
		 where id = $1 and person_id = $2 and active`, choreID, personID, until)
	if err != nil {
		return false, fmt.Errorf("snoozing chore: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) DeactivateChore(ctx context.Context, choreID int64) error {
	_, err := s.pool.Exec(ctx,
		`update chores set active = false, updated_at = now() where id = $1`, choreID)
	return err
}

// baselineCTE computes, per chore, the moment its clock last started: the most
// recent completion event, or creation if never completed. Derived rather than
// stored, so a sensor-written event resets the clock with no extra code.
//
// last_shown is filtered to p.kind in ('digest', 'nudge'). `?` records a
// prompt_line for every active chore, so without this filter one keystroke marked
// every chore as shown and the tolerance gate hid them for up to a week.
//
// 'evening' is excluded: on a nudge day it shows nothing new, and on a quiet day
// it carries no chore lines. Without 'nudge' here last_shown was permanently null
// once nothing wrote 'digest', so the tolerance gate always took its null branch
// and the same most-overdue chore was nudged every day.
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
	// The tolerance gate compares absolute instants, but the digest fires on a wall
	// clock: tick jitter and the DST shift make consecutive mornings land a little
	// under or over 24 hours apart. Against an untouched tolerance a short morning
	// silently pushes a day-or-less chore out of its window. The slack absorbs both,
	// and cannot cause a second nudge because the digest fires once daily.
	// Every date in the weekday rule below is read in $3, where the person is,
	// rather than in the session's own zone — which is UTC in production and in
	// the suite. extract(dow) and ::date both follow the session, so between
	// midnight and 02:00 in Amsterdam the rule was asking about yesterday: the
	// bins were not due at 00:30 on a Thursday and were due at 00:30 on a
	// Friday. Issue #148, one feature further on.
	const q = baselineCTE + `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       extract(epoch from ($2::timestamptz - b.since))::bigint,
		       exists (select 1 from events e
		                where e.chore_id = c.id and e.retracted_at is null),
		       c.ask_days, c.ask_part, c.on_weekday, c.every_weeks
		  from chores c join baseline b on b.id = c.id
		 where c.person_id = $1 and c.active
		   -- Snoozed is not done: the baseline above is untouched, so the chore
		   -- is exactly as due as it was when the clock comes back round. This
		   -- only stops the asking.
		   and (c.snoozed_until is null or $2::timestamptz >= c.snoozed_until)
		   and case when c.on_weekday is null then
		            -- The interval rhythm, unchanged since 0002.
		            $2::timestamptz >= b.since + make_interval(secs => c.interval_seconds)
		       else
		            -- A day, not an interval. Due when today *is* the day, the
		            -- week lines up, and it has not already been done today.
		            --
		            -- The parity is counted in whole weeks from the chore's own
		            -- creation rather than from an ISO week number, which wraps
		            -- at the turn of the year and would flip every alternating
		            -- chore in the house on 1 January.
		            extract(dow from ($2::timestamptz at time zone coalesce(nullif($3, ''), current_setting('TimeZone'))))::int
		                    = c.on_weekday
		            and (($2::timestamptz at time zone coalesce(nullif($3, ''), current_setting('TimeZone')))::date
		                 - (c.created_at at time zone coalesce(nullif($3, ''), current_setting('TimeZone')))::date) / 7
		                    % c.every_weeks = 0
		            and (b.since at time zone coalesce(nullif($3, ''), current_setting('TimeZone')))::date
		                    < ($2::timestamptz at time zone coalesce(nullif($3, ''), current_setting('TimeZone')))::date
		       end
		   and (b.last_shown is null
		        or $2::timestamptz >= b.last_shown
		               + make_interval(secs => c.tolerance_seconds) - interval '2 hours')
		 order by extract(epoch from ($2::timestamptz - b.since)) / c.interval_seconds desc, c.name`

	return s.scanChores(ctx, q, personID, now, s.zone())
}

// SearchChores finds an active chore by name, because searching is one thing
// rather than two: the lid carries one field, and a person searching for a word
// does not first classify what kind of thing they are looking for.
//
// Escaped like SearchItems: a typed % is a character.
func (s *Store) SearchChores(ctx context.Context, personID int64, query string, limit int) ([]Chore, error) {
	const q = baselineCTE + `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       extract(epoch from (now() - b.since))::bigint,
		       exists (select 1 from events e
		                where e.chore_id = c.id and e.retracted_at is null),
		       c.ask_days, c.ask_part, c.on_weekday, c.every_weeks
		  from chores c join baseline b on b.id = c.id
		 where c.person_id = $1 and c.active
		   and lower(c.name) like $2 escape '\'
		 order by c.name
		 limit $3`

	pattern := "%" + likeEscape(strings.ToLower(query)) + "%"
	return s.scanChores(ctx, q, personID, pattern, limit)
}

func (s *Store) ActiveChores(ctx context.Context, personID int64) ([]Chore, error) {
	const q = baselineCTE + `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       extract(epoch from (now() - b.since))::bigint,
		       exists (select 1 from events e
		                where e.chore_id = c.id and e.retracted_at is null),
		       c.ask_days, c.ask_part, c.on_weekday, c.every_weeks
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
		// Both nullable, and null means no preference rather than a preference
		// for nothing.
		var askDays *int16
		var askPart *string
		// Both null together, or both set: the schema has a constraint saying
		// so, because a weekday with no period would fall back to the interval
		// silently.
		var onWeekday, everyWeeks *int16
		if err := rows.Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec, &sinceSec,
			&c.EverDone, &askDays, &askPart, &onWeekday, &everyWeeks); err != nil {
			return nil, fmt.Errorf("scanning chore: %w", err)
		}
		if onWeekday != nil && everyWeeks != nil {
			c.Weekday, c.Weeks = time.Weekday(*onWeekday), int(*everyWeeks)
		}
		if askDays != nil {
			c.Ask.Days = Days(*askDays)
		}
		if askPart != nil {
			c.Ask.Part = DayPart(*askPart)
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
		 where person_id = $1 and received_at >= $2 and has_content
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
		// A tap is stored in items like a genuine capture, and ParseAction matches on
		// text alone, so isActionPayload is the only thing telling a tap from someone
		// typing the same shape. Requiring only ParseAction here would give the digest a
		// looser definition of "tap" and silently drop a message that looked like one.
		if _, isTap := ParseAction(text); isTap && isActionPayload(payload) {
			continue
		}
		if matchFn(text).Kind == IntentCapture {
			texts = append(texts, text)
		}
	}
	return texts, rows.Err()
}

// HandledToday is everything that happened today worth saying back: the chores
// and tasks named, the notes counted.
//
// The count is the one number in this product. The banned counter counts what
// remains — it grows unwatched, sits beside an implied zero, and can be lost.
// This counts what happened, in the past, on one day.
type Handled struct {
	Chores []string
	Tasks  []string
	// Notes is how many notes were triaged today, in any direction. Clearing
	// one is work whichever exit it took, and splitting it into done, dropped
	// and kept would be three numbers where one is already the most that
	// should be said.
	Notes int
}

// HandledSince gathers it in one place so the evening message does not have to
// know three queries.
func (s *Store) HandledSince(ctx context.Context, personID int64, since time.Time) (Handled, error) {
	var h Handled

	chores, err := s.CompletedToday(ctx, personID, since)
	if err != nil {
		return Handled{}, err
	}
	h.Chores = chores

	rows, err := s.pool.Query(ctx, `
		select raw_text from items
		 where person_id = $1 and kind = 'task' and state = 'done'
		   and state_at >= $2
		 order by state_at`, personID, since)
	if err != nil {
		return Handled{}, fmt.Errorf("querying what you did: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return Handled{}, fmt.Errorf("scanning what you did: %w", err)
		}
		h.Tasks = append(h.Tasks, text)
	}
	if err := rows.Err(); err != nil {
		return Handled{}, fmt.Errorf("querying what you did: %w", err)
	}

	// Notes only: a task leaving the pile is already named above, and counting
	// it here as well would say the same thing twice.
	if err := s.pool.QueryRow(ctx, `
		select count(*) from items
		 where person_id = $1 and kind = 'note' and state <> 'open'
		   and state_at >= $2 and has_content`, personID, since).Scan(&h.Notes); err != nil {
		return Handled{}, fmt.Errorf("counting what you cleared: %w", err)
	}
	return h, nil
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

// SetChoreRhythm makes a chore come back on a day rather than after an interval,
// or puts it back on an interval.
//
// Separate from UpsertChore because the two writes answer different questions and
// almost every caller only asks the first.
//
// It writes interval_seconds too, which is the point of the design: everything
// that renders "how often", every asking window and the tolerance gate keep
// reading the column they always read. weeks of 0 clears the rhythm.
func (s *Store) SetChoreRhythm(ctx context.Context, personID, choreID int64, day time.Weekday, weeks int) error {
	if weeks < 0 || weeks > 2 {
		return fmt.Errorf("not a rhythm this product has: every %d weeks", weeks)
	}
	if weeks == 0 {
		_, err := s.pool.Exec(ctx, `
			update chores set on_weekday = null, every_weeks = null, updated_at = now()
			 where id = $1 and person_id = $2`, choreID, personID)
		if err != nil {
			return fmt.Errorf("putting a chore back on an interval: %w", err)
		}
		return nil
	}
	if day < time.Sunday || day > time.Saturday {
		return fmt.Errorf("not a day: %d", day)
	}

	every := time.Duration(weeks) * 7 * 24 * time.Hour
	_, err := s.pool.Exec(ctx, `
		update chores
		   set on_weekday = $3, every_weeks = $4,
		       interval_seconds = $5,
		       tolerance_seconds = $6,
		       updated_at = now()
		 where id = $1 and person_id = $2`,
		choreID, personID, int16(day), int16(weeks),
		int64(every/time.Second), int64(DefaultTolerance(every)/time.Second))
	if err != nil {
		return fmt.Errorf("giving a chore a day: %w", err)
	}
	return nil
}

// DayNamed is a weekday from what somebody typed, and whether it was one.
//
// Three letters is enough and the whole word is accepted, because a picker
// sends "thursday" and a person types "thu". Nothing here is localised: the
// screen offers the days it knows and this reads them back.
func DayNamed(said string) (time.Weekday, bool) {
	said = strings.ToLower(strings.TrimSpace(said))
	if len(said) < 3 {
		return 0, false
	}
	for d := time.Sunday; d <= time.Saturday; d++ {
		name := strings.ToLower(d.String())
		if said == name || said == name[:3] {
			return d, true
		}
	}
	return 0, false
}
