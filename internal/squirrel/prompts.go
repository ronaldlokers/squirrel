package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDigestAlreadySent means the unique index refused a second digest for a
// date. It is the expected outcome of a restart inside the send window, not a
// failure.
var ErrDigestAlreadySent = errors.New("digest already sent for this date")

// numberedKinds are the prompt kinds whose lines are numbered — the ones where
// a position means something.
//
// The rule for adding one: it prints a list, and it carries no sent_for_date so
// it cannot collide in the once-a-day unique index. Newest numbering wins, so a
// bare `done 1` after `!notes` means line 1 of the pile rather than line 1 of
// this morning's nudge.
//
// 'evening' is deliberately absent, and putting it back reopens a fixed bug. On
// a nudge day the evening row is stamped with the nudge's sent_at and ties
// break on id, which the evening row wins — so a numbered 'evening' won
// latestPrompt while carrying no lines of its own, and a typed "done 1" found
// nothing to resolve against on the very day the button beside it worked.
//
// 'now' carries exactly one line, which may be a chore or an item, so its
// buttons resolve through LineOnPrompt rather than ChoreOnPrompt.
const numberedKinds = `('digest', 'query', 'nudge', 'notes', 'find', 'tasks', 'now')`

// Prompt is a sent prompt, as much of it as anything outside this file needs.
type Prompt struct {
	ID                int64
	Kind              string
	ConversationID    string
	ExternalMessageID string
}

// LineRef is what a numbered line points at. Exactly one field is set, and the
// database enforces that rather than trusting every caller to remember.
type LineRef struct {
	ChoreID *int64
	ItemID  *int64
}

// Line is a resolved numbered line. Exactly one of Chore and Item is non-nil,
// which is why callers switch on nil rather than on a kind string: the compiler
// can check the first and cannot check the second.
type Line struct {
	Position int
	Chore    *Chore
	Item     *Item
}

// RecordPrompt records a prompt whose lines are all chores. It is the older,
// narrower shape of RecordPromptLines, kept because roughly twenty call sites
// across three phases pass []Chore — rewriting them all would spread the risk
// of this change over every one of them for no gain.
func (s *Store) RecordPrompt(ctx context.Context, personID int64, conversationID, kind string, sentAt time.Time, forDate *time.Time, chores []Chore) (int64, error) {
	lines := make([]LineRef, 0, len(chores))
	for _, c := range chores {
		lines = append(lines, LineRef{ChoreID: &c.ID})
	}
	return s.RecordPromptLines(ctx, personID, conversationID, kind, sentAt, forDate, lines)
}

