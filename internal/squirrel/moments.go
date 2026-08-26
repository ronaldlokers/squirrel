package squirrel

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// A fixed point, and the chain that hangs off it. See migration 0018 for the
// rule this is allowed under and the three things it still refuses.

// Defaults for the two halves of the chain nobody wants to type.
//
// Fifteen minutes of travel and ten of getting ready are guesses, and the
// message says they are guesses. A default that presents itself as a fact is
// how someone ends up late while trusting the machine.
const (
	defaultTravel = 15 * time.Minute
	defaultReady  = 10 * time.Minute
)

// Moment is a time the world imposed.
type Moment struct {
	ID       int64
	PersonID int64
	Label    string
	Starts   time.Time
	Travel   time.Duration
	Ready    time.Duration
	Bring    string
	// Guessed says the travel time was never given, so anything printed about
	// leaving has to admit it.
	Guessed bool
	Said    bool
}

// LeaveAt is when you would have to walk out. Never called a deadline and
// never rendered as one: it is arithmetic about a distance, not a judgement
// about you.
func (m Moment) LeaveAt() time.Time { return m.Starts.Add(-m.Travel) }

// WarnAt is when saying something is worth doing — one "get ready" ahead of
// leaving, because the failure is not knowing you had to go, it is not being
// ready to.
func (m Moment) WarnAt() time.Time { return m.LeaveAt().Add(-m.Ready) }

// Open reports whether now is inside the window where this moment is the
// answer to "what now". It closes at the start rather than at the leaving
// time: if you are already late there is nothing useful left to say, and
// saying it anyway would be the one sentence this product does not write.
func (m Moment) Open(now time.Time) bool {
	return !now.Before(m.WarnAt()) && now.Before(m.Starts)
}

// momentPattern is deliberately narrow: when in doubt, capture.
//
// The prefix is mandatory and is the whole of the bar — one of "at" or "tomorrow"
// has to be there. Without it "14:30 dentist" parsed as a fixed point, which is a
// note somebody wrote quickly, silently turned into something that interrupts
// them.
var momentPattern = regexp.MustCompile(
	`(?i)^(?:(tomorrow)\s+(?:at\s+)?|at\s+)(\d{1,2})(?::(\d{2}))?\s*(am|pm)?\s+(.+)$`)

// travelSuffix peels "20 minutes away" off the end, so the rest can be a label
// with commas in it.
var travelSuffix = regexp.MustCompile(`(?i)^(.*?)[,\s]+(\d{1,3})\s*(?:m|min|mins|minute|minutes)\s+away$`)

// ParseMoment reads a fixed point out of a sentence, in the process's own zone.
//
// It never guesses a date beyond tomorrow: a calendar is a thing you are behind
// on. A time that has already passed today means tomorrow, which is what a person
// means when they type it.
//
// Kept for callers that only ask whether a message has the shape of a fixed point
// — Match does, on stored rows. Anything that books one calls ParseMomentIn: a
// container's zone is an accident of its deployment. See issue #148.
func ParseMoment(s string, now time.Time) (Moment, bool) {
	return ParseMomentIn(now.Location(), s, now)
}

// ParseMomentIn reads one where the person is.
func ParseMomentIn(loc *time.Location, s string, now time.Time) (Moment, bool) {
	if loc == nil {
		loc = now.Location()
	}
	return MomentOn(loc, now.In(loc), s, now)
}

