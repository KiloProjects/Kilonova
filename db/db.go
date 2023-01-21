package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type DB struct {
	conn *sqlx.DB
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func NewPSQL(ctx context.Context, dsn string) (*DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	//config.Tracer = &tracelog.TraceLog{Logger: tracelog.LoggerFunc(log), LogLevel: tracelog.LogLevelDebug}
	dsn = stdlib.RegisterConnConfig(config)

	conn, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(40)

	return &DB{conn}, nil
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

func mapper[T1 any, T2 any](lst []T1, f func(T1) T2) []T2 {
	if len(lst) == 0 {
		return []T2{}
	}
	rez := make([]T2, len(lst))
	for i := range rez {
		rez[i] = f(lst[i])
	}
	return rez
}

func mapperCtx[T1 any, T2 any](ctx context.Context, lst []T1, f func(context.Context, T1) (T2, error)) []T2 {
	if len(lst) == 0 {
		return []T2{}
	}
	rez := make([]T2, len(lst))
	for i := range rez {
		var err error
		rez[i], err = f(ctx, lst[i])
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().WithOptions(zap.AddCallerSkip(1)).Warn(err)
		}
	}
	return rez
}

func log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	logger := zap.L()
	fields := make([]zapcore.Field, len(data))
	i := 0
	for k, v := range data {
		fields[i] = zap.Any(k, v)
		i++
	}

	switch level {
	case tracelog.LogLevelTrace:
		logger.Debug(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	case tracelog.LogLevelDebug:
		logger.Debug(msg, fields...)
	case tracelog.LogLevelInfo:
		logger.Info(msg, fields...)
	case tracelog.LogLevelWarn:
		logger.Warn(msg, fields...)
	case tracelog.LogLevelError:
		logger.Error(msg, fields...)
	default:
		logger.Error(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	}
}
