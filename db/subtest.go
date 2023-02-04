package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) SubTestsBySubID(ctx context.Context, subid int) ([]*kilonova.SubTest, error) {
	var subtests []*kilonova.SubTest
	err := s.conn.SelectContext(ctx, &subtests, "SELECT * FROM submission_tests WHERE submission_id = $1 ORDER BY visible_id ASC", subid)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTest{}, nil
	}
	return subtests, err
}

func (s *DB) SubTest(ctx context.Context, id int) (*kilonova.SubTest, error) {
	var subtest kilonova.SubTest
	err := s.conn.GetContext(ctx, &subtest, "SELECT * FROM submission_tests WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &subtest, err
}

func (s *DB) InitSubTests(ctx context.Context, userID int, submissionID int, problemID int) error {
	if userID == 0 || problemID == 0 || submissionID == 0 {
		return kilonova.ErrMissingRequired
	}
	_, err := s.conn.ExecContext(ctx, `
INSERT INTO submission_tests (user_id, submission_id, test_id, visible_id, max_score) SELECT $1, $2, id, visible_id, score FROM tests WHERE problem_id = $3 AND orphaned = false
`, userID, submissionID, problemID)
	return err
}

func (s *DB) UpdateSubTest(ctx context.Context, id int, upd kilonova.SubTestUpdate) error {
	toUpd, args := subtestUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.conn.Rebind(fmt.Sprintf("UPDATE submission_tests SET %s WHERE id = ?", strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) UpdateSubmissionSubTests(ctx context.Context, subID int, upd kilonova.SubTestUpdate) error {
	toUpd, args := subtestUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, subID)
	query := s.conn.Rebind(fmt.Sprintf("UPDATE submission_tests SET %s WHERE submission_id = ?", strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func subtestUpdateQuery(upd *kilonova.SubTestUpdate) ([]string, []any) {
	toUpd, args := []string{}, []any{}
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

	return toUpd, args
}