// MomentOn is the same sentence, anchored to a day you chose. One parser and one
// definition of what "14:30" means: a grammar that takes dates would widen the
// bar momentPattern sets, and a second builder in the web package is what
// composeEvery exists not to be.
func MomentOn(loc *time.Location, day time.Time, s string, now time.Time) (Moment, bool) {
	if loc == nil {
		loc = now.Location()
	}
	text := strings.TrimSpace(s)

	m := momentPattern.FindStringSubmatch(text)
	if m == nil {
		return Moment{}, false
	}

	hour, err := strconv.Atoi(m[2])
	if err != nil || hour > 23 {
		return Moment{}, false
	}
	minute := 0
	if m[3] != "" {
		if minute, err = strconv.Atoi(m[3]); err != nil || minute > 59 {
			return Moment{}, false
		}
	}
	meridiem := strings.ToLower(m[4])
	// A bare hour with no minutes and no am/pm is not a time, it is a number
	// someone wrote. "at 5 dentist" could be five o'clock or the fifth of
	// something, and guessing wrong books a warning for the wrong hour.
	if m[3] == "" && meridiem == "" {
		return Moment{}, false
	}
	switch {
	case meridiem == "pm" && hour < 12:
		hour += 12
	case meridiem == "am" && hour == 12:
		hour = 0
	}
	if hour > 23 {
		return Moment{}, false
	}

	label := strings.TrimSpace(m[5])
	travel := time.Duration(0)
	guessed := true
	if t := travelSuffix.FindStringSubmatch(label); t != nil {
		mins, err := strconv.Atoi(t[2])
		if err == nil && mins > 0 && mins <= 600 {
			label, travel, guessed = strings.TrimSpace(t[1]), time.Duration(mins)*time.Minute, false
		}
	}
	if label == "" {
		return Moment{}, false
	}
	if guessed {
		travel = defaultTravel
	}

	starts := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
	// Tomorrow if it was said, and tomorrow if the time has already gone. No
	// carve-out is needed for a chosen day: a day later than today is always after
	// now, and the picker offers no day in the past.
	if m[1] != "" || !starts.After(now) {
		starts = starts.AddDate(0, 0, 1)
	}

	return Moment{
		Label:   label,
		Starts:  starts,
		Travel:  travel,
		Ready:   defaultReady,
		Guessed: guessed,
	}, true
}

// CreateMoment stores one. It replaces nothing and merges nothing: two things
// at 14:30 is a real situation, and the picker names the nearer one.
func (s *Store) CreateMoment(ctx context.Context, personID int64, m Moment) (Moment, error) {
	var travel, ready *int64
	if !m.Guessed {
		secs := int64(m.Travel.Seconds())
		travel = &secs
	}
	if m.Ready > 0 {
		secs := int64(m.Ready.Seconds())
		ready = &secs
	}

	err := s.pool.QueryRow(ctx, `
		insert into moments (person_id, label, starts_at, travel_secs, ready_secs)
		values ($1, $2, $3, $4, $5)
		returning id, person_id`,
		personID, m.Label, m.Starts, travel, ready).Scan(&m.ID, &m.PersonID)
	if err != nil {
		return Moment{}, fmt.Errorf("keeping a fixed point: %w", err)
	}
	return m, nil
}

// NextMoment is the soonest one still to come, or nothing.
//
// One, never a list. There is deliberately no function here that returns
// several — the same guarantee the check-in keeps, and for a related reason: a
// list of what is coming is a day you are already behind on before it starts.
func (s *Store) NextMoment(ctx context.Context, personID int64, now time.Time) (Moment, bool, error) {
	const q = `
		select id, person_id, label, starts_at, travel_secs, ready_secs,
		       coalesce(bring, ''), said_at is not null
		  from moments
		 where person_id = $1 and done_at is null and starts_at > $2
		 order by starts_at limit 1`

	return s.scanMoment(ctx, q, personID, now)
}

