package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
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
	Score        int       `db:"score"`

	FinalPercentage *int `db:"final_percentage"`
}

func (s *DB) InitSubmissionSubtasks(ctx context.Context, userID int, submissionID int, problemID int, contestID *int) (err error) {
	_, err = s.conn.ExecContext(ctx, `
INSERT INTO submission_subtasks 
	(user_id, submission_id, contest_id, subtask_id, problem_id, visible_id, score) 
	SELECT $1, $2, $3, id, problem_id, visible_id, score FROM subtasks WHERE problem_id = $4;
`, userID, submissionID, contestID, problemID)
	if err != nil {
		zap.S().Warn("Couldn't insert submission subtask: ", err)
		return
	}

	_, err = s.conn.ExecContext(ctx, `
INSERT INTO submission_subtask_subtests 
	SELECT sstk.id, st.id
	FROM subtask_tests stks
	INNER JOIN submission_subtasks sstk ON stks.subtask_id = sstk.subtask_id
	INNER JOIN submission_tests st ON stks.test_id = st.test_id AND sstk.submission_id = st.submission_id WHERE st.submission_id = $1;
`, submissionID)
	if err != nil {
		zap.S().Warn("Couldn't insert submission subtask's subtests: ", err)
		return
	}

	return
}

func (s *DB) UpdateSubmissionSubtaskPercentage(ctx context.Context, id int, percentage int) (err error) {
	_, err = s.conn.ExecContext(ctx, `UPDATE submission_subtasks SET final_percentage = $1 WHERE id = $2`, percentage, id)
	return
}

func (s *DB) SubmissionSubTasksBySubID(ctx context.Context, subid int) ([]*kilonova.SubmissionSubTask, error) {
	var subtasks []*subSubtask
	err := s.conn.SelectContext(ctx, &subtasks, "SELECT * FROM submission_subtasks WHERE submission_id = $1 ORDER BY visible_id ASC", subid)
	if errors.Is(err, sql.ErrNoRows) {
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
	err := s.conn.SelectContext(ctx, &subtasks, getSubmissionSubtaskQuery(contestID != nil), args...)
	if errors.Is(err, sql.ErrNoRows) {
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

	var ids []int
	err := s.conn.SelectContext(ctx, &ids, s.conn.Rebind(`
SELECT submission_subtask_subtests.submission_test_id
FROM submission_subtask_subtests
INNER JOIN submission_tests subtests
	ON subtests.id = submission_subtask_subtests.submission_test_id 
WHERE 
	submission_subtask_subtests.submission_subtask_id = ? 
ORDER BY subtests.visible_id ASC
`), st.ID)
	if errors.Is(err, sql.ErrNoRows) {
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
	}, nil
}
