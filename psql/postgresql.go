package psql

import (
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
	// Get the DB driver
	_ "github.com/lib/pq"
)

var _ kilonova.TypeServicer = &DB{}

type DB struct {
	conn *sqlx.DB
}

func (db *DB) UserService() kilonova.UserService {
	return &UserService{db.conn}
}

func (db *DB) ProblemService() kilonova.ProblemService {
	return &ProblemService{db.conn}
}

func (db *DB) SubmissionService() kilonova.SubmissionService {
	return &SubmissionService{db.conn}
}

func (db *DB) SubTestService() kilonova.SubTestService {
	return &SubTestService{db.conn}
}

func (db *DB) TestService() kilonova.TestService {
	return &TestService{db.conn}
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func New(dsn string) (*DB, error) {
	conn, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
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
