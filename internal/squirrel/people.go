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
// It creates and ResolvePerson deliberately does not: auto-vivifying a person on
// first sight would once have re-admitted anyone the guard turned away. Authentik's
// application binding is the gate now, and it can change without a deploy.
// ResolvePerson keeps its behaviour, because the drain still needs a lookup that
// refuses strangers.
//
// Two identities, always. A capture typed on the screen goes through the spool
// with a sender string and the drain resolves its owner from that, so a person
// with only an oidc identity would spool notes belonging to nobody. Both keyed by
// the sub, the only thing about an account that does not change.
//
// The handle is a display name and is never matched on: two accounts can share
// one, and usernames get reassigned.
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

// Whom is what the screen says about you: a name to show and whether there is
// a picture to show beside it. Not an identity — see the migration.
type Whom struct {
	Name    string
	Handle  string
	HasFace bool
}

// WhoIs is the display name, falling back to the handle. Both may be empty on
// a person who signed in before there was anything to ask for.
func (s *Store) WhoIs(ctx context.Context, personID int64) (Whom, error) {
	var out Whom
	var name *string
	var face []byte
	err := s.pool.QueryRow(ctx,
		`select display_name, handle, face from people where id = $1`,
		personID).Scan(&name, &out.Handle, &face)
	if err != nil {
		return Whom{}, fmt.Errorf("reading who you are: %w", err)
	}
	if name != nil {
		out.Name = *name
	}
	// Deliberately no fall back to the handle. handleFor appends a hash of the
	// sub to make the column unique, so the stored handle reads
	// "ronald-cf1cab94" — legible to somebody reading rows, not a name to put
	// on a screen. A person who has not signed in since display names existed
	// gets a monogram and no name until they next do.
	out.HasFace = len(face) > 0
	return out, nil
}

// PersonFace is the stored picture and its type.
func (s *Store) PersonFace(ctx context.Context, personID int64) ([]byte, string, bool, error) {
	var face []byte
	var kind *string
	err := s.pool.QueryRow(ctx,
		`select face, face_type from people where id = $1`, personID).Scan(&face, &kind)
	if err != nil {
		return nil, "", false, fmt.Errorf("reading your picture: %w", err)
	}
	if len(face) == 0 {
		return nil, "", false, nil
	}
	out := ""
	if kind != nil {
		out = *kind
	}
	return face, out, true, nil
}

// RememberPerson keeps what the gate said about you this time.
//
// A missing name or picture never erases one already held: the provider may
// stop sending a claim it once sent, and losing your face because Authentik
// was reconfigured is not something to store.
func (s *Store) RememberPerson(ctx context.Context, personID int64, name string, face []byte, faceType string) error {
	_, err := s.pool.Exec(ctx, `
		update people set
		  display_name = coalesce(nullif($2, ''), display_name),
		  face         = case when $3::bytea is null then face else $3 end,
		  face_type    = case when $3::bytea is null then face_type else nullif($4, '') end
		 where id = $1`,
		personID, name, face, faceType)
	if err != nil {
		return fmt.Errorf("remembering who you are: %w", err)
	}
	return nil
}
