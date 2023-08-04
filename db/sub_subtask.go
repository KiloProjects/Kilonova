package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type subSubtask struct {
	ID           int       `db:"id"`
	CreatedAt    time.Time `db:"created_at"`
	SubmissionID int       `db:"submission_id"`
	UserID       int       `db:"user_id"`
	SubtaskID    *int      `db:"subtask_id"`
	ProblemID    int       `db:"problem_id"`
	ContestID    *int      `db:"contest_id"`
	VisibleID    int       `db:"visible_id"`

	ScorePrecision int `db:"digit_precision"`

	Score           decimal.Decimal  `db:"score"`
	FinalPercentage *decimal.Decimal `db:"final_percentage"`
}

func (s *DB) UpdateSubmissionSubtaskPercentage(ctx context.Context, id int, percentage decimal.Decimal) (err error) {
	_, err = s.conn.Exec(ctx, `UPDATE submission_subtasks SET final_percentage = $1 WHERE id = $2`, percentage, id)
	return
}

func (s *DB) SubmissionSubTasksBySubID(ctx context.Context, subid int) ([]*kilonova.SubmissionSubTask, error) {
	var subtasks []*subSubtask
	err := Select(s.conn, ctx, &subtasks, "SELECT * FROM submission_subtasks WHERE submission_id = $1 ORDER BY visible_id ASC", subid)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.SubmissionSubTask{}, nil
	} else if err != nil {
		return []*kilonova.SubmissionSubTask{}, err
	}

	return mapperCtx(ctx, subtasks, s.internalToSubmissionSubTask), nil
}

func getSubmissionSubtaskQuery(inContest bool) string {
	if inContest {
		return `
WITH org AS (
	SELECT DISTINCT ON (subtask_id) * FROM submission_subtasks WHERE subtask_id IS NOT NULL AND problem_id = $1 AND user_id = $2 AND contest_id = $3 ORDER BY subtask_id ASC, final_percentage DESC NULLS LAST, submission_id ASC
) SELECT * FROM org ORDER BY visible_id ASC
`
	} else {
		return `
WITH org AS (
	SELECT DISTINCT ON (subtask_id) * FROM submission_subtasks WHERE subtask_id IS NOT NULL AND problem_id = $1 AND user_id = $2 ORDER BY subtask_id ASC, final_percentage DESC NULLS LAST, submission_id ASC
) SELECT * FROM org ORDER BY visible_id ASC
`
	}
}

// Note that they will probably not be from the same submission!
func (s *DB) MaximumScoreSubTasks(ctx context.Context, problemID int, userID int, contestID *int) ([]*kilonova.SubmissionSubTask, error) {
	var subtasks []*subSubtask
	args := []any{problemID, userID}
	if contestID != nil {
		args = append(args, contestID)
	}
	err := Select(s.conn, ctx, &subtasks, getSubmissionSubtaskQuery(contestID != nil), args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.SubmissionSubTask{}, nil
	} else if err != nil {
		return []*kilonova.SubmissionSubTask{}, err
	}

	return mapperCtx(ctx, subtasks, s.internalToSubmissionSubTask), nil
}

func (s *DB) internalToSubmissionSubTask(ctx context.Context, st *subSubtask) (*kilonova.SubmissionSubTask, error) {
	if st == nil {
		return nil, nil
	}

	rows, _ := s.conn.Query(ctx, `SELECT submission_subtask_subtests.submission_test_id
	FROM submission_subtask_subtests
	INNER JOIN submission_tests subtests
		ON subtests.id = submission_subtask_subtests.submission_test_id 
	WHERE 
		submission_subtask_subtests.submission_subtask_id = $1 
	ORDER BY subtests.visible_id ASC`, st.ID)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if errors.Is(err, pgx.ErrNoRows) {
		ids = []int{}
	} else if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		ids = []int{}
	}

	return &kilonova.SubmissionSubTask{
		ID:           st.ID,
		CreatedAt:    st.CreatedAt,
		SubmissionID: st.SubmissionID,
		UserID:       st.UserID,
		SubtaskID:    st.SubtaskID,
		ProblemID:    st.ProblemID,
		ContestID:    st.ContestID,
		VisibleID:    st.VisibleID,
		Score:        st.Score,
		Subtests:     ids,

		FinalPercentage: st.FinalPercentage,
		ScorePrecision:  st.ScorePrecision,
	}, nil
}
