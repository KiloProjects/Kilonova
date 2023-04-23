package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	loggerOnce sync.Once
	loggerFile *os.File
	dbLogger   *zap.Logger
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
	config.Tracer = &tracelog.TraceLog{Logger: tracelog.LoggerFunc(log), LogLevel: tracelog.LogLevelDebug}
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
	loggerOnce.Do(func() {
		var err error
		loggerFile, err = os.OpenFile(path.Join(config.Common.LogDir, "db.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			zap.S().Fatal("Could not open db.log for writing")
		}
		dbLogger = zap.New(kilonova.GetZapCore(config.Common.Debug, false, loggerFile), zap.AddCaller())
	})

	fields := make([]zapcore.Field, len(data))
	i := 0
	for k, v := range data {
		fields[i] = zap.Any(k, v)
		i++
	}

	switch level {
	case tracelog.LogLevelTrace:
		dbLogger.Debug(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	case tracelog.LogLevelDebug:
		dbLogger.Debug(msg, fields...)
	case tracelog.LogLevelInfo:
		dbLogger.Info(msg, fields...)
	case tracelog.LogLevelWarn:
		dbLogger.Warn(msg, fields...)
	case tracelog.LogLevelError:
		dbLogger.Error(msg, fields...)
	default:
		dbLogger.Error(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	}
}
