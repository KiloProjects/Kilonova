package psql

import (
	"context"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.TestService = &TestService{}

type TestService struct {
	db *sqlx.DB
}

func (s *TestService) CreateTest(ctx context.Context, test *kilonova.Test) error {
	return s.createTest(ctx, test)
}

func (s *TestService) Test(ctx context.Context, pbID, testVID int) (*kilonova.Test, error) {
	var test kilonova.Test
	err := s.db.GetContext(ctx, &test, "SELECT * FROM tests WHERE problem_id = $1 AND visible_id = $2 AND orphaned = false ORDER BY visible_id LIMIT 1", pbID, testVID)
	return &test, err
}

func (s *TestService) TestByID(ctx context.Context, id int) (*kilonova.Test, error) {
	var test kilonova.Test
	err := s.db.GetContext(ctx, &test, "SELECT * FROM tests WHERE id = $1 LIMIT 1", id)
	return &test, err
}

func (s *TestService) Tests(ctx context.Context, pbID int) ([]*kilonova.Test, error) {
	var tests []*kilonova.Test
	err := s.db.SelectContext(ctx, &tests, "SELECT * FROM tests WHERE problem_id = $1 AND orphaned = false ORDER BY visible_id", pbID)
	return tests, err
}

func (s *TestService) UpdateTest(ctx context.Context, id int, upd kilonova.TestUpdate) error {
	return s.updateTest(ctx, id, upd)
}

func (s *TestService) OrphanProblemTests(ctx context.Context, problemID int) error {
	_, err := s.db.ExecContext(ctx, "UPDATE tests SET orphaned = true WHERE problem_id = $1", problemID)
	return err
}

func (s *TestService) BiggestVID(ctx context.Context, problemID int) (int, error) {
	var id int
	err := s.db.GetContext(ctx, &id, "SELECT visible_id FROM tests WHERE problem_id = $1 AND orphaned = false ORDER BY visible_id DESC LIMIT 1;", problemID)
	return id, err
}

func (s *TestService) updateTest(ctx context.Context, id int, upd kilonova.TestUpdate) error {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.Score; v != nil {
		toUpd, args = append(toUpd, "score = ?"), append(args, v)
	}
	if v := upd.VisibleID; v != nil {
		toUpd, args = append(toUpd, "visible_id = ?"), append(args, v)
	}
	if v := upd.Orphaned; v != nil {
		toUpd, args = append(toUpd, "orphaned = ?"), append(args, v)
	}
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)

	query := s.db.Rebind("UPDATE tests SET " + strings.Join(toUpd, ", ") + " WHERE id = ?")
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *TestService) createTest(ctx context.Context, test *kilonova.Test) error {
	if test.ProblemID == 0 || test.Score == 0 {
		return kilonova.ErrMissingRequired
	}

	var id int
	err := s.db.GetContext(ctx, &id, "INSERT INTO tests (score, problem_id, visible_id, orphaned) VALUES ($1, $2, $3, $4) RETURNING id", test.Score, test.ProblemID, test.VisibleID, test.Orphaned)
	if err == nil {
		test.ID = id
	}
	return err
}
