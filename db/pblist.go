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

func (s *DB) ProblemList(ctx context.Context, id int) (*kilonova.ProblemList, error) {
	var pblist pblist
	err := s.conn.GetContext(ctx, &pblist, "SELECT * FROM problem_lists WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, context.Canceled) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToPbList(ctx, &pblist)
}

func (s *DB) ProblemLists(ctx context.Context, root bool) ([]*kilonova.ProblemList, error) {
	var lists []*pblist

	q := "SELECT * FROM problem_lists ORDER BY id ASC"
	if root {
		q = "SELECT * FROM problem_lists WHERE NOT EXISTS (SELECT 1 FROM problem_list_pblists WHERE child_id = problem_lists.id) ORDER BY id ASC"
	}
	err := s.conn.SelectContext(ctx, &lists, q)

	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) ProblemListsByProblemID(ctx context.Context, problemID int, showHidable bool) ([]*kilonova.ProblemList, error) {
	var lists []*pblist

	q := "SELECT * FROM problem_lists WHERE EXISTS (SELECT 1 FROM problem_list_problems WHERE pblist_id = problem_lists.id AND problem_id = $1) ORDER BY id ASC"
	if !showHidable {
		q = "SELECT * FROM problem_lists WHERE EXISTS (SELECT 1 FROM problem_list_problems WHERE pblist_id = problem_lists.id AND problem_id = $1) AND sidebar_hidable = false ORDER BY id ASC"
	}
	err := s.conn.SelectContext(ctx, &lists, q, problemID)

	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) ProblemListsByPblistID(ctx context.Context, pblistID int) ([]*kilonova.ProblemList, error) {
	var lists []*pblist

	q := "SELECT * FROM problem_lists WHERE EXISTS (SELECT 1 FROM problem_list_pblists WHERE parent_id = problem_lists.id AND child_id = $1) ORDER BY id ASC"
	err := s.conn.SelectContext(ctx, &lists, q, pblistID)

	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) shallowProblemLists(ctx context.Context, parentID int) ([]*kilonova.ShallowProblemList, error) {
	var lists []*pblist
	query := "SELECT pls.* FROM problem_lists pls INNER JOIN problem_list_pblists plpb ON plpb.parent_id = $1 AND pls.id = plpb.child_id ORDER BY plpb.position ASC, id ASC"
	err := s.conn.SelectContext(ctx, &lists, query, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ShallowProblemList{}, nil
	}

	outLists := make([]*kilonova.ShallowProblemList, 0, len(lists))
	for _, el := range lists {
		pblist, err := s.internalToShallowProblemList(ctx, el)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		outLists = append(outLists, pblist)
	}
	return outLists, err
}

func (s *DB) numPblistProblems(ctx context.Context, listID int) (int, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, `
WITH RECURSIVE nested_lists AS (
	SELECT id FROM problem_lists WHERE id = $1
	UNION
	SELECT plp.child_id FROM problem_list_pblists plp, nested_lists WHERE nested_lists.id = plp.parent_id
), problem_ids AS (
	SELECT DISTINCT pbs.id FROM problems pbs, problem_list_problems plp, nested_lists lists 
		WHERE pbs.id = plp.problem_id AND lists.id = plp.pblist_id
) SELECT COUNT(*) FROM problem_ids;
`, listID)
	if err != nil {
		zap.S().Warn(err)
		return -1, err
	}
	return cnt, nil
}

func (s *DB) NumSolvedPblistProblems(ctx context.Context, listID, userID int) (int, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, `
	WITH RECURSIVE nested_lists AS (
		SELECT id FROM problem_lists WHERE id = $1
		UNION
		SELECT plp.child_id FROM problem_list_pblists plp, nested_lists 
			WHERE nested_lists.id = plp.parent_id
	), problem_ids AS (
		SELECT DISTINCT pbs.id FROM problems pbs, problem_list_problems plp, nested_lists lists 
			WHERE pbs.id = plp.problem_id AND lists.id = plp.pblist_id
	), solved_pbs AS (
		SELECT DISTINCT subs.problem_id FROM submissions subs, problem_ids pbids 
			WHERE subs.problem_id = pbids.id AND subs.score = 100 AND subs.user_id = $2
	) SELECT COUNT(*) FROM solved_pbs;
	`, listID, userID)
	if err != nil {
		zap.S().Warn(err)
		return -1, err
	}
	return cnt, nil
}

