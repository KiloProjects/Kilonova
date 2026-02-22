package postgres

import (
	"context"
	"log/slog"
	"path"
	"sync/atomic"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/exaring/otelpgx"
	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	DSN string

	CountQueries bool
	LogQueries   bool
}

type DB struct {
	conn   *pgxpool.Pool
	logger *slog.Logger

	countQueries bool
	logQueries   bool
}

func (db *DB) Close() error {
	db.conn.Close()
	return nil
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.conn
}

func (db *DB) initLogger() {
	lvl := slog.LevelInfo
	if config.Common.Debug {
		lvl = slog.LevelDebug
	}
	db.logger = slog.New(slog.NewJSONHandler(&lumberjack.Logger{
		Filename: path.Join(config.Common.LogDir, "db.log"),
		MaxSize:  200, // MB
		Compress: true,
	}, &slog.HandlerOptions{
		Level: lvl,
	}))
}

func NewDB(ctx context.Context, conf Config) (*DB, error) {
	db := &DB{
		countQueries: conf.CountQueries,
		logQueries:   conf.LogQueries,
	}
	db.initLogger()

	pgconf, err := pgxpool.ParseConfig(conf.DSN)
	if err != nil {
		return nil, err
	}
	pgconf.ConnConfig.Tracer = multitracer.New(
		otelpgx.NewTracer(otelpgx.WithTrimSQLInSpanName()),
		&tracelog.TraceLog{Logger: tracelog.LoggerFunc(db.log), LogLevel: tracelog.LogLevelDebug},
	)

	pgconf.MaxConns = 40
	pgconf.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		pgxdecimal.Register(c.TypeMap())
		return nil
	}

	db.conn, err = pgxpool.NewWithConfig(ctx, pgconf)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	if msg == "Prepare" {
		return
	}

	dur, ok := data["time"].(time.Duration)
	if ok {
		if dur > 1*time.Second {
			db.logger.WarnContext(ctx, "Really slow operation", slog.Duration("duration", dur), slog.Any("query", data["sql"]), slog.Any("args", data["args"]))
		}
	}

	if db.countQueries {
		if v, ok := ctx.Value(queryCount).(*atomic.Int64); ok {
			v.Add(1)
		}
	}

	if db.logQueries {
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
		db.logger.LogAttrs(ctx, lvl, msg, fields...)
	}
}