// RecordPromptLines records a prompt whose numbered lines may be chores, notes,
// or both. See RecordPrompt for the chore-only shape.
func (s *Store) RecordPromptLines(ctx context.Context, personID int64, conversationID, kind string, sentAt time.Time, forDate *time.Time, lines []LineRef) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("recording prompt: %w", err)
	}
	defer tx.Rollback(ctx)

	var promptID int64
	err = tx.QueryRow(ctx, `
		insert into prompts (person_id, conversation_id, kind, sent_at, sent_for_date)
		values ($1, $2, $3, $4, $5) returning id`,
		personID, conversationID, kind, sentAt, forDate).Scan(&promptID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return 0, ErrDigestAlreadySent
	}
	if err != nil {
		return 0, fmt.Errorf("inserting prompt: %w", err)
	}

	for i, l := range lines {
		if _, err := tx.Exec(ctx, `
			insert into prompt_lines (prompt_id, position, chore_id, item_id)
			values ($1, $2, $3, $4)`, promptID, i+1, l.ChoreID, l.ItemID); err != nil {
			return 0, fmt.Errorf("inserting prompt line %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("committing prompt: %w", err)
	}
	return promptID, nil
}

// DeletePrompt removes a prompt row, and its lines by cascade, that RecordPrompt
// claimed but which was never delivered.
//
// Nudge uses it when Chat.Send fails: the dated row is committed before the send
// is attempted, so a transport error otherwise leaves a row claiming the day's
// slot in the unique index without having reached the room.
func (s *Store) DeletePrompt(ctx context.Context, promptID int64) error {
	if _, err := s.pool.Exec(ctx, `delete from prompts where id = $1`, promptID); err != nil {
		return fmt.Errorf("deleting prompt: %w", err)
	}
	return nil
}

// MarkPromptSent records that the send succeeded, and what Campfire called the
// message. Both facts arrive together and neither is useful without the other:
// delivered_at is how LastDigestSentAt tells a real digest from a row whose
// message never arrived, and external_message_id is how a later prompt disables
// this one's buttons and how a tap resolves back to it.
func (s *Store) MarkPromptSent(ctx context.Context, promptID int64, messageID string, deliveredAt time.Time) error {
	var id *string
	if messageID != "" {
		id = &messageID
	}
	_, err := s.pool.Exec(ctx,
		`update prompts set delivered_at = $2, external_message_id = $3 where id = $1`,
		promptID, deliveredAt, id)
	if err != nil {
		return fmt.Errorf("marking prompt sent: %w", err)
	}
	return nil
}

// PromptByMessageID resolves a tap back to the prompt that printed the button.
// Scoped by person: the message id arrives from the client, and this lookup is
// the only thing standing between one person's tap and another's chore.
func (s *Store) PromptByMessageID(ctx context.Context, personID int64, messageID string) (Prompt, bool, error) {
	const q = `
		select id, kind, conversation_id, coalesce(external_message_id, '')
		  from prompts where person_id = $1 and external_message_id = $2`

	var p Prompt
	err := s.pool.QueryRow(ctx, q, personID, messageID).
		Scan(&p.ID, &p.Kind, &p.ConversationID, &p.ExternalMessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Prompt{}, false, nil
	}
	if err != nil {
		return Prompt{}, false, fmt.Errorf("reading prompt by message: %w", err)
	}
	return p, true, nil
}

// PreviousNumberedPrompt is the numbered prompt before this one that actually
// reached the room. Its buttons are the ones to disable, and only a prompt with
// a message id has any.
func (s *Store) PreviousNumberedPrompt(ctx context.Context, personID int64, before int64) (Prompt, bool, error) {
	const q = `
		select id, kind, conversation_id, external_message_id
		  from prompts
		 where person_id = $1
		   and id <> $2
		   and kind in ` + numberedKinds + `
		   and external_message_id is not null
		 order by sent_at desc, id desc limit 1`

	var p Prompt
	err := s.pool.QueryRow(ctx, q, personID, before).
		Scan(&p.ID, &p.Kind, &p.ConversationID, &p.ExternalMessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Prompt{}, false, nil
	}
	if err != nil {
		return Prompt{}, false, fmt.Errorf("reading previous prompt: %w", err)
	}
	return p, true, nil
}

const latestPrompt = `
	select id, sent_at from prompts
	 where person_id = $1 and kind in ` + numberedKinds + `
	   and delivered_at is not null
	 order by sent_at desc, id desc limit 1`

// ChoreAtPosition resolves a numbered line back to the chore it named. Scoped by
// personID, and pinned to that person's single most recent prompt so a number is
// only read against the list that printed it.
//
// delivered_at is not null, matching latestPrompt: a failed send otherwise leaves
// a phantom numbered row that becomes "current" for a typed position while the
// room's buttons still point at the last prompt that went out.
func (s *Store) ChoreAtPosition(ctx context.Context, personID int64, position int) (Chore, bool, error) {
	const q = `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds
		  from prompt_lines l
		  join chores c on c.id = l.chore_id
		 where l.prompt_id = (select id from prompts
		                       where person_id = $1 and kind in ` + numberedKinds + `
		                         and delivered_at is not null
		                       order by sent_at desc, id desc limit 1)
		   and l.position = $2`

	var c Chore
	var everySec, tolSec int64
	err := s.pool.QueryRow(ctx, q, personID, position).
		Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec)
	if errors.Is(err, pgx.ErrNoRows) {
		return Chore{}, false, nil
	}
	if err != nil {
		return Chore{}, false, fmt.Errorf("reading prompt line: %w", err)
	}
	c.Active = true
	c.Every = time.Duration(everySec) * time.Second
	c.Tolerance = time.Duration(tolSec) * time.Second
	c.EveryDays = int(c.Every.Hours() / 24)
	return c, true, nil
}