const createProblemListQuery = "INSERT INTO problem_lists (author_id, title, description, sidebar_hidable) VALUES (?, ?, ?, ?) RETURNING id;"

func (s *DB) CreateProblemList(ctx context.Context, list *kilonova.ProblemList) error {
	if list.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	// Do insertion
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(createProblemListQuery), list.AuthorID, list.Title, list.Description, list.SidebarHidable)
	if err != nil {
		return err
	}
	list.ID = id

	// Add problems
	return s.UpdateProblemListProblems(ctx, list.ID, list.List)
}

func (s *DB) UpdateProblemList(ctx context.Context, id int, upd kilonova.ProblemListUpdate) error {
	toUpd, args := pblistUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.conn.Rebind(fmt.Sprintf(`UPDATE problem_lists SET %s WHERE id = ?;`, strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) DeleteProblemList(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM problem_lists WHERE id = ?"), id)
	return err
}

func (s *DB) UpdateProblemListProblems(ctx context.Context, id int, problemIDs []int) error {
	return s.updateManyToMany(ctx, "problem_list_problems", "pblist_id", "problem_id", id, problemIDs, true)
}

func (s *DB) UpdateProblemListSublists(ctx context.Context, id int, listIDs []int) error {
	// Quick sanity check first
	for _, listID := range listIDs {
		if id == listID {
			return kilonova.Statusf(400, "List %d cannot nest itself!", id)
		}
	}

	return s.updateManyToMany(ctx, "problem_list_pblists", "parent_id", "child_id", id, listIDs, true)
}

func pblistUpdateQuery(upd *kilonova.ProblemListUpdate) ([]string, []any) {
	toUpd, args := []string{}, []any{}
	if v := upd.AuthorID; v != nil {
		toUpd, args = append(toUpd, "author_id = ?"), append(args, v)
	}
	if v := upd.Title; v != nil {
		toUpd, args = append(toUpd, "title = ?"), append(args, v)
	}
	if v := upd.Description; v != nil {
		toUpd, args = append(toUpd, "description = ?"), append(args, v)
	}
	if v := upd.SidebarHidable; v != nil {
		toUpd, args = append(toUpd, "sidebar_hidable = ?"), append(args, v)
	}
	return toUpd, args
}

type pblist struct {
	ID          int
	CreatedAt   time.Time `db:"created_at"`
	AuthorID    int       `db:"author_id"`
	Title       string
	Description string

	SidebarHidable bool `db:"sidebar_hidable"`
}

func (s *DB) internalToPbList(ctx context.Context, list *pblist) (*kilonova.ProblemList, error) {
	pblist := &kilonova.ProblemList{
		ID:          list.ID,
		CreatedAt:   list.CreatedAt,
		Title:       list.Title,
		Description: list.Description,
		AuthorID:    list.AuthorID,

		SidebarHidable: list.SidebarHidable,
	}

	err := s.conn.SelectContext(ctx, &pblist.List, "SELECT problem_id FROM problem_list_problems WHERE pblist_id = $1 ORDER BY position ASC, problem_id ASC", list.ID)
	if errors.Is(err, sql.ErrNoRows) || len(pblist.List) == 0 {
		pblist.List = []int{}
	} else if err != nil {
		return nil, err
	}

	pblist.SubLists, err = s.shallowProblemLists(ctx, list.ID)
	if err != nil {
		return nil, err
	}

	pblist.NumProblems, err = s.numPblistProblems(ctx, list.ID)
	if err != nil {
		return nil, err
	}

	return pblist, nil
}

func (s *DB) internalToShallowProblemList(ctx context.Context, list *pblist) (*kilonova.ShallowProblemList, error) {

	numProblems, err := s.numPblistProblems(ctx, list.ID)
	if err != nil {
		return nil, err
	}

	return &kilonova.ShallowProblemList{
		ID:       list.ID,
		Title:    list.Title,
		AuthorID: list.AuthorID,

		SidebarHidable: list.SidebarHidable,

		NumProblems: numProblems,
	}, nil
}
