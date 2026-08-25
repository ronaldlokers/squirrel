package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Who is signed in.
//
// See migrations/0029_sessions.sql for why the table holds a hash rather than
// the token, and why the sub travels with the session.

// Session is what a cookie resolves to.
type Session struct {
	PersonID int64
	// Sub is the OIDC subject. The capture path writes it as a sender string
	// so the drain can resolve a spooled capture's owner — see the design's
	// section 4.
	Sub       string
	ExpiresAt time.Time
}

// OpenSession records a login.
func (s *Store) OpenSession(ctx context.Context, personID int64, sub string, token []byte, at time.Time, life time.Duration) error {
	_, err := s.pool.Exec(ctx, `
		insert into sessions (person_id, sub, token_sha256, created_at, seen_at, expires_at)
		values ($1, $2, $3, $4, $4, $5)`,
		personID, sub, token, at, at.Add(life))
	if err != nil {
		return fmt.Errorf("opening a session: %w", err)
	}
	return nil
}

// SessionFor resolves a cookie, and pushes the expiry out while it is at it.
//
// One statement rather than a read and then a write, because the two are the
// same fact: a session that resolves has just been used. Splitting them would
// mean a request that read a session and then failed to say so, and a session
// in daily use quietly expiring from when it was opened.
//
// Expiry is in the where clause rather than checked by the caller, so there is
// no call site that can forget it.
func (s *Store) SessionFor(ctx context.Context, token []byte, at time.Time) (Session, bool, error) {
	var out Session
	err := s.pool.QueryRow(ctx, `
		update sessions
		   set seen_at = $2, expires_at = $2 + (expires_at - seen_at)
		 where token_sha256 = $1 and expires_at > $2
		 returning person_id, sub, expires_at`, token, at).
		Scan(&out.PersonID, &out.Sub, &out.ExpiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Unknown, or expired. They are the same answer on purpose: telling a
		// caller which would be telling a stranger that a token was once real.
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, fmt.Errorf("reading a session: %w", err)
	}
	out.ExpiresAt = s.here(out.ExpiresAt)
	return out, true, nil
}

// EndSession is signing out. An unknown token is not an error: signing out
// twice is a thing somebody does, and the second one must not be a failure
// page.
func (s *Store) EndSession(ctx context.Context, token []byte) error {
	if _, err := s.pool.Exec(ctx,
		`delete from sessions where token_sha256 = $1`, token); err != nil {
		return fmt.Errorf("ending a session: %w", err)
	}
	return nil
}

// ReapSessions removes what has expired, so the table does not grow forever.
// Nothing depends on it having run, because SessionFor refuses an expired row
// anyway.
func (s *Store) ReapSessions(ctx context.Context, at time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `delete from sessions where expires_at <= $1`, at)
	if err != nil {
		return 0, fmt.Errorf("reaping sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
