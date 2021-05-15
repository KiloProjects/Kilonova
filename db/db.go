package db

import (
	"context"
	"embed"
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

var _ kilonova.TypeServicer = &DB{}

type DB struct {
	conn *sqlx.DB
}

func (d *DB) UserService() kilonova.UserService {
	return NewUserService(d.conn)
}

func (d *DB) ProblemService() kilonova.ProblemService {
	return NewProblemService(d.conn)
}

func (d *DB) TestService() kilonova.TestService {
	return NewTestService(d.conn)
}

func (d *DB) SubmissionService() kilonova.SubmissionService {
	return NewSubmissionService(d.conn)
}

func (d *DB) SubTestService() kilonova.SubTestService {
	return NewSubTestService(d.conn)
}

func (d *DB) ProblemListService() kilonova.ProblemListService {
	return NewProblemListService(d.conn)
}

func (d *DB) SubTaskService() kilonova.SubTaskService {
	return NewSubTaskService(d.conn)
}

func (d *DB) VerificationService() kilonova.Verificationer {
	return NewVerificationService(d.conn)
}

func (d *DB) SessionService() kilonova.Sessioner {
	return NewSessionService(d.conn)
}

func (d *DB) AttachmentService() kilonova.AttachmentService {
	return NewAttachmentService(d.conn)
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
	for _, elem := range elems {
		data, err := fs.ReadFile(dir, elem.Name())
		if err != nil {
			return err
		}
		if _, err := d.conn.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("%s: %w", elem.Name(), err)
		}
	}
	return nil
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

func AppropriateDB(ctx context.Context, conf config.DBConf) (kilonova.TypeServicer, error) {
	if conf.Type == "postgres" {
		return NewPSQL(config.Database.DSN)
	} else {
		return NewSQLite(ctx, config.Database.DSN)
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