// LineAtPosition is ChoreAtPosition generalised to a chore or a note, carrying
// every one of its predicates for the same reasons.
//
// Exactly one of Line.Chore and Line.Item is non-nil: the check constraint on
// prompt_lines makes that true, and the left joins make it observable.
func (s *Store) LineAtPosition(ctx context.Context, personID int64, position int) (Line, bool, error) {
	const q = `
		select l.position,
		       c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       i.id, i.raw_text, i.received_at, i.kind
		  from prompt_lines l
		  left join chores c on c.id = l.chore_id
		  left join items  i on i.id = l.item_id
		 where l.prompt_id = (select id from prompts
		                       where person_id = $1 and kind in ` + numberedKinds + `
		                         and delivered_at is not null
		                       order by sent_at desc, id desc limit 1)
		   and l.position = $2`

	var (
		line                 Line
		choreID, chorePerson *int64
		choreName            *string
		everySec, tolSec     *int64
		itemID               *int64
		itemText             *string
		itemAt               *time.Time
		itemKind             *string
	)
	err := s.pool.QueryRow(ctx, q, personID, position).Scan(
		&line.Position,
		&choreID, &chorePerson, &choreName, &everySec, &tolSec,
		&itemID, &itemText, &itemAt, &itemKind)
	if errors.Is(err, pgx.ErrNoRows) {
		return Line{}, false, nil
	}
	if err != nil {
		return Line{}, false, fmt.Errorf("reading prompt line: %w", err)
	}

	switch {
	case choreID != nil:
		c := Chore{
			ID:        *choreID,
			PersonID:  *chorePerson,
			Name:      *choreName,
			Active:    true,
			Every:     time.Duration(*everySec) * time.Second,
			Tolerance: time.Duration(*tolSec) * time.Second,
		}
		c.EveryDays = int(c.Every.Hours() / 24)
		line.Chore = &c
	case itemID != nil:
		// The kind comes with it because the caller needs to tell a note from
		// a task: only a task earns the hand-off after it is completed, and
		// without this that check silently compares against an empty string.
		line.Item = &Item{ID: *itemID, RawText: *itemText, ReceivedAt: *itemAt,
			Kind: itemKindOf(itemKind)}
	default:
		// Unreachable while prompt_lines_one_target holds. Reported rather
		// than returned as a zero Line, because a silent empty line would read
		// to the caller as "no such position" and send the wrong reply.
		return Line{}, false, fmt.Errorf("prompt line %d names neither a chore nor a note", position)
	}
	return line, true, nil
}

// latestChorePrompt is the most recent delivered numbered prompt that actually
// named a chore.
//
// Not latestPrompt: a numbered prompt can be a list of notes carrying no chore, so
// a `!notes` between the nudge and the answer made OutstandingLines return
// nothing and a bare `done` replied "Nothing outstanding." while the morning's
// chore sat unmet.
//
// Positions still resolve against latestPrompt — a typed number must mean the
// list that printed it.
const latestChorePrompt = `
	select p.id, p.sent_at from prompts p
	 where p.person_id = $1 and p.kind in ` + numberedKinds + `
	   and p.delivered_at is not null
	   and exists (select 1 from prompt_lines l
	                where l.prompt_id = p.id and l.chore_id is not null)
	 order by p.sent_at desc, p.id desc limit 1`

