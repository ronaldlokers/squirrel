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
// a position means something. A definition confirmation carries a button but is
// a standalone surface: it names one chore and is never counted against.
const numberedKinds = `('digest', 'query')`

// Prompt is a sent prompt, as much of it as anything outside this file needs.
type Prompt struct {
	ID                int64
	Kind              string
	ConversationID    string
	ExternalMessageID string
}

func (s *Store) RecordPrompt(ctx context.Context, personID int64, conversationID, kind string, sentAt time.Time, forDate *time.Time, chores []Chore) (int64, error) {
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

	for i, c := range chores {
		if _, err := tx.Exec(ctx, `
			insert into prompt_lines (prompt_id, position, chore_id)
			values ($1, $2, $3)`, promptID, i+1, c.ID); err != nil {
			return 0, fmt.Errorf("inserting prompt line %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("committing prompt: %w", err)
	}
	return promptID, nil
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
	 order by sent_at desc, id desc limit 1`

// ChoreAtPosition resolves a numbered line — "done 2" — back to the chore it
// named. Scoped by personID so one person's number can never resolve to
// another person's chore, and pinned to that one person's single most recent
// prompt so the number is only ever read against the list that printed it,
// never against some other prompt that happens to share a position.
func (s *Store) ChoreAtPosition(ctx context.Context, personID int64, position int) (Chore, bool, error) {
	const q = `
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds
		  from prompt_lines l
		  join chores c on c.id = l.chore_id
		 where l.prompt_id = (select id from prompts
		                       where person_id = $1 and kind in ` + numberedKinds + `
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

// OutstandingLines is the lines of the most recent prompt whose chore has had
// no completion since that prompt was sent. A bare `done` resolves against
// exactly one of these.
func (s *Store) OutstandingLines(ctx context.Context, personID int64) ([]Chore, error) {
	const q = `
		with latest as (` + latestPrompt + `)
		select c.id, c.person_id, c.name, c.interval_seconds, c.tolerance_seconds, 0::bigint
		  from prompt_lines l
		  join latest p on p.id = l.prompt_id
		  join chores c on c.id = l.chore_id
		 where not exists (
		         select 1 from events e
		          where e.chore_id = c.id and e.occurred_at >= p.sent_at)
		 order by l.position`

	return s.scanChores(ctx, q, personID)
}

// LastDigestSentAt is the sent_at of the person's most recent digest — a
// prompt with a non-null sent_for_date, which is exactly what distinguishes a
// digest from a query prompt issued on demand. The scheduler anchors its
// capture window to this instant rather than to a fixed "yesterday midnight"
// offset, so a prompt with a null date (a query, not a digest) must never be
// picked here: anchoring to one would shrink the window to however many
// minutes ago the last "?" was sent and hide everything captured before it.
//
// delivered_at is not null is required too. RecordPrompt commits a digest's
// row before Send is attempted, so a row can exist for a date whose message
// never reached Campfire. Anchoring to that row anyway would skip the
// capture window right past every capture made on the day the send failed —
// gone from every digest forever, since the next successful digest's window
// starts from the (wrongly early) failed row's sent_at, not from the last
// time a message actually arrived.
func (s *Store) LastDigestSentAt(ctx context.Context, personID int64) (time.Time, bool, error) {
	const q = `
		select sent_at from prompts
		 where person_id = $1 and sent_for_date is not null and delivered_at is not null
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
