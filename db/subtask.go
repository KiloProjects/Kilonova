package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) CreateSubTask(ctx context.Context, subtask *kilonova.SubTask) error {
	if subtask.ProblemID == 0 || subtask.Tests == nil {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind("INSERT INTO subtasks (problem_id, visible_id, score, tests) VALUES (?, ?, ?, ?) RETURNING id"), subtask.ProblemID, subtask.VisibleID, subtask.Score, kilonova.SerializeIntList(subtask.Tests))
	if err == nil {
		subtask.ID = id
	}
	return err
}

func (s *DB) SubTask(ctx context.Context, pbid, stvid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.conn.GetContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE problem_id = ? AND visible_id = ?"), pbid, stvid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return InternalToSubTask(&st), err
}

func (s *DB) SubTaskByID(ctx context.Context, stid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.conn.GetContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE id = ?"), stid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return InternalToSubTask(&st), err
}

func (s *DB) SubTasks(ctx context.Context, pbid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := s.conn.SelectContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE problem_id = ? ORDER BY visible_id"), pbid)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTask{}, nil
	}
	sts := []*kilonova.SubTask{}
	for _, s := range st {
		sts = append(sts, InternalToSubTask(s))
	}
	return sts, err
}

func (s *DB) SubTasksByTest(ctx context.Context, pbid, tid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := s.conn.SelectContext(ctx, &st, s.conn.Rebind("SELECT * FROM subtasks WHERE tests LIKE ? AND problem_id = ? ORDER BY visible_id"), fmt.Sprintf("%%%d%%", tid), pbid)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTask{}, nil
	}
	sts := []*kilonova.SubTask{}
	for _, s := range st {
		sts = append(sts, InternalToSubTask(s))
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
	if v := upd.Tests; v != nil {
		toUpd, args = append(toUpd, "tests = ?"), append(args, kilonova.SerializeIntList(v))
	}

	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)

	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(fmt.Sprintf("UPDATE subtasks SET %s WHERE id = ?", strings.Join(toUpd, ", "))), args...)
	return err
}

func (s *DB) DeleteSubTask(ctx context.Context, stid int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM subtasks WHERE id = ?"), stid)
	return err
}

func (s *DB) DeleteSubTasks(ctx context.Context, pbid int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM subtasks WHERE problem_id = ?"), pbid)
	return err
}

type subtask struct {
	ID        int
	CreatedAt time.Time `db:"created_at"`
	ProblemID int       `db:"problem_id"`
	VisibleID int       `db:"visible_id"`
	Score     int
	Tests     string
}

func InternalToSubTask(st *subtask) *kilonova.SubTask {
	if st == nil {
		return nil
	}
	return &kilonova.SubTask{
		ID:        st.ID,
		CreatedAt: st.CreatedAt,
		ProblemID: st.ProblemID,
		VisibleID: st.VisibleID,
		Score:     st.Score,
		Tests:     kilonova.DeserializeIntList(st.Tests),
	}
}
