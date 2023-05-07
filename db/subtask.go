package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *DB) CreateSubTask(ctx context.Context, subtask *kilonova.SubTask) error {
	if subtask.ProblemID == 0 || subtask.Tests == nil {
		return kilonova.ErrMissingRequired
	}
	var id int
	// Do insertion
	err := s.conn.GetContext(ctx, &id, "INSERT INTO subtasks (problem_id, visible_id, score) VALUES ($1, $2, $3) RETURNING id", subtask.ProblemID, subtask.VisibleID, subtask.Score)
	if err != nil {
		return err
	}
	subtask.ID = id
	// Add subtask's tests
	return s.UpdateSubTaskTests(ctx, subtask.ID, subtask.Tests)
}

func (s *DB) SubTask(ctx context.Context, pbid, stvid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.conn.GetContext(ctx, &st, "SELECT * FROM subtasks WHERE problem_id = $1 AND visible_id = $2", pbid, stvid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.internalToSubTask(ctx, &st)
}

func (s *DB) SubTaskByID(ctx context.Context, stid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.conn.GetContext(ctx, &st, "SELECT * FROM subtasks WHERE id = $1", stid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.internalToSubTask(ctx, &st)
}

func (s *DB) SubTasks(ctx context.Context, pbid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := s.conn.SelectContext(ctx, &st, "SELECT * FROM subtasks WHERE problem_id = $1 ORDER BY visible_id", pbid)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTask{}, nil
	}
	sts := []*kilonova.SubTask{}
	for _, ss := range st {
		stk, err := s.internalToSubTask(ctx, ss)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				zap.S().Warn(err)
			}
			continue
		}
		sts = append(sts, stk)
	}
	return sts, err
}

func (s *DB) SubTasksByTest(ctx context.Context, pbid, tid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := s.conn.SelectContext(ctx, &st, "SELECT stks.* FROM subtasks stks LEFT JOIN subtask_tests stt ON stks.id = stt.subtask_id WHERE stt.test_id = $1 AND stks.problem_id = $2 ORDER BY visible_id", tid, pbid)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTask{}, nil
	}
	sts := []*kilonova.SubTask{}
	for _, ss := range st {
		stk, err := s.internalToSubTask(ctx, ss)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		sts = append(sts, stk)
	}
	return sts, err
}

func (s *DB) UpdateSubTask(ctx context.Context, id int, upd kilonova.SubTaskUpdate) error {
	b := newUpdateBuilder()
	if v := upd.VisibleID; v != nil {
		b.AddUpdate("visible_id = %s", v)
	}
	if v := upd.Score; v != nil {
		b.AddUpdate("score = %s", v)
	}

	if b.CheckUpdates() != nil {
		return kilonova.ErrNoUpdates
	}
	toUpd, args := b.ToUpdate(), b.Args()
	args = append(args, id)

	_, err := s.pgconn.Exec(ctx, fmt.Sprintf("UPDATE subtasks SET %s WHERE id = $%d", toUpd, len(args)), args...)
	if err != nil {
		zap.S().Warn(err)
	}
	return err
}

func (s *DB) UpdateSubTaskTests(ctx context.Context, id int, testIDs []int) error {
	return s.updateManyToMany(ctx, "subtask_tests", "subtask_id", "test_id", id, testIDs, false)
}

func (s *DB) DeleteSubTask(ctx context.Context, stid int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM subtasks WHERE id = $1", stid)
	return err
}

func (s *DB) DeleteSubTasks(ctx context.Context, pbid int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM subtasks WHERE problem_id = $1", pbid)
	return err
}

func (s *DB) CleanupSubTasks(ctx context.Context, pbid int) error {
	_, err := s.pgconn.Exec(ctx, `
DELETE FROM subtasks 
WHERE 
	problem_id = $1 AND 
	id IN (
		WITH stk_count AS (
			SELECT subtask_id, COUNT(*) as cnt FROM subtask_tests GROUP BY subtask_id
		) 
		SELECT subtask_id FROM stk_count WHERE cnt = 0
	);
`, pbid)
	return err
}

type subtask struct {
	ID        int
	CreatedAt time.Time `db:"created_at"`
	ProblemID int       `db:"problem_id"`
	VisibleID int       `db:"visible_id"`
	Score     int
}

func (s *DB) internalToSubTask(ctx context.Context, st *subtask) (*kilonova.SubTask, error) {
	if st == nil {
		return nil, nil
	}

	var ids []int
	err := s.conn.SelectContext(ctx, &ids, `
SELECT subtask_tests.test_id 
FROM subtask_tests 
INNER JOIN tests 
	ON tests.id = subtask_tests.test_id 
WHERE 
	subtask_tests.subtask_id = $1 AND tests.orphaned = false 
ORDER BY tests.visible_id ASC
`, st.ID)
	if errors.Is(err, sql.ErrNoRows) {
		ids = []int{}
	} else if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		ids = []int{}
	}

	return &kilonova.SubTask{
		ID:        st.ID,
		CreatedAt: st.CreatedAt,
		ProblemID: st.ProblemID,
		VisibleID: st.VisibleID,
		Score:     st.Score,
		Tests:     ids,
	}, nil
}
