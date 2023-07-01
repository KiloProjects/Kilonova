package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

func (s *DB) CreateTest(ctx context.Context, test *kilonova.Test) error {
	if test.ProblemID == 0 {
		return kilonova.ErrMissingRequired
	}

	var id int
	err := s.pgconn.QueryRow(ctx, "INSERT INTO tests (score, problem_id, visible_id) VALUES ($1, $2, $3) RETURNING id", test.Score, test.ProblemID, test.VisibleID).Scan(&id)
	if err == nil {
		test.ID = id
	}
	return err
}

func (s *DB) Test(ctx context.Context, pbID, testVID int) (*kilonova.Test, error) {
	var test kilonova.Test
	err := s.conn.GetContext(ctx, &test, "SELECT * FROM tests WHERE problem_id = $1 AND visible_id = $2 ORDER BY visible_id LIMIT 1", pbID, testVID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &test, err
}

func (s *DB) Tests(ctx context.Context, pbID int) ([]*kilonova.Test, error) {
	var tests []*kilonova.Test
	err := s.conn.SelectContext(ctx, &tests, "SELECT * FROM tests WHERE problem_id = $1 ORDER BY visible_id", pbID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Test{}, nil
	}
	return tests, err
}

func (s *DB) UpdateTest(ctx context.Context, id int, upd kilonova.TestUpdate) error {
	ub := newUpdateBuilder()
	if v := upd.Score; v != nil {
		ub.AddUpdate("score = %s", v)
	}
	if v := upd.VisibleID; v != nil {
		ub.AddUpdate("visible_id = %s", v)
	}
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)

	_, err := s.conn.ExecContext(ctx, "UPDATE tests SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) DeleteProblemTests(ctx context.Context, problemID int) ([]int, error) {
	rows, _ := s.pgconn.Query(ctx, "DELETE FROM tests WHERE problem_id = $1 RETURNING id", problemID)
	vals, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []int{}, nil
		}
		return nil, err
	}
	return vals, err
}

func (s *DB) DeleteTest(ctx context.Context, id int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM tests WHERE id = $1", id)
	if err != nil {
		return err
	}
	return err
}

func (s *DB) BiggestVID(ctx context.Context, problemID int) (int, error) {
	var id int
	err := s.pgconn.QueryRow(ctx, "SELECT visible_id FROM tests WHERE problem_id = $1 ORDER BY visible_id DESC LIMIT 1", problemID).Scan(&id)
	return id, err
}
