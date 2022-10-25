package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) CreateTest(ctx context.Context, test *kilonova.Test) error {
	if test.ProblemID == 0 || test.Score == 0 {
		return kilonova.ErrMissingRequired
	}

	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind("INSERT INTO tests (score, problem_id, visible_id, orphaned) VALUES (?, ?, ?, ?) RETURNING id"), test.Score, test.ProblemID, test.VisibleID, test.Orphaned)
	if err == nil {
		test.ID = id
	}
	return err
}

func (s *DB) Test(ctx context.Context, pbID, testVID int) (*kilonova.Test, error) {
	var test kilonova.Test
	err := s.conn.GetContext(ctx, &test, s.conn.Rebind("SELECT * FROM tests WHERE problem_id = ? AND visible_id = ? AND orphaned = false ORDER BY visible_id LIMIT 1"), pbID, testVID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &test, err
}

func (s *DB) TestByID(ctx context.Context, id int) (*kilonova.Test, error) {
	var test kilonova.Test
	err := s.conn.GetContext(ctx, &test, s.conn.Rebind("SELECT * FROM tests WHERE id = ? LIMIT 1"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &test, err
}

func (s *DB) Tests(ctx context.Context, pbID int) ([]*kilonova.Test, error) {
	var tests []*kilonova.Test
	err := s.conn.SelectContext(ctx, &tests, s.conn.Rebind("SELECT * FROM tests WHERE problem_id = ? AND orphaned = false ORDER BY visible_id"), pbID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Test{}, nil
	}
	return tests, err
}

func (s *DB) UpdateTest(ctx context.Context, id int, upd kilonova.TestUpdate) error {
	toUpd, args := []string{}, []any{}
	if v := upd.Score; v != nil {
		toUpd, args = append(toUpd, "score = ?"), append(args, v)
	}
	if v := upd.VisibleID; v != nil {
		toUpd, args = append(toUpd, "visible_id = ?"), append(args, v)
	}
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)

	query := s.conn.Rebind("UPDATE tests SET " + strings.Join(toUpd, ", ") + " WHERE id = ?")
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

// TODO: Test subtask stuff actually works
func (s *DB) OrphanProblemTests(ctx context.Context, problemID int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("UPDATE tests SET orphaned = true WHERE problem_id = ?"), problemID)
	if err != nil {
		return err
	}
	_, err = s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM subtask_tests WHERE test_id IN (SELECT id FROM tests WHERE problem_id = ?);"), problemID)
	return err
}

// TODO: Test subtask stuff actually works
func (s *DB) OrphanTest(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("UPDATE tests SET orphaned = true WHERE id = ?"), id)
	if err != nil {
		return err
	}
	_, err = s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM subtask_tests WHERE test_id = ?;"), id)
	return err
}

func (s *DB) BiggestVID(ctx context.Context, problemID int) (int, error) {
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind("SELECT visible_id FROM tests WHERE problem_id = ? AND orphaned = false ORDER BY visible_id DESC LIMIT 1;"), problemID)
	return id, err
}
