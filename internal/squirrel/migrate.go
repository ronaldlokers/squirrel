package squirrel

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate applies every embedded migration not yet recorded, each inside a
// transaction that also records it. Migrations ship in the binary, so there is
// no directory to copy into the image and no generator to be surprised by.
func (s *Store) Migrate(ctx context.Context) error {
	const createTable = `
		create table if not exists schema_migrations (
			version    text primary key,
			applied_at timestamptz not null default now()
		)`
	if _, err := s.pool.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
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

		body, err := migrationFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("applying migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			`insert into schema_migrations (version) values ($1)`, name,
		); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("recording migration %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing migration %s: %w", name, err)
		}
		slog.Info("migration applied", "version", name)
	}
	return nil
}
