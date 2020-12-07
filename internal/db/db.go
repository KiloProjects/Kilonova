package db

import (
	"context"
	"database/sql"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
)

var GlobalDB *DB

type DB struct {
	conn *sql.DB
	raw  *rawdb.Queries
}

// Raw should be deleted when we won't need to ever use rawdb in the platform
func (d *DB) Raw() *rawdb.Queries {
	return d.raw
}

func New(dsn string) (*DB, error) {
	rawConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := rawConn.Ping(); err != nil {
		return nil, err
	}

	conn, err := rawdb.Prepare(context.Background(), rawConn)
	if err != nil {
		return nil, err
	}

	db := &DB{rawConn, conn}

	if GlobalDB == nil {
		GlobalDB = db
	}
	return db, nil
}
