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
//
// 'nudge' belongs here, not 'evening'. A nudge always carries its chosen
// chore as its own line 1 and, when it is delivered standalone or claimed
// fresh inside an evening send, the real message id too — see schedule.go's
// once() and nudgeFor. 'evening' was added here briefly in an earlier round
// and then taken back out: on a nudge day the evening row is stamped with
// the same sent_at as the nudge row it rode in on, and ties break on id,
// which the evening row always wins since it is recorded second. With
// 'evening' numbered, that made the evening row — which carries no lines of
// its own — win latestPrompt on the very day a nudge just opened a live
// button, so a typed "done 1" or a bare "done" found nothing to resolve
// against while the button on the very same message worked fine. Leaving
// 'evening' out means the typed path always resolves through the nudge row,
// which is the one that actually owns both the line and, when it has one,
// the id — agreeing with the tapped path in every case rather than only
// some.
const numberedKinds = `('digest', 'query', 'nudge')`

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

// DeletePrompt removes a prompt row (and, via the cascading foreign key on
// prompt_lines, its lines) that was claimed by RecordPrompt but never
// delivered. Nudge uses it when Chat.Send fails: RecordPrompt commits the
// dated row before the send is attempted, the same ordering the evening
// message uses and for the same reason, so a transport error leaves a row
// that claims the day's nudge slot in the unique index on
// (person_id, kind, sent_for_date) without ever having reached the room.
// Nothing depends on an undelivered row — the message never went out — so
// deleting it gives a later trigger the same day, including the 19:00
// fallback, a real chance to claim the slot instead of being refused by a
// send that never happened.
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

// ChoreAtPosition resolves a numbered line — "done 2" — back to the chore it
// named. Scoped by personID so one person's number can never resolve to
// another person's chore, and pinned to that one person's single most recent
// prompt so the number is only ever read against the list that printed it,
// never against some other prompt that happens to share a position.
//
// delivered_at is not null, matching latestPrompt: replyFor commits a query
// prompt before the message carrying its buttons is sent, so a failed send
// leaves a numbered row with no message ever in the room. Without this
// predicate that phantom row would become "current" for a typed position
// even though the room's actual buttons still point at the last prompt that
// really went out — a typed "done 1" and a tap on button 1 resolving to two
// different chores.
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
		          where e.chore_id = c.id and e.occurred_at >= p.sent_at
		            and e.retracted_at is null)
		 order by l.position`

	return s.scanChores(ctx, q, personID)
}

// LastDigestSentAt is the sent_at of the person's most recent evening
// message — a digest, in the old naming this method keeps. The scheduler
// anchors its capture window to this instant rather than to a fixed
// "yesterday midnight" offset.
//
// kind is filtered to ('digest', 'evening') rather than just "sent_for_date
// is not null", because a nudge carries a date too — it has to, to get its
// own slot in the per-day index — and a nudge is the newest dated prompt on
// most days while showing no captures at all. Anchoring to it instead of the
// last real evening message would skip the capture window forward past
// everything captured between the two, silently and permanently: gone from
// every future evening message, since each one's window starts from the
// last one's sent_at. 'digest' stays in the list so the first evening
// message after this kind was renamed still anchors off the last digest
// that actually sent, rather than jumping backwards over however many days
// of captures came before the rename.
//
// delivered_at is not null is required too. RecordPrompt commits a dated
// prompt's row before Send is attempted, so a row can exist for a date whose
// message never reached Campfire. Anchoring to that row anyway would skip
// the capture window right past every capture made on the day the send
// failed — gone from every evening message forever, since the next
// successful one's window starts from the (wrongly early) failed row's
// sent_at, not from the last time a message actually arrived.
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

// EveningDeliveredFor reports whether a delivered evening message already
// exists for the given date. once() checks this before nudgeFor claims a
// nudge slot: without it, a process that delivered today's evening message
// and then restarted — losing its in-memory sentDate guard in the process —
// would still let nudgeFor claim and commit a nudge row on its way to a
// doomed RecordPrompt("evening", ...) collision, spending today's nudge slot
// on a chore that is never shown for it.
//
// This is a plain read, not a lock: a genuinely concurrent second process can
// still race between this check and the writes that follow it in once(), and
// when it does, RecordPrompt's unique index is what actually decides,
// same as always — this closes the deterministic restart case, not the rare
// concurrent one.
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
