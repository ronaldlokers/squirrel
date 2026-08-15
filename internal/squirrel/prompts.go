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

const latestPrompt = `
	select id, sent_at from prompts
	 where person_id = $1 order by sent_at desc, id desc limit 1`

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
		 where l.prompt_id = (select id from prompts where person_id = $1
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
func (s *Store) LastDigestSentAt(ctx context.Context, personID int64) (time.Time, bool, error) {
	const q = `
		select sent_at from prompts
		 where person_id = $1 and sent_for_date is not null
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
