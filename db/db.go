package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var _ kilonova.DB = &DB{}

type DB struct {
	conn *sqlx.DB
}

func (d *DB) Close() error {
	return d.conn.Close()
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
