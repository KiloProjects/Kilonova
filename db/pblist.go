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
	err := s.conn.GetContext(ctx, &pblist, s.conn.Rebind("SELECT * FROM problem_lists WHERE id = ? LIMIT 1"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToPbList(ctx, &pblist)
}

func (s *DB) ProblemLists(ctx context.Context, filter kilonova.ProblemListFilter) ([]*kilonova.ProblemList, error) {
	var lists []*pblist
	where, args := pblistFilterQuery(&filter)
	query := "SELECT * FROM problem_lists WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset)
	query = s.conn.Rebind(query)
	err := s.conn.SelectContext(ctx, &lists, query, args...)

	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ProblemList{}, nil
	}

	outLists := make([]*kilonova.ProblemList, 0, len(lists))
	for _, el := range lists {
		pblist, err := s.internalToPbList(ctx, el)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		outLists = append(outLists, pblist)
	}
	return outLists, err
}

func (s *DB) shallowProblemLists(ctx context.Context, parentID int) ([]*kilonova.ShallowProblemList, error) {
	var lists []*pblist
	query := s.conn.Rebind("SELECT pls.* FROM problem_lists pls INNER JOIN problem_list_pblists plpb ON plpb.parent_id = ? AND pls.id = plpb.child_id ORDER BY plpb.position ASC, id ASC")
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

const createProblemListQuery = "INSERT INTO problem_lists (author_id, title, description) VALUES (?, ?, ?) RETURNING id;"

func (s *DB) CreateProblemList(ctx context.Context, list *kilonova.ProblemList) error {
	if list.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	// Do insertion
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(createProblemListQuery), list.AuthorID, list.Title, list.Description)
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

func pblistFilterQuery(filter *kilonova.ProblemListFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.AuthorID; v != nil {
		where, args = append(where, "author_id = ?"), append(args, v)
	}
	if v := filter.Root; v {
		where = append(where, "id NOT IN (SELECT child_id FROM problem_list_pblists)")
	}

	return where, args
}

func pblistUpdateQuery(upd *kilonova.ProblemListUpdate) ([]string, []interface{}) {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.AuthorID; v != nil {
		toUpd, args = append(toUpd, "author_id = ?"), append(args, v)
	}
	if v := upd.Title; v != nil {
		toUpd, args = append(toUpd, "title = ?"), append(args, v)
	}
	if v := upd.Description; v != nil {
		toUpd, args = append(toUpd, "description = ?"), append(args, v)
	}
	return toUpd, args
}

type pblist struct {
	ID          int
	CreatedAt   time.Time `db:"created_at"`
	AuthorID    int       `db:"author_id"`
	Title       string
	Description string
}

func (s *DB) internalToPbList(ctx context.Context, list *pblist) (*kilonova.ProblemList, error) {
	pblist := &kilonova.ProblemList{
		ID:          list.ID,
		CreatedAt:   list.CreatedAt,
		Title:       list.Title,
		Description: list.Description,
		AuthorID:    list.AuthorID,
	}

	err := s.conn.SelectContext(ctx, &pblist.List, s.conn.Rebind("SELECT problem_id FROM problem_list_problems WHERE pblist_id = ? ORDER BY position ASC, problem_id ASC"), list.ID)
	if errors.Is(err, sql.ErrNoRows) || len(pblist.List) == 0 {
		pblist.List = []int{}
	} else if err != nil {
		return nil, err
	}

	pblist.SubLists, err = s.shallowProblemLists(ctx, list.ID)
	if err != nil {
		return nil, err
	}

	return pblist, nil
}

func (s *DB) internalToShallowProblemList(ctx context.Context, list *pblist) (*kilonova.ShallowProblemList, error) {

	var problems []int
	err := s.conn.SelectContext(ctx, &problems, s.conn.Rebind("SELECT problem_id FROM problem_list_problems WHERE pblist_id = ? ORDER BY position ASC, problem_id ASC"), list.ID)
	if errors.Is(err, sql.ErrNoRows) || len(problems) == 0 {
		problems = []int{}
	} else if err != nil {
		return nil, err
	}

	return &kilonova.ShallowProblemList{
		ID:       list.ID,
		Title:    list.Title,
		AuthorID: list.AuthorID,

		List: problems,
	}, nil
}
