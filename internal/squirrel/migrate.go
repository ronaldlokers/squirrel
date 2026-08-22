package squirrel

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// ErrMigration marks a failure that is the migration's own fault rather than
// the database being out of reach. The two retry identically — nothing may
// block a capture being accepted — but they are diagnosed nothing alike, and
// they used to be logged with the same words.
var ErrMigration = errors.New("a migration will not apply")

// IsMigrationFailure reports whether an error from Migrate is the migration's
// own doing. Boot uses it to say which of the two things has happened.
func IsMigrationFailure(err error) bool { return errors.Is(err, ErrMigration) }

// Migrate applies every embedded migration not yet recorded, each inside a
// transaction that also records it. Migrations ship in the binary, so there is
// no directory to copy into the image and no generator to be surprised by.
func (s *Store) Migrate(ctx context.Context) error {
	return s.migrateFrom(ctx, migrationFiles)
}

// migrateFrom is Migrate against a given set of files, so a test can hand it a
// migration that will not apply and watch what comes back. The shipped set is
// embedded and cannot be made to fail on purpose, which left the one error
// path here provable only by writing the error out by hand and asserting on
// the thing the test had just written — which proves nothing.
func (s *Store) migrateFrom(ctx context.Context, files fs.FS) error {
	const createTable = `
		create table if not exists schema_migrations (
			version    text primary key,
			applied_at timestamptz not null default now()
		)`
	if _, err := s.pool.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	entries, err := fs.Glob(files, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("listing migrations: %w", err)
	}
	sort.Strings(entries)

	for _, name := range entries {
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`select exists (select 1 from schema_migrations where version = $1)`, name,
		).Scan(&exists); err != nil {
			return fmt.Errorf("checking migration %s: %w", name, err)
		}
		if exists {
			continue
		}

		body, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("%w: applying %s: %w", ErrMigration, name, err)
		}
		if _, err := tx.Exec(ctx,
			`insert into schema_migrations (version) values ($1)`, name,
		); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("%w: recording %s: %w", ErrMigration, name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing migration %s: %w", name, err)
		}
		slog.Info("migration applied", "version", name)
	}
	return nil
}
