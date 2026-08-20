package squirrel

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func URLFor(c PostgresConfig) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   "/" + c.Database,
	}
	return u.String()
}

func OpenStore(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Item is a capture as it lands in the database. SenderID survives alongside
// PersonID on purpose: PersonID is an interpretation and can be absent or
// wrong, SenderID is what the transport actually said.
type Item struct {
	// ID is filled on the read path only. InsertItem answers "was this a fresh
	// row", which is the question the drain asks; nothing needed an item's id
	// until something wanted to point at one — a numbered line, or a state
	// change. Filling it at capture time would put a returning clause on the
	// path that exists to be as short as it can be.
	ID             int64
	Transport      string
	ExternalID     *string
	ConversationID *string
	SenderID       *string
	PersonID       *int64
	RawText        string
	Payload        json.RawMessage
	ReceivedAt     time.Time
	// State is filled on the read path only, by itemsWhere. The capture path
	// does not set it: a fresh row takes the column default, which is `open`,
	// and a capture that had to know about triage would be a capture that can
	// fail for a triage reason.
	State ItemState
	// Kind is filled on the read path only, like State. A fresh row takes the
	// column default, which is `note`: capture does not decide what a thought
	// turns out to be.
	Kind ItemKind
	// PhotoName and PhotoType are the photograph this note carries, or empty.
	// A note has at most one: a note that can carry five is an album, and an
	// album is a thing you organise rather than glance at.
	PhotoName string
	PhotoType string
}

// InsertItemReturningID is InsertItem for the one caller that has to point at
// the row afterwards: something decided outright is a task the moment it
// exists, and marking it needs its id.
//
// No ON CONFLICT clause, because there is no external id to conflict on — this
// row did not arrive from a transport, it was typed here.
func (s *Store) InsertItemReturningID(ctx context.Context, i Item) (int64, error) {
	const q = `
		insert into items (
			transport, external_id, conversation_id, sender_id,
			person_id, raw_text, payload, received_at,
			attachment_path, attachment_type
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		returning id`

	var id int64
	if err := s.pool.QueryRow(ctx, q,
		i.Transport, i.ExternalID, i.ConversationID, i.SenderID,
		i.PersonID, i.RawText, []byte(i.Payload), i.ReceivedAt,
		nilIfEmpty(i.PhotoName), nilIfEmpty(i.PhotoType),
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("inserting item: %w", err)
	}
	return id, nil
}

// InsertItem is idempotent for a real external id. Redelivery is harmless to
// the row: the window between a committed insert and deleting the spool file
// is small, but it is real. The returned bool is what lets a caller tell a
// fresh row from an ON CONFLICT no-op — the drain needs it to apply an intent
// at most once per external id, not once per delivery.
//
// The `where external_id is not null` predicate is mandatory, not decoration.
// The unique index is partial, and Postgres rejects an ON CONFLICT target
// whose predicate does not match one.
func (s *Store) InsertItem(ctx context.Context, i Item) (bool, error) {
	const q = `
		insert into items (
			transport, external_id, conversation_id, sender_id,
			person_id, raw_text, payload, received_at,
			attachment_path, attachment_type
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		on conflict (transport, external_id) where external_id is not null
		do nothing`

	tag, err := s.pool.Exec(ctx, q,
		i.Transport, i.ExternalID, i.ConversationID, i.SenderID,
		i.PersonID, i.RawText, []byte(i.Payload), i.ReceivedAt,
		nilIfEmpty(i.PhotoName), nilIfEmpty(i.PhotoType))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// nilIfEmpty writes an absent photograph as null rather than as an empty
// string, so has_content's `attachment_path is not null` means what it says
// and a row with "" in it cannot pretend to hold a picture.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
