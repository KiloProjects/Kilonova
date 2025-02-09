package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"path"

	"github.com/jackc/pgx/v5"
)

// Inspired by https://github.com/miniflux/v2/blob/main/internal/database/database.go

//go:embed psql_schema
var migrationFiles embed.FS

type migration struct {
	id      int
	name    string
	handler func(ctx context.Context, tx pgx.Tx) error
}

// Order is important. Add new migrations at the end of the list.
var migrations = []migration{
	{
		id:      1,
		name:    "Base schema",
		handler: runFile("001.base.sql"),
	},
	{
		id:      2,
		name:    "Discord Integration",
		handler: runFile("002.discord_oauth_secret.sql"),
	},
	{
		id:      3,
		name:    "Discord Avatar as main",
		handler: runFile("003.use_discord_avatar.sql"),
	},
	{
		id:      4,
		name:    "Add Stripe support to donations workflow",
		handler: runFile("004.stripe.sql"),
	},
	{
		id:      5,
		name:    "External resources support",
		handler: runFile("005.external_resources.sql"),
	},
}

var specialMigrations = []migration{
	{
		id:      999,
		name:    "Views",
		handler: runFile("999.views.sql"),
	},
}

func (s *DB) RunMigrations(ctx context.Context) error {
	if err := s.checkLegacy(ctx); err != nil {
		return err
	}
	var databaseSchema int
	var runSpecial bool
	s.conn.QueryRow(ctx, "SELECT version FROM kn_schema_version").Scan(&databaseSchema)
	slog.DebugContext(ctx, "Checked DB schema", slog.Int("version", databaseSchema))
	for _, mig := range migrations {
		if mig.id <= databaseSchema {
			continue
		}
		runSpecial = true
		slog.InfoContext(ctx, "Executing migration", slog.Int("migration_id", mig.id))

		if err := pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
			if err := mig.handler(ctx, tx); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM kn_schema_version`); err != nil {
				return fmt.Errorf("could not clear schema version: %w", err)
			}

			if _, err := tx.Exec(ctx, `INSERT INTO kn_schema_version (version) VALUES ($1)`, mig.id); err != nil {
				return fmt.Errorf("could not update schema version: %w", err)
			}

			return nil
		}); err != nil {
			return err
		}

	}

	if runSpecial {
		for _, mig := range specialMigrations {
			pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
				return mig.handler(ctx, tx)
			})
		}
	}

	return nil
}

// Check if database was created before current migration system
// If so, set database version to 1 to skip base
// The `max_score_view` view is the discriminator, since it's been removed in the current psql_schema/
func (s *DB) checkLegacy(ctx context.Context) error {
	var schemaVersion int
	s.conn.QueryRow(ctx, "SELECT version FROM kn_schema_version").Scan(&schemaVersion)
	var hasMSV bool
	if err := s.conn.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'max_score_view');").Scan(&hasMSV); err != nil {
		return err
	}
	if !hasMSV || schemaVersion > 0 {
		return nil
	}
	slog.InfoContext(ctx, "Legacy database detected. Initialized as schema version 1")
	_, err := s.conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS kn_schema_version (
			version integer not null
		);
		INSERT INTO kn_schema_version (version) VALUES (1);
	`)
	return err
}

func runFile(name string) func(ctx context.Context, tx pgx.Tx) error {
	return func(ctx context.Context, tx pgx.Tx) error {
		f, err := migrationFiles.ReadFile(path.Join("psql_schema", name))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, string(f))
		return err
	}
}
