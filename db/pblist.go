package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgtype"
	"go.uber.org/zap"
)

func (s *DB) ProblemList(ctx context.Context, id int) (*kilonova.ProblemList, error) {
	var pblist pblist
	err := s.conn.GetContext(ctx, &pblist, s.conn.Rebind("SELECT * FROM problem_lists WHERE id = ? LIMIT 1"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return internalToPbList(&pblist), err
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
		outLists = append(outLists, internalToPbList(el))
	}
	return outLists, err
}

const createProblemListQuery = "INSERT INTO problem_lists (author_id, title, description, list) VALUES (?, ?, ?, ?) RETURNING id;"

func (s *DB) CreateProblemList(ctx context.Context, list *kilonova.ProblemList) error {
	if list.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	// Convert subtask list to array
	arr := pgtype.Int4Array{}
	if err := arr.Set(list.List); err != nil {
		zap.S().Warn("Wtf", err)
		return err
	}
	// Do insertion
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(createProblemListQuery), list.AuthorID, list.Title, list.Description, arr)
	if err == nil {
		list.ID = id
	}
	return err
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

func pblistFilterQuery(filter *kilonova.ProblemListFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.AuthorID; v != nil {
		where, args = append(where, "author_id = ?"), append(args, v)
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
	if v := upd.List; v != nil {
		toUpd, args = append(toUpd, "list = ?"), append(args, kilonova.SerializeIntList(v))
	}
	return toUpd, args
}

type pblist struct {
	ID          int
	CreatedAt   time.Time `db:"created_at"`
	AuthorID    int       `db:"author_id"`
	Title       string
	Description string
	List        pgtype.Int8Array
}

func internalToPbList(list *pblist) *kilonova.ProblemList {
	ids := make([]int, 0, len(list.List.Elements))
	for _, id := range list.List.Elements {
		ids = append(ids, int(id.Int))
	}
	return &kilonova.ProblemList{
		ID:          list.ID,
		CreatedAt:   list.CreatedAt,
		Title:       list.Title,
		Description: list.Description,
		List:        ids,
	}
}
