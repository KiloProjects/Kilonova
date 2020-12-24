package db

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
	"github.com/KiloProjects/Kilonova/internal/rclient"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type logger struct{}

func (l logger) Log(ctx context.Context, loglevel pgx.LogLevel, msg string, data map[string]interface{}) {
	t := data["time"]
	if t == nil {
		return
	}

	if t.(time.Duration) > 10*time.Millisecond {
		s := data["sql"].(string)
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "--") {
			s = strings.SplitN(s, "\n", 2)[1]
		}
		s = strings.ReplaceAll(s, "\n", " ")
		s = strings.ReplaceAll(s, "  ", " ")
		log.Printf("SLOW %s: %s: %s time=%s\n", loglevel, msg, s, t)
	}
}

var GlobalDB *DB

type DB struct {
	cache *rclient.RClient
	conn  *pgx.Conn
	// TODO: Phase out rawdb and dbconn
	dbconn *sqlx.DB
	raw    *rawdb.Queries
}

func (d *DB) Close() error {
	if err := stdlib.ReleaseConn(d.dbconn.DB, d.conn); err != nil {
		return err
	}

	return d.dbconn.Close()
}

func New(dsn string, cache *rclient.RClient) (*DB, error) {

	connConf, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	connConf.Logger = logger{}
	cst := stdlib.RegisterConnConfig(connConf)

	rawConn, err := sqlx.Connect("pgx", cst)
	if err != nil {
		return nil, err
	}

	conn := rawdb.New(rawConn)

	pg, err := stdlib.AcquireConn(rawConn.DB)
	if err != nil {
		return nil, err
	}

	db := &DB{cache, pg, rawConn, conn}

	if GlobalDB == nil {
		GlobalDB = db
	}
	return db, nil
}
