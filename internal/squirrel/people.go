package squirrel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// OIDCTransport is a login. It is a transport like any other, which is what
// lets one person be reached by chat, by the screen and by a browser session
// without any of the three knowing about the others.
const OIDCTransport = "oidc"

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

// PersonForLogin is who just signed in, creating them if this is the first time.
//
// It creates, and ResolvePerson deliberately does not. That difference is the
// whole of the change on 25 August 2026, and the rule being overturned is
// worth quoting rather than deleting:
//
//	An unknown identity is nil. It is never created: auto-vivifying a person
//	on first sight would quietly re-admit anyone the guard had just turned
//	away.
//
// That was correct when the guard was the only gate. Authentik's application
// binding is the gate now, and it is a better one: it can be changed without a
// deploy. ResolvePerson keeps its behaviour and its comment, because the drain
// still needs a lookup that refuses strangers.
//
// Two identities, always. A capture typed on the screen goes through the spool
// with a sender string, and the drain resolves its owner from that — so a
// person with only an oidc identity would spool notes belonging to nobody.
// Both are keyed by the sub, because the sub is the only thing about an
// account that does not change.
//
// The handle is a display name and is never matched on. Two Authentik accounts
// can share one, and usernames get reassigned; that is what the sub is for.
func (s *Store) PersonForLogin(ctx context.Context, sub, handle string) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolving a login: %w", err)
	}
	defer tx.Rollback(ctx)

	var personID int64
	err = tx.QueryRow(ctx, `
		select person_id from identities
		where transport = $1 and external_id = $2
		limit 1`, OIDCTransport, sub).Scan(&personID)

	switch {
	case err == nil:
		// Known. Nothing to write, and deliberately not even the handle: a
		// person renaming themselves in Authentik is not a reason to rewrite
		// rows here.
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("resolving a login: %w", err)
		}
		return personID, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return 0, fmt.Errorf("resolving a login: %w", err)
	}

	if err := tx.QueryRow(ctx, `
		insert into people (handle) values ($1) returning id`, handleFor(sub, handle)).Scan(&personID); err != nil {
		return 0, fmt.Errorf("making a person: %w", err)
	}
	for _, transport := range []string{OIDCTransport, ScreenTransport} {
		if _, err := tx.Exec(ctx, `
			insert into identities (person_id, transport, external_id)
			values ($1, $2, $3)
			on conflict (transport, external_id) do nothing`,
			personID, transport, sub); err != nil {
			return 0, fmt.Errorf("giving them an identity: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("resolving a login: %w", err)
	}
	return personID, nil
}

// handleFor is a display name that is unique without being an opaque
// identifier. handle is a unique column and two Authentik accounts may share a
// username, so the sub is what makes one handle different from another — but
// only eight characters of it, because the whole thing on a screen is noise.
func handleFor(sub, handle string) string {
	if handle == "" {
		handle = "someone"
	}
	sum := sha256.Sum256([]byte(sub))
	return handle + "-" + hex.EncodeToString(sum[:])[:8]
}