// DueMoment is the next one whose warning has not been given and is due to be.
//
// Its own query rather than NextMoment's, which answers "what is the next fixed
// point". Answering this with that left an already-warned moment at the head of
// the queue and every later one invisible until it started.
//
// The cost was not a missed test: a moment blocked past its own warn point is
// never warned about at all, because the scheduler refuses to send one late — so
// two appointments half an hour apart meant the second arrived in silence.
//
// `said_at is null` is what makes it a queue rather than a peek. The upper bound
// stays `starts_at > $2`: once a thing has started, its warning is over.
func (s *Store) DueMoment(ctx context.Context, personID int64, now time.Time) (Moment, bool, error) {
	const q = `
		select id, person_id, label, starts_at, travel_secs, ready_secs,
		       coalesce(bring, ''), said_at is not null
		  from moments
		 where person_id = $1 and done_at is null and starts_at > $2
		   and said_at is null
		 order by starts_at limit 1`

	m, found, err := s.scanMoment(ctx, q, personID, now)
	if err != nil || !found || now.Before(m.WarnAt()) {
		return Moment{}, false, err
	}
	return m, true, nil
}

// scannable is what pgx.Row and pgx.Rows have in common, which is all this
// needs. Two read paths that default travel and ready differently would put two
// screens out of step about when to leave, so there is one body and both use it.
type scannable interface {
	Scan(dest ...any) error
}

// momentFrom is the column order and the defaulting, in one place.
//
// A fixed point with no travel time recorded guesses fifteen minutes and says
// so with Guessed, because anything printed about leaving has to admit it was
// a guess.
func momentFrom(row scannable) (Moment, error) {
	var (
		m            Moment
		travel, redy *int64
	)
	if err := row.Scan(&m.ID, &m.PersonID, &m.Label, &m.Starts,
		&travel, &redy, &m.Bring, &m.Said); err != nil {
		return Moment{}, err
	}
	m.Travel, m.Guessed = defaultTravel, true
	if travel != nil {
		m.Travel, m.Guessed = time.Duration(*travel)*time.Second, false
	}
	m.Ready = defaultReady
	if redy != nil {
		m.Ready = time.Duration(*redy) * time.Second
	}
	return m, nil
}

func (s *Store) scanMoment(ctx context.Context, q string, args ...any) (Moment, bool, error) {
	m, err := momentFrom(s.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Moment{}, false, nil
	}
	if err != nil {
		return Moment{}, false, fmt.Errorf("reading a fixed point: %w", err)
	}
	m.Starts = s.here(m.Starts)
	return m, true, nil
}

// MomentByID is one fixed point, yours or nobody's.
//
// The person is in the where clause rather than checked after, so a stranger's
// id and a missing id are the same answer and neither leaks that the row
// exists.
func (s *Store) MomentByID(ctx context.Context, personID, id int64) (Moment, bool, error) {
	const q = `
		select id, person_id, label, starts_at, travel_secs, ready_secs,
		       coalesce(bring, ''), said_at is not null
		  from moments
		 where id = $2 and person_id = $1`

	return s.scanMoment(ctx, q, personID, id)
}

// MarkMomentSaid records that the leave-by warning went out, so a second tick
// does not repeat it.
func (s *Store) MarkMomentSaid(ctx context.Context, id int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`update moments set said_at = $2 where id = $1`, id, at); err != nil {
		return fmt.Errorf("marking a fixed point said: %w", err)
	}
	return nil
}

// MomentDone closes one: you left, or it stopped mattering. Either way nothing
// will raise it again, and nothing anywhere records which of the two it was.
func (s *Store) MomentDone(ctx context.Context, personID, id int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`update moments set done_at = $3 where id = $1 and person_id = $2`,
		id, personID, at); err != nil {
		return fmt.Errorf("closing a fixed point: %w", err)
	}
	return nil
}

// SetMomentBring attaches what to take to the next moment.
//
// The next one rather than a named one, because the question is only ever
// asked a moment after making it — and asking which appointment you mean, of
// the one you just typed, is the tax this product exists to stop charging.
func (s *Store) SetMomentBring(ctx context.Context, personID int64, bring string, now time.Time) (Moment, bool, error) {
	m, found, err := s.NextMoment(ctx, personID, now)
	if err != nil || !found {
		return Moment{}, false, err
	}
	if _, err := s.pool.Exec(ctx,
		`update moments set bring = $3 where id = $1 and person_id = $2`,
		m.ID, personID, bring); err != nil {
		return Moment{}, false, fmt.Errorf("noting what to take: %w", err)
	}
	m.Bring = bring
	return m, true, nil
}

