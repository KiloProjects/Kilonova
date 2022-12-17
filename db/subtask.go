package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind("INSERT INTO subtasks (problem_id, visible_id, score) VALUES (?, ?, ?) RETURNING id"), subtask.ProblemID, subtask.VisibleID, subtask.Score)
	if err != nil {
		return err
	}
	subtask.ID = id
	// Add subtask's tests
	return s.UpdateSubTaskTests(ctx, subtask.ID, subtask.Tests)
}

func (s *DB) SubTask(ctx context.Context, pbid, stvid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.conn.GetContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE problem_id = ? AND visible_id = ?"), pbid, stvid)
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
	err := s.conn.GetContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE id = ?"), stid)
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
	err := s.conn.SelectContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE problem_id = ? ORDER BY visible_id"), pbid)
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
	err := s.conn.SelectContext(ctx, &st, s.conn.Rebind("SELECT stks.* FROM subtasks stks LEFT JOIN subtask_tests stt ON stks.id = stt.subtask_id WHERE stt.test_id = ? AND stks.problem_id = ? ORDER BY visible_id"), tid, pbid)
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
	toUpd, args := []string{}, []interface{}{}
	if v := upd.VisibleID; v != nil {
		toUpd, args = append(toUpd, "visible_id = ?"), append(args, v)
	}
	if v := upd.Score; v != nil {
		toUpd, args = append(toUpd, "score = ?"), append(args, v)
	}

	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)

	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(fmt.Sprintf("UPDATE subtasks SET %s WHERE id = ?", strings.Join(toUpd, ", "))), args...)
	if err != nil {
		zap.S().Warn(err)
	}
	return err
}

func (s *DB) UpdateSubTaskTests(ctx context.Context, id int, testIDs []int) error {
	return s.updateManyToMany(ctx, "subtask_tests", "subtask_id", "test_id", id, testIDs, false)
}

func (s *DB) DeleteSubTask(ctx context.Context, stid int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM subtasks WHERE id = ?"), stid)
	return err
}

func (s *DB) DeleteSubTasks(ctx context.Context, pbid int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM subtasks WHERE problem_id = ?"), pbid)
	return err
}

func (s *DB) CleanupSubTasks(ctx context.Context, pbid int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`
DELETE FROM subtasks 
WHERE 
	problem_id = ? AND 
	id IN (
		WITH stk_count AS (
			SELECT subtask_id, COUNT(*) as cnt FROM subtask_tests GROUP BY subtask_id
		) 
		SELECT subtask_id FROM stk_count WHERE cnt = 0
	);
`), pbid)
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
	err := s.conn.SelectContext(ctx, &ids, s.conn.Rebind("SELECT subtask_tests.test_id FROM subtask_tests INNER JOIN tests ON tests.id = subtask_tests.test_id WHERE subtask_tests.subtask_id = ? AND tests.orphaned = false ORDER BY tests.visible_id ASC"), st.ID)
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
