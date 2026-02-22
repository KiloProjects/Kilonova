package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type SquirrelQuery interface {
	ToSql() (string, []any, error)
}

func (db *DB) Query(ctx context.Context, qb SquirrelQuery) (pgx.Rows, error) {
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}
	return db.conn.Query(ctx, sql, args...)
}

func (db *DB) QueryRow(ctx context.Context, qb SquirrelQuery) pgx.Row {
	sql, args, err := qb.ToSql()
	if err != nil {
		panic("invalid squirrel query: " + err.Error())
	}
	return db.conn.QueryRow(ctx, sql, args...)
}
