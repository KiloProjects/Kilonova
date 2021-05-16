package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.SubTaskService = &SubTaskService{}

type SubTaskService struct {
	db *sqlx.DB
}

func (s *SubTaskService) CreateSubTask(ctx context.Context, subtask *kilonova.SubTask) error {
	if subtask.ProblemID == 0 || subtask.Tests == nil {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := s.db.GetContext(ctx, &id, s.db.Rebind("INSERT INTO subtasks (problem_id, visible_id, score, tests) VALUES (?, ?, ?, ?) RETURNING id"), subtask.ProblemID, subtask.VisibleID, subtask.Score, kilonova.SerializeIntList(subtask.Tests))
	if err == nil {
		subtask.ID = id
	}
	return err
}

func (s *SubTaskService) SubTask(ctx context.Context, pbid, stvid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.db.GetContext(ctx, &st, s.db.Rebind("SELECT * FROM subtasks WHERE problem_id = ? AND visible_id = ?"), pbid, stvid)
	if err != nil {
		return nil, err
	}
	return InternalToSubTask(&st), nil
}

func (s *SubTaskService) SubTaskByID(ctx context.Context, stid int) (*kilonova.SubTask, error) {
	var st subtask
	err := s.db.GetContext(ctx, &st, s.db.Rebind("SELECT * FROM subtasks WHERE id = ?"), stid)
	if err != nil {
		return nil, err
	}
	return InternalToSubTask(&st), nil
}

func (s *SubTaskService) SubTasks(ctx context.Context, pbid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := s.db.SelectContext(ctx, &st, s.db.Rebind("SELECT * FROM subtasks WHERE problem_id = ? ORDER BY visible_id"), pbid)
	if err != nil {
		return nil, err
	}
	sts := []*kilonova.SubTask{}
	for _, s := range st {
		sts = append(sts, InternalToSubTask(s))
	}
	return sts, nil
}

func (s *SubTaskService) SubTasksByTest(ctx context.Context, pbid, tid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := s.db.SelectContext(ctx, &st, s.db.Rebind("SELECT * FROM subtasks WHERE tests LIKE ? AND problem_id = ? ORDER BY visible_id"), fmt.Sprintf("%%%d%%", tid), pbid)
	if err != nil {
		return nil, err
	}
	sts := []*kilonova.SubTask{}
	for _, s := range st {
		sts = append(sts, InternalToSubTask(s))
	}
	return sts, nil
}

func (s *SubTaskService) UpdateSubTask(ctx context.Context, id int, upd kilonova.SubTaskUpdate) error {
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

	_, err := s.db.ExecContext(ctx, s.db.Rebind(fmt.Sprintf("UPDATE subtasks SET %s WHERE id = ?", strings.Join(toUpd, ", "))), args...)
	return err
}

func (s *SubTaskService) DeleteSubTask(ctx context.Context, stid int) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind("DELETE FROM subtasks WHERE id = ?"), stid)
	return err
}

func (s *SubTaskService) DeleteSubTasks(ctx context.Context, pbid int) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind("DELETE FROM subtasks WHERE problem_id = ?"), pbid)
	return err
}

func NewSubTaskService(db *sqlx.DB) kilonova.SubTaskService {
	return &SubTaskService{db}
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
	return &kilonova.SubTask{
		ID:        st.ID,
		CreatedAt: st.CreatedAt,
		ProblemID: st.ProblemID,
		VisibleID: st.VisibleID,
		Score:     st.Score,
		Tests:     kilonova.DeserializeIntList(st.Tests),
	}
}
