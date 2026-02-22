package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// Inspired by https://github.com/miniflux/v2/blob/main/internal/database/database.go

type Migration struct {
	ID      int
	Name    string
	Handler func(ctx context.Context, tx pgx.Tx) error
}

type MigrationConfig struct {
	SchemaTable       string
	Migrations        []Migration
	SpecialMigrations []Migration
}

func (cfg *MigrationConfig) SanityCheck() error {
	if cfg.SchemaTable == "" {
		return errors.New("schema table is required")
	}
	for i := range cfg.Migrations {
		if i > 0 && cfg.Migrations[i].ID <= cfg.Migrations[i-1].ID {
			return errors.New("migrations must be strictly ascending by ID")
		}
	}
	return nil
}

func RunMigrations(ctx context.Context, db *DB, config MigrationConfig) error {
	if err := config.SanityCheck(); err != nil {
		return err
	}

	var databaseSchema int
	var runSpecial bool
	db.QueryRow(ctx, sq.Select("version").From(config.SchemaTable)).Scan(&databaseSchema)
	slog.DebugContext(ctx, "Checked DB schema", slog.Int("version", databaseSchema))
	for _, mig := range config.Migrations {
		if mig.ID <= databaseSchema {
			continue
		}
		runSpecial = true
		slog.InfoContext(
			ctx, "Executing migration",
			slog.Int("migration_id", mig.ID),
			slog.String("name", mig.Name),
		)

		if err := pgx.BeginFunc(ctx, db.conn, func(tx pgx.Tx) error {
			if err := mig.Handler(ctx, tx); err != nil {
				return err
			}

			sql, args := sq.Delete(config.SchemaTable).MustSql()
			if _, err := tx.Exec(ctx, sql, args...); err != nil {
				return fmt.Errorf("could not clear schema version: %w", err)
			}

			sql, args = sq.Insert(config.SchemaTable).Columns("version").Values(mig.ID).MustSql()
			if _, err := tx.Exec(ctx, sql, args...); err != nil {
				return fmt.Errorf("could not update schema version: %w", err)
			}

			return nil
		}); err != nil {
			return err
		}

	}

	if runSpecial {
		for _, mig := range config.SpecialMigrations {
			pgx.BeginFunc(ctx, db.conn, func(tx pgx.Tx) error {
				return mig.Handler(ctx, tx)
			})
		}
	}

	return nil
}
