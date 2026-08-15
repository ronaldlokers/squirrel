package squirrel

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type IdentitySeed struct {
	Transport  string
	ExternalID string
}

// SeedOwner is declarative rather than administrative: the desired state lives
// in configuration, in Git, and every boot reconciles to it. There is no admin
// screen to forget about, which is why it must be idempotent.
func (s *Store) SeedOwner(ctx context.Context, handle string, seeds []IdentitySeed) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("seeding owner: %w", err)
	}
	defer tx.Rollback(ctx)

	var personID int64
	err = tx.QueryRow(ctx, `
		insert into people (handle) values ($1)
		on conflict (handle) do update set handle = excluded.handle
		returning id`, handle).Scan(&personID)
	if err != nil {
		return 0, fmt.Errorf("seeding person %s: %w", handle, err)
	}

	for _, seed := range seeds {
		if _, err := tx.Exec(ctx, `
			insert into identities (person_id, transport, external_id)
			values ($1, $2, $3)
			on conflict (transport, external_id) do nothing`,
			personID, seed.Transport, seed.ExternalID); err != nil {
			return 0, fmt.Errorf("seeding identity %s/%s: %w", seed.Transport, seed.ExternalID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("committing seed: %w", err)
	}
	return personID, nil
}

// ResolvePerson runs in the drain loop, never on the request path — a person
// lookup is a database read, and the request path not touching Postgres is
// what makes an outage survivable.
//
// An unknown identity is nil. It is never created: auto-vivifying a person on
// first sight would quietly re-admit anyone the guard had just turned away.
func (s *Store) ResolvePerson(ctx context.Context, transport string, externalID *string) (*int64, error) {
	if externalID == nil {
		return nil, nil
	}

	var personID int64
	err := s.pool.QueryRow(ctx, `
		select person_id from identities
		where transport = $1 and external_id = $2
		limit 1`, transport, *externalID).Scan(&personID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolving person: %w", err)
	}
	return &personID, nil
}