// pickMoment is rule 1: a fixed point, inside the window where leaving matters.
// Ahead of everything including a running timer, and untouched by the capacity
// gate — a low day is exactly the day you most need telling to leave.
func (s *Store) pickMoment(ctx context.Context, personID int64, now time.Time) (Offer, bool, error) {
	m, found, err := s.NextMoment(ctx, personID, now)
	if err != nil || !found || !m.Open(now) {
		return Offer{}, false, err
	}
	because := LeaveWords(m)
	if m.Bring != "" {
		// The thing that actually fails is not knowing you had to go, it is
		// standing in the hall without your keys. If it was written down, it
		// belongs on the one surface you are looking at as you leave.
		because += " · take " + m.Bring
	}
	return Offer{
		Kind:    OfferMoment,
		RefID:   m.ID,
		Text:    m.Label,
		Because: because,
	}, true, nil
}

// LeaveWords says the start time and the leaving time, and says so when the
// travel time was a guess. Never "you are late", never a countdown.
func LeaveWords(m Moment) string {
	at := m.Starts.Format("15:04")
	leave := m.LeaveAt().Format("15:04")
	if m.Guessed {
		return fmt.Sprintf("at %s — leave about %s, if it is a quarter of an hour away", at, leave)
	}
	return fmt.Sprintf("at %s — leave about %s", at, leave)
}

// AttachNote points a note at a fixed point. The person is in the where clause
// rather than checked first, so there is no window between read and write and no
// way to forget the check. The moment is checked in the same statement.
func (s *Store) AttachNote(ctx context.Context, personID, itemID, momentID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set moment_id = $3
		 where id = $2 and person_id = $1
		   and exists (select 1 from moments where id = $3 and person_id = $1)`,
		personID, itemID, momentID)
	if err != nil {
		return false, fmt.Errorf("pointing a note at a fixed point: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// DetachNote puts it back in the pile. There is no previous value to remember
// because the pointer was the whole of the change.
func (s *Store) DetachNote(ctx context.Context, personID, itemID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`update items set moment_id = null where id = $2 and person_id = $1`,
		personID, itemID)
	if err != nil {
		return false, fmt.Errorf("returning a note to the pile: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// NotesFor is what is pointing at one fixed point, newest first like every
// other list in this product.
func (s *Store) NotesFor(ctx context.Context, personID, momentID int64) ([]Item, error) {
	items, _, err := s.itemsWhere(ctx,
		`person_id = $1 and moment_id = $2`, notesForLimit, personID, momentID)
	return items, err
}

// notesForLimit is a bound rather than a page: nothing here says how many there
// are, and a limit nothing reaches in practice is a limit nobody sees.
const notesForLimit = 50

// Upcoming is what is still ahead, soonest first. What makes a list of
// appointments allowed here is that it holds nothing you can be behind on —
// `starts_at > $2` and `done_at is null`.
//
// It returns rows and never a total.
func (s *Store) Upcoming(ctx context.Context, personID int64, now time.Time, limit int) ([]Moment, error) {
	const q = `
		select id, person_id, label, starts_at, travel_secs, ready_secs,
		       coalesce(bring, ''), said_at is not null
		  from moments
		 where person_id = $1 and done_at is null and starts_at > $2
		 order by starts_at limit $3`

	rows, err := s.pool.Query(ctx, q, personID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("reading what is coming: %w", err)
	}
	defer rows.Close()

	out := []Moment{}
	for rows.Next() {
		m, err := momentFrom(rows)
		if err != nil {
			return nil, fmt.Errorf("reading what is coming: %w", err)
		}
		m.Starts = s.here(m.Starts)
		out = append(out, m)
	}
	return out, rows.Err()
}
