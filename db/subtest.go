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
	} else if err != nil {
		return []*kilonova.SubTest{}, err
	} else if len(subtests) == 0 {
		return []*kilonova.SubTest{}, nil
	}
	return subtests, err
}

func (s *DB) MaximumScoreSubTaskTests(ctx context.Context, problemID int, userID int, contestID *int) ([]*kilonova.SubTest, error) {
	var subtests []*kilonova.SubTest
	args := []any{problemID, userID}
	if contestID != nil {
		args = append(args, contestID)
	}
	err := s.conn.SelectContext(ctx, &subtests, fmt.Sprintf(`
WITH subtask_ids AS (
	%s
) SELECT sts.* 
	FROM submission_tests sts, submission_subtask_subtests ssst, subtask_ids 
	WHERE sts.id = ssst.submission_test_id 
	  AND ssst.submission_subtask_id = subtask_ids.id 
	ORDER BY id ASC
`, getSubmissionSubtaskQuery(contestID != nil)), args...)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTest{}, nil
	} else if err != nil {
		return []*kilonova.SubTest{}, err
	} else if len(subtests) == 0 {
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

func (s *DB) InitSubTests(ctx context.Context, userID int, submissionID int, problemID int, contestID *int) error {
	if userID == 0 || problemID == 0 || submissionID == 0 {
		return kilonova.ErrMissingRequired
	}
	_, err := s.pgconn.Exec(ctx, `
INSERT INTO submission_tests (user_id, submission_id, contest_id, test_id, visible_id, max_score) SELECT $1, $2, $3, id, visible_id, score FROM tests WHERE problem_id = $4 AND orphaned = false
`, userID, submissionID, contestID, problemID)
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