// OutstandingLines is the lines of the most recent chore-bearing prompt whose
// chore has had no completion since that prompt was sent. A bare `done`
// resolves against exactly one of these.
func (s *Store) OutstandingLines(ctx context.Context, personID int64) ([]Chore, error) {
	const q = `
		with latest as (` + latestChorePrompt + `)
		-- 0 and false: this query answers "what is still outstanding on the
		-- last numbered surface", where neither the elapsed time nor whether
		-- the chore was ever done is read by anyone. The asking window is read
		-- for the same reason the others are not: nothing here raises a chore,
		-- it only names ones already raised.
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds, 0::bigint, false,
		       c.ask_days, c.ask_part, c.on_weekday, c.every_weeks
		  from prompt_lines l
		  join latest p on p.id = l.prompt_id
		  join chores c on c.id = l.chore_id
		 where not exists (
		         select 1 from events e
		          where e.chore_id = c.id and e.occurred_at >= p.sent_at
		            and e.retracted_at is null)
		 order by l.position`

	return s.scanChores(ctx, q, personID)
}

// LastDigestSentAt is the sent_at of the most recent evening message. The
// scheduler anchors its capture window to it rather than to a fixed offset.
//
// kind is filtered to ('digest', 'evening') because a nudge carries a date too
// and is the newest dated prompt on most days while showing no captures.
// Anchoring to it would skip the window past everything captured between the two,
// permanently. 'digest' stays for rows predating the rename.
//
// delivered_at is not null is required too: a row can exist for a date whose
// message never reached Campfire, and anchoring to it would skip every capture
// made that day, forever.
func (s *Store) LastDigestSentAt(ctx context.Context, personID int64) (time.Time, bool, error) {
	const q = `
		select sent_at from prompts
		 where person_id = $1 and kind in ('digest', 'evening') and delivered_at is not null
		 order by sent_at desc, id desc limit 1`

	var sentAt time.Time
	err := s.pool.QueryRow(ctx, q, personID).Scan(&sentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("reading last digest: %w", err)
	}
	return sentAt, true, nil
}

// EveningDeliveredFor reports whether a delivered evening message already exists
// for a date. once() checks it before nudgeFor claims a slot: a process that
// delivered today's message and then restarted would otherwise spend today's
// nudge on a chore never shown.
//
// A plain read, not a lock. RecordPrompt's unique index is what actually decides;
// this closes the deterministic restart case, not the rare concurrent one.
func (s *Store) EveningDeliveredFor(ctx context.Context, personID int64, forDate time.Time) (bool, error) {
	const q = `
		select exists (
		  select 1 from prompts
		   where person_id = $1 and kind = 'evening' and sent_for_date = $2
		     and delivered_at is not null
		)`

	var exists bool
	if err := s.pool.QueryRow(ctx, q, personID, forDate).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking evening delivery: %w", err)
	}
	return exists, nil
}

// ChoreOnPrompt resolves a position against one specific prompt, rather than
// against whichever prompt is currently newest. A tap names the message it came
// from, so it must resolve against that message even if a newer prompt has
// since been sent.
func (s *Store) ChoreOnPrompt(ctx context.Context, promptID int64, position int) (Chore, bool, error) {
	const q = `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds
		  from prompt_lines l join chores c on c.id = l.chore_id
		 where l.prompt_id = $1 and l.position = $2`

	var c Chore
	var everySec, tolSec int64
	err := s.pool.QueryRow(ctx, q, promptID, position).
		Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec)
	if errors.Is(err, pgx.ErrNoRows) {
		return Chore{}, false, nil
	}
	if err != nil {
		return Chore{}, false, fmt.Errorf("reading prompt line: %w", err)
	}
	c.Active = true
	c.Every = time.Duration(everySec) * time.Second
	c.Tolerance = time.Duration(tolSec) * time.Second
	c.EveryDays = int(c.Every.Hours() / 24)
	return c, true, nil
}

