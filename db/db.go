package db

import (
	"context"
	"fmt"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/multitracer"
	"log/slog"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"gopkg.in/natefinch/lumberjack.v2"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
)

var (
	loggerOnce sync.Once
	dbLogger   *slog.Logger
)

type dbCtx string

const (
	queryCount = dbCtx("queryCount")
)

var (
	LogQueries   = config.GenFlag[bool]("behavior.db.log_sql", false, "Log SQL Requests (for debugging purposes)")
	CountQueries = config.GenFlag[bool]("behavior.db.count_queries", false, "Count SQL Queries (for debugging purposes)")
)

type DB struct {
	conn *pgxpool.Pool
}

func (s *DB) Close() error {
	s.conn.Close()
	return nil
}

// Deprecated: It's just a placeholder for old behavior
func Get[T any](pgconn *pgxpool.Pool, ctx context.Context, dest *T, query string, args ...any) error {
	rows, _ := pgconn.Query(ctx, query, args...)
	val, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[T])
	if err != nil {
		return err
	}
	*dest = val
	return nil
}

// Deprecated: It's just a placeholder for old behavior
func Select[T any](pgconn *pgxpool.Pool, ctx context.Context, dest *[]*T, query string, args ...any) error {
	rows, _ := pgconn.Query(ctx, query, args...)
	vals, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByNameLax[T])
	if err != nil {
		return err
	}
	*dest = vals
	return nil
}

func NewPSQL(ctx context.Context, dsn string) (*DB, error) {
	pgconf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pgconf.ConnConfig.Tracer = multitracer.New(
		&tracelog.TraceLog{Logger: tracelog.LoggerFunc(log), LogLevel: tracelog.LogLevelDebug},
		otelpgx.NewTracer(otelpgx.WithTrimSQLInSpanName()),
	)

	pgconf.MaxConns = 40
	pgconf.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		pgxdecimal.Register(c.TypeMap())
		return nil
	}

	pgconn, err := pgxpool.NewWithConfig(ctx, pgconf)
	if err != nil {
		return nil, err
	}

	return &DB{pgconn}, nil
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
		if err != nil {
			slog.WarnContext(ctx, "Error running mapper", slog.Any("err", err))
		}
	}
	return rez
}

func toSingular[T1, T2 any](ctx context.Context, filter T1, f func(ctx context.Context, filter T1) ([]*T2, error)) (*T2, error) {
	many, err := f(ctx, filter)
	if err != nil || len(many) == 0 {
		return nil, err
	}
	return many[0], nil
}

func InitContextCounter(rootCtx context.Context) context.Context {
	return context.WithValue(rootCtx, queryCount, &atomic.Int64{})
}

func GetContextQueryCount(ctx context.Context) int64 {
	cnt, ok := ctx.Value(queryCount).(*atomic.Int64)
	if !ok {
		return -1
	}
	return cnt.Load()
}

func log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	loggerOnce.Do(func() {
		lvl := slog.LevelInfo
		if config.Common.Debug {
			lvl = slog.LevelDebug
		}
		dbLogger = slog.New(slog.NewJSONHandler(&lumberjack.Logger{
			Filename: path.Join(config.Common.LogDir, "db.log"),
			MaxSize:  200, // MB
			Compress: true,
		}, &slog.HandlerOptions{
			Level: lvl,
		}))
	})

	if msg == "Prepare" {
		return
	}

	dur, ok := data["time"].(time.Duration)
	if ok {
		if dur > 1*time.Second {
			dbLogger.WarnContext(ctx, "Really slow operation", slog.Duration("duration", dur), slog.Any("query", data["sql"]), slog.Any("args", data["args"]))
		}
	} else {
		slog.WarnContext(ctx, "DB time is not duration", slog.Any("time", data["time"]))
	}

	if CountQueries.Value() {
		if v, ok := ctx.Value(queryCount).(*atomic.Int64); ok {
			v.Add(1)
		}
	}

	if LogQueries.Value() {
		fields := make([]slog.Attr, 0, len(data))
		for k, v := range data {
			fields = append(fields, slog.Any(k, v))
		}

		var lvl slog.Level
		switch level {
		case tracelog.LogLevelTrace:
			lvl = slog.LevelDebug - 1
			fields = append(fields, slog.Any("PGX_LOG_LEVEL", level))
		case tracelog.LogLevelDebug:
			lvl = slog.LevelDebug
		case tracelog.LogLevelInfo:
			lvl = slog.LevelInfo
		case tracelog.LogLevelWarn:
			lvl = slog.LevelWarn
		case tracelog.LogLevelError:
			lvl = slog.LevelError
		default:
			lvl = slog.LevelError
			fields = append(fields, slog.Any("PGX_LOG_LEVEL", level))
		}
		dbLogger.LogAttrs(ctx, lvl, msg, fields...)
	}
}
