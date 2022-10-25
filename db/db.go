package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"

	"github.com/jmoiron/sqlx"
	//_ "github.com/jackc/pgx/v4/stdlib"
)

type DB struct {
	conn    *sqlx.DB
	pgxConf *pgx.ConnConfig

	listener *NotifyListener
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func NewPSQL(ctx context.Context, dsn string) (*DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	//config.Logger = &logger{}
	dsn = stdlib.RegisterConnConfig(config)

	conn, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(20)

	listener, err := NewListener(ctx, config)
	if err != nil {
		return nil, err
	}

	return &DB{conn, config, listener}, nil
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

// type logger struct{}

// // Log a message at the given level with data key/value pairs. data may be nil.
// func (l *logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]any) {
// 	log.Println(level, msg, data)
// }

func mapper[T1 any, T2 any](lst []*T1, f func(*T1) *T2) []*T2 {
	if len(lst) == 0 {
		return []*T2{}
	}
	rez := make([]*T2, len(lst))
	for i := range rez {
		rez[i] = f(lst[i])
	}
	return rez
}
