package store

import (
	"context"
	"embed"
	"fmt"
	"sort"
)

//go:embed migrations/*.up.sql
var migrationFS embed.FS

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.exec(ctx, s.db, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		sqlBytes, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.begin(ctx)
		if err != nil {
			return err
		}
		tag, err := s.exec(ctx, tx, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, name)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		claimed, err := tag.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if claimed == 0 {
			_ = tx.Rollback()
			continue
		}
		if _, err := s.exec(ctx, tx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