// LineOnPrompt resolves a position against one specific prompt, whatever the line
// named. A tap names the message it came from, so it resolves against that
// message even if a newer prompt has since been sent.
//
// Exactly one of Line.Chore and Line.Item is non-nil.
func (s *Store) LineOnPrompt(ctx context.Context, promptID int64, position int) (Line, bool, error) {
	const q = `
		select l.position,
		       c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds,
		       i.id, i.raw_text, i.received_at, i.kind
		  from prompt_lines l
		  left join chores c on c.id = l.chore_id
		  left join items  i on i.id = l.item_id
		 where l.prompt_id = $1 and l.position = $2`

	var (
		line                 Line
		choreID, chorePerson *int64
		choreName            *string
		everySec, tolSec     *int64
		itemID               *int64
		itemText             *string
		itemAt               *time.Time
		itemKind             *string
	)
	err := s.pool.QueryRow(ctx, q, promptID, position).Scan(
		&line.Position,
		&choreID, &chorePerson, &choreName, &everySec, &tolSec,
		&itemID, &itemText, &itemAt, &itemKind)
	if errors.Is(err, pgx.ErrNoRows) {
		return Line{}, false, nil
	}
	if err != nil {
		return Line{}, false, fmt.Errorf("reading prompt line: %w", err)
	}

	switch {
	case choreID != nil:
		c := Chore{
			ID:        *choreID,
			PersonID:  *chorePerson,
			Name:      *choreName,
			Active:    true,
			Every:     time.Duration(*everySec) * time.Second,
			Tolerance: time.Duration(*tolSec) * time.Second,
		}
		c.EveryDays = int(c.Every.Hours() / 24)
		line.Chore = &c
	case itemID != nil:
		// The kind comes with it because the caller needs to tell a note from
		// a task: only a task earns the hand-off after it is completed, and
		// without this that check silently compares against an empty string.
		line.Item = &Item{ID: *itemID, RawText: *itemText, ReceivedAt: *itemAt,
			Kind: itemKindOf(itemKind)}
	default:
		return Line{}, false, fmt.Errorf("prompt line %d names neither a chore nor a note", position)
	}
	return line, true, nil
}

// ChoresOnPrompt is every chore a prompt carried, in the position order it
// printed them. closePrevious uses it to rebuild the exact action values a
// prompt was originally sent with — not a synthetic replacement — so that
// disabling those buttons is the only thing the update changes.
func (s *Store) ChoresOnPrompt(ctx context.Context, promptID int64) ([]Chore, error) {
	const q = `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds
		  from prompt_lines l join chores c on c.id = l.chore_id
		 where l.prompt_id = $1
		 order by l.position`

	rows, err := s.pool.Query(ctx, q, promptID)
	if err != nil {
		return nil, fmt.Errorf("reading prompt chores: %w", err)
	}
	defer rows.Close()

	chores := []Chore{}
	for rows.Next() {
		var c Chore
		var everySec, tolSec int64
		if err := rows.Scan(&c.ID, &c.PersonID, &c.Name, &everySec, &tolSec); err != nil {
			return nil, fmt.Errorf("scanning prompt chore: %w", err)
		}
		c.Active = true
		c.Every = time.Duration(everySec) * time.Second
		c.Tolerance = time.Duration(tolSec) * time.Second
		c.EveryDays = int(c.Every.Hours() / 24)
		chores = append(chores, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading prompt chores: %w", err)
	}
	return chores, nil
}

// CompletedSince reports whether the chore already has a live completion from
// after this prompt was sent. It is what makes a repeated "selected" tap a
// no-op instead of a second event.
func (s *Store) CompletedSince(ctx context.Context, choreID, promptID int64) (bool, error) {
	const q = `
		select exists (
		  select 1 from events e
		    join prompts p on p.id = $2
		   where e.chore_id = $1 and e.retracted_at is null and e.occurred_at >= p.sent_at)`

	var done bool
	if err := s.pool.QueryRow(ctx, q, choreID, promptID).Scan(&done); err != nil {
		return false, fmt.Errorf("checking completion: %w", err)
	}
	return done, nil
}

// itemKindOf reads the kind column, which is not null in the table but arrives
// through a left join and so can be null here — a line that names a chore has
// no item beside it. An absent kind is a note, which is what the column's own
// default says.
func itemKindOf(kind *string) ItemKind {
	if kind == nil {
		return ItemNote
	}
	return ItemKind(*kind)
}
