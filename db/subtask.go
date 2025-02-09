package db

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func (s *DB) CreateSubTask(ctx context.Context, subtask *kilonova.SubTask) error {
	if subtask.ProblemID == 0 || subtask.Tests == nil {
		return kilonova.ErrMissingRequired
	}
	var id int
	// Do insertion
	err := s.conn.QueryRow(ctx, "INSERT INTO subtasks (problem_id, visible_id, score) VALUES ($1, $2, $3) RETURNING id", subtask.ProblemID, subtask.VisibleID, subtask.Score).Scan(&id)
	if err != nil {
		return err
	}
	subtask.ID = id
	// Add subtask's tests
	return s.UpdateSubTaskTests(ctx, subtask.ID, subtask.Tests)
}

func (s *DB) SubTask(ctx context.Context, pbid, stvid int) (*kilonova.SubTask, error) {
	var st subtask
	err := Get(s.conn, ctx, &st, "SELECT * FROM subtasks WHERE problem_id = $1 AND visible_id = $2", pbid, stvid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.internalToSubTask(ctx, &st)
}

func (s *DB) SubTaskByID(ctx context.Context, stid int) (*kilonova.SubTask, error) {
	var st subtask
	err := Get(s.conn, ctx, &st, "SELECT * FROM subtasks WHERE id = $1", stid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.internalToSubTask(ctx, &st)
}

func (s *DB) SubTasks(ctx context.Context, pbid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := Select(s.conn, ctx, &st, "SELECT * FROM subtasks WHERE problem_id = $1 ORDER BY visible_id", pbid)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.SubTask{}, nil
	}
	sts := []*kilonova.SubTask{}
	for _, ss := range st {
		stk, err := s.internalToSubTask(ctx, ss)
		if err != nil {
			slog.WarnContext(ctx, "Could not get subtask", slog.Any("err", err))
			continue
		}
		sts = append(sts, stk)
	}
	return sts, err
}

func (s *DB) SubTasksByTest(ctx context.Context, pbid, tid int) ([]*kilonova.SubTask, error) {
	var st []*subtask
	err := Select(s.conn, ctx, &st, "SELECT stks.* FROM subtasks stks LEFT JOIN subtask_tests stt ON stks.id = stt.subtask_id WHERE stt.test_id = $1 AND stks.problem_id = $2 ORDER BY visible_id", tid, pbid)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.SubTask{}, nil
	}
	sts := []*kilonova.SubTask{}
	for _, ss := range st {
		stk, err := s.internalToSubTask(ctx, ss)
		if err != nil {
			slog.WarnContext(ctx, "Could not get subtask", slog.Any("err", err))
			continue
		}
		sts = append(sts, stk)
	}
	return sts, err
}

func (s *DB) UpdateSubTask(ctx context.Context, id int, upd kilonova.SubTaskUpdate) error {
	ub := newUpdateBuilder()
	if v := upd.VisibleID; v != nil {
		ub.AddUpdate("visible_id = %s", v)
	}
	if v := upd.Score; v != nil {
		ub.AddUpdate("score = %s", v)
	}

	if ub.CheckUpdates() != nil {
		return kilonova.ErrNoUpdates
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)

	_, err := s.conn.Exec(ctx, "UPDATE subtasks SET "+fb.WithUpdate(), fb.Args()...)
	if err != nil {
		slog.WarnContext(ctx, "Could not update subtask", slog.Any("err", err))
	}
	return err
}

func (s *DB) UpdateSubTaskTests(ctx context.Context, id int, testIDs []int) error {
	return s.updateManyToMany(ctx, "subtask_tests", "subtask_id", "test_id", id, testIDs, false)
}

func (s *DB) DeleteSubTask(ctx context.Context, stid int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM subtasks WHERE id = $1", stid)
	return err
}

func (s *DB) DeleteSubTasks(ctx context.Context, pbid int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM subtasks WHERE problem_id = $1", pbid)
	return err
}

func (s *DB) CleanupSubTasks(ctx context.Context, pbid int) error {
	_, err := s.conn.Exec(ctx, `
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

	Score decimal.Decimal
}

func (s *DB) internalToSubTask(ctx context.Context, st *subtask) (*kilonova.SubTask, error) {
	if st == nil {
		return nil, nil
	}

	rows, _ := s.conn.Query(ctx, `
SELECT subtask_tests.test_id 
FROM subtask_tests 
INNER JOIN tests 
	ON tests.id = subtask_tests.test_id 
WHERE 
	subtask_tests.subtask_id = $1
ORDER BY tests.visible_id ASC
`, st.ID)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if errors.Is(err, pgx.ErrNoRows) {
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
