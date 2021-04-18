package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.ProblemListService = &ProblemListService{}

type ProblemListService struct {
	db *sqlx.DB
}

func (p *ProblemListService) ProblemList(ctx context.Context, id int) (*kilonova.ProblemList, error) {
	var pblist pblist
	err := p.db.GetContext(ctx, &pblist, p.db.Rebind("SELECT * FROM problem_lists WHERE id = ? LIMIT 1"), id)
	return internalToPbList(&pblist), err
}

func (p *ProblemListService) ProblemLists(ctx context.Context, filter kilonova.ProblemListFilter) ([]*kilonova.ProblemList, error) {
	var lists []*pblist
	where, args := p.filterQueryMaker(&filter)
	query := "SELECT * FROM problem_lists WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset)
	query = p.db.Rebind(query)
	err := p.db.SelectContext(ctx, &lists, query, args...)

	outLists := make([]*kilonova.ProblemList, 0, len(lists))
	for _, el := range lists {
		outLists = append(outLists, internalToPbList(el))
	}
	return outLists, err
}

const createProblemListQuery = "INSERT INTO problem_lists (author_id, title, description, list) VALUES (?, ?, ?, ?) RETURNING id;"

func (p *ProblemListService) CreateProblemList(ctx context.Context, list *kilonova.ProblemList) error {
	if list.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := p.db.GetContext(ctx, &id, p.db.Rebind(createProblemListQuery), list.AuthorID, list.Title, list.Description, kilonova.SerializeIntList(list.List))
	if err == nil {
		list.ID = id
	}
	return err
}

func (p *ProblemListService) UpdateProblemList(ctx context.Context, id int, upd kilonova.ProblemListUpdate) error {
	toUpd, args := p.updateQueryMaker(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := p.db.Rebind(fmt.Sprintf(`UPDATE problem_lists SET %s WHERE id = ?;`, strings.Join(toUpd, ", ")))
	_, err := p.db.ExecContext(ctx, query, args...)
	return err
}

func (p *ProblemListService) DeleteProblemList(ctx context.Context, id int) error {
	_, err := p.db.ExecContext(ctx, p.db.Rebind("DELETE FROM problem_lists WHERE id = ?"), id)
	return err
}

func (p *ProblemListService) filterQueryMaker(filter *kilonova.ProblemListFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.AuthorID; v != nil {
		where, args = append(where, "author_id = ?"), append(args, v)
	}

	return where, args
}

func (p *ProblemListService) updateQueryMaker(upd *kilonova.ProblemListUpdate) ([]string, []interface{}) {
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
	List        string
}

func internalToPbList(list *pblist) *kilonova.ProblemList {
	return &kilonova.ProblemList{
		ID:          list.ID,
		CreatedAt:   list.CreatedAt,
		Title:       list.Title,
		Description: list.Description,
		List:        kilonova.DeserializeIntList(list.List),
	}
}

func NewProblemListService(db *sqlx.DB) kilonova.ProblemListService {
	return &ProblemListService{db}
}
