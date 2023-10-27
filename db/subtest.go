package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

func (s *DB) SubTestsBySubID(ctx context.Context, subid int) ([]*kilonova.SubTest, error) {
	var subtests []*kilonova.SubTest
	err := Select(s.conn, ctx, &subtests, "SELECT * FROM submission_tests WHERE submission_id = $1 ORDER BY visible_id ASC", subid)
	if errors.Is(err, pgx.ErrNoRows) {
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
	err := Select(s.conn, ctx, &subtests, fmt.Sprintf(`
WITH subtask_ids AS (
	%s
) SELECT DISTINCT ON (sts.id) sts.* 
	FROM submission_tests sts, submission_subtask_subtests ssst, subtask_ids 
	WHERE sts.id = ssst.submission_test_id 
	  AND ssst.submission_subtask_id = subtask_ids.id 
	ORDER BY id ASC
`, getSubmissionSubtaskQuery(contestID != nil)), args...)
	if errors.Is(err, pgx.ErrNoRows) {
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
	err := Get(s.conn, ctx, &subtest, "SELECT * FROM submission_tests WHERE id = $1", id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &subtest, err
}

func (s *DB) UpdateSubTest(ctx context.Context, id int, upd kilonova.SubTestUpdate) error {
	ub := newUpdateBuilder()
	subtestUpdateQuery(&upd, ub)
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)
	_, err := s.conn.Exec(ctx, "UPDATE submission_tests SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func subtestUpdateQuery(upd *kilonova.SubTestUpdate, ub *updateBuilder) {
	if v := upd.Memory; v != nil {
		ub.AddUpdate("memory = %s", v)
	}
	if v := upd.Percentage; v != nil {
		ub.AddUpdate("percentage = %s", v)
	}
	if v := upd.Time; v != nil {
		ub.AddUpdate("time = %s", v)
	}
	if v := upd.Verdict; v != nil {
		ub.AddUpdate("verdict = %s", v)
	}
	if v := upd.Done; v != nil {
		ub.AddUpdate("done = %s", v)
	}
	if v := upd.Skipped; v != nil {
		ub.AddUpdate("skipped = %s", v)
	}
}
