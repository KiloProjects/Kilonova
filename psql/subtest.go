package psql

import (
	"context"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.SubTestService = &SubTestService{}

type SubTestService struct {
	db *sqlx.DB
}

func (s *SubTestService) SubTestsBySubID(ctx context.Context, subid int) ([]*kilonova.SubTest, error) {
	var subtests []*kilonova.SubTest
	err := s.db.SelectContext(ctx, &subtests, "SELECT * FROM submission_tests WHERE submission_id = $1 ORDER BY id ASC", subid)
	return subtests, err
}

func (s *SubTestService) SubTest(ctx context.Context, id int) (*kilonova.SubTest, error) {
	var subtest kilonova.SubTest
	err := s.db.GetContext(ctx, &subtest, "SELECT * FROM submission_tests WHERE id = $1", id)
	return &subtest, err
}

func (s *SubTestService) CreateSubTest(ctx context.Context, subtest *kilonova.SubTest) error {
	if subtest.UserID == 0 || subtest.TestID == 0 || subtest.SubmissionID == 0 {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := s.db.GetContext(ctx, &id, `INSERT INTO submission_tests (user_id, test_id, submission_id) VALUES ($1, $2, $3) RETURNING id;`, subtest.UserID, subtest.TestID, subtest.SubmissionID)
	if err == nil {
		subtest.ID = id
	}
	return err
}

func (s *SubTestService) UpdateSubTest(ctx context.Context, id int, upd kilonova.SubTestUpdate) error {
	toUpd, args := s.updateQueryMaker(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.db.Rebind(fmt.Sprintf("UPDATE submission_tests SET %s WHERE id = ?", strings.Join(toUpd, ", ")))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SubTestService) updateQueryMaker(upd *kilonova.SubTestUpdate) ([]string, []interface{}) {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.Memory; v != nil {
		toUpd, args = append(toUpd, "memory = ?"), append(args, v)
	}
	if v := upd.Score; v != nil {
		toUpd, args = append(toUpd, "score = ?"), append(args, v)
	}
	if v := upd.Time; v != nil {
		toUpd, args = append(toUpd, "time = ?"), append(args, v)
	}
	if v := upd.Verdict; v != nil {
		toUpd, args = append(toUpd, "verdict = ?"), append(args, v)
	}
	if v := upd.Done; v != nil {
		toUpd, args = append(toUpd, "done = ?"), append(args, v)
	}
	if v := upd.Skipped; v != nil {
		toUpd, args = append(toUpd, "skipped = ?"), append(args, v)
	}

	return toUpd, args
}
