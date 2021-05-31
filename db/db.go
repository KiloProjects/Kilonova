package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed sqlite_schema
var sqliteSchema embed.FS

var _ kilonova.DB = &DB{}

type DB struct {
	conn *sqlx.DB
}

func (d *DB) Close() error {
	return d.conn.Close()
}

type SQLiteDB struct {
	DB
}

// initDB runs all sql in the schema directory
func (d *SQLiteDB) initDB(ctx context.Context, dir fs.FS) error {
	elems, err := fs.ReadDir(dir, ".")
	if err != nil {
		return err
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, elem := range elems {
		data, err := fs.ReadFile(dir, elem.Name())
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("%s: %w", elem.Name(), err)
		}
	}
	return tx.Commit()
}

func NewSQLite(ctx context.Context, filename string) (*SQLiteDB, error) {
	conn, err := sqlx.Connect("sqlite3", "file:"+filename+"?_fk=on&cache=shared")
	if err != nil {
		return nil, err
	}
	db := &SQLiteDB{DB{conn}}
	subbed, err := fs.Sub(sqliteSchema, "sqlite_schema")
	if err != nil {
		return nil, err
	}
	if err := db.initDB(ctx, subbed); err != nil {
		return nil, err
	}
	return db, nil
}

func NewPSQL(dsn string) (*DB, error) {
	conn, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &DB{conn}, nil
}

func AppropriateDB(ctx context.Context, conf config.DBConf) (kilonova.DB, error) {
	if conf.Type == "postgres" {
		return NewPSQL(config.Database.DSN)
	} else if conf.Type == "sqlite" {
		return NewSQLite(ctx, config.Database.DSN)
	} else {
		return nil, errors.New("invalid DB type")
	}
}

func FormatLimitOffset(limit int, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	}

	if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	}

	if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}

	return ""
}
