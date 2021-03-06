package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/gosimple/slug"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.ProblemService = &ProblemService{}

type ProblemService struct {
	db *sqlx.DB
}

func (s *ProblemService) ProblemByID(ctx context.Context, id int) (*kilonova.Problem, error) {
	return s.problemByID(ctx, id)
}

func (s *ProblemService) Problems(ctx context.Context, filter kilonova.ProblemFilter) ([]*kilonova.Problem, error) {
	var pbs []*kilonova.Problem
	where, args := s.filterQueryMaker(&filter)
	query := s.db.Rebind("SELECT * FROM problems WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.db.SelectContext(ctx, &pbs, query, args...)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		err = nil
		pbs = []*kilonova.Problem{}
	}
	return pbs, err
}

const problemCreateQuery = `INSERT INTO problems (
	name, description, author_id, console_input, test_name, memory_limit, stack_limit, source_size, time_limit, visible, source_credits, author_credits, short_description, default_points, pb_type, helper_code, helper_code_lang
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING id;`

func (s *ProblemService) CreateProblem(ctx context.Context, p *kilonova.Problem) error {
	if p.Name == "" || p.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	if p.TestName == "" {
		p.TestName = slug.Make(p.Name)
	}
	if p.MemoryLimit == 0 {
		p.MemoryLimit = 65536 // 64MB
	}
	if p.StackLimit == 0 {
		p.StackLimit = 16384 // 16MB
	}
	if p.TimeLimit == 0 {
		p.TimeLimit = 1 // 1s
	}
	if p.SourceSize == 0 {
		p.SourceSize = 10000
	}
	if p.Type == kilonova.ProblemTypeNone {
		p.Type = kilonova.ProblemTypeClassic
	}
	var id int
	err := s.db.GetContext(ctx, &id, s.db.Rebind(problemCreateQuery), p.Name, p.Description, p.AuthorID, p.ConsoleInput, p.TestName, p.MemoryLimit, p.StackLimit, p.SourceSize, p.TimeLimit, p.Visible, p.SourceCredits, p.AuthorCredits, p.ShortDesc, p.DefaultPoints, p.Type, p.HelperCode, p.HelperCodeLang)
	if err == nil {
		p.ID = id
	}
	return err
}

const problemUpdateQuery = `UPDATE problems SET %s WHERE id = ?`

func (s *ProblemService) UpdateProblem(ctx context.Context, id int, upd kilonova.ProblemUpdate) error {
	toUpd, args := s.updateQueryMaker(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.db.Rebind(fmt.Sprintf(problemUpdateQuery, strings.Join(toUpd, ", ")))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

const bulkProblemUpdateQuery = `UPDATE problems SET %s WHERE %s`

func (s *ProblemService) BulkUpdateProblems(ctx context.Context, filter kilonova.ProblemFilter, upd kilonova.ProblemUpdate) error {
	toUpd, args := s.updateQueryMaker(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	where, args1 := s.filterQueryMaker(&filter)
	args = append(args, args1...)
	query := s.db.Rebind(fmt.Sprintf(bulkProblemUpdateQuery, strings.Join(toUpd, ", "), strings.Join(where, ", ")))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *ProblemService) DeleteProblem(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind("DELETE FROM problems WHERE id = ?"), id)
	return err
}

func (s *ProblemService) problemByID(ctx context.Context, id int) (*kilonova.Problem, error) {
	var pb kilonova.Problem
	err := s.db.GetContext(ctx, &pb, s.db.Rebind("SELECT * FROM problems WHERE id = ? LIMIT 1"), id)
	return &pb, err
}

func (s *ProblemService) filterQueryMaker(filter *kilonova.ProblemFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.IDs; v != nil && len(v) > 0 {
		where = append(where, "id IN (?"+strings.Repeat(",?", len(v)-1)+")")
		for _, el := range v {
			args = append(args, el)
		}
	}
	if v := filter.Name; v != nil {
		where, args = append(where, "lower(name) = lower(?)"), append(args, v)
	}
	if v := filter.AuthorID; v != nil {
		where, args = append(where, "author_id = ?"), append(args, v)
	}
	if v := filter.ConsoleInput; v != nil {
		where, args = append(where, "console_input = ?"), append(args, v)
	}
	if v := filter.Visible; v != nil {
		where, args = append(where, "visible = ?"), append(args, v)
	}
	if v := filter.LookingUserID; v != nil && *v >= 0 {
		where, args = append(where, "(visible = true OR author_id = ?)"), append(args, v)
	}
	return where, args
}

func (s *ProblemService) updateQueryMaker(upd *kilonova.ProblemUpdate) ([]string, []interface{}) {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}
	if v := upd.Description; v != nil {
		toUpd, args = append(toUpd, "description = ?"), append(args, v)
	}
	if v := upd.ShortDesc; v != nil {
		toUpd, args = append(toUpd, "short_description = ?"), append(args, v)
	}

	if v := upd.TestName; v != nil {
		toUpd, args = append(toUpd, "test_name = ?"), append(args, v)
	}
	if v := upd.AuthorID; v != nil {
		toUpd, args = append(toUpd, "author_id = ?"), append(args, v)
	}

	if v := upd.TimeLimit; v != nil {
		toUpd, args = append(toUpd, "time_limit = ?"), append(args, v)
	}
	if v := upd.MemoryLimit; v != nil {
		toUpd, args = append(toUpd, "memory_limit = ?"), append(args, v)
	}
	if v := upd.StackLimit; v != nil {
		toUpd, args = append(toUpd, "stack_limit = ?"), append(args, v)
	}

	if v := upd.DefaultPoints; v != nil {
		toUpd, args = append(toUpd, "default_points = ?"), append(args, v)
	}

	if v := upd.SourceCredits; v != nil {
		toUpd, args = append(toUpd, "source_credits = ?"), append(args, v)
	}
	if v := upd.AuthorCredits; v != nil {
		toUpd, args = append(toUpd, "author_credits = ?"), append(args, v)
	}

	if v := upd.Type; v != kilonova.ProblemTypeNone {
		toUpd, args = append(toUpd, "pb_type = ?"), append(args, v)
	}
	if v := upd.HelperCode; v != nil {
		toUpd, args = append(toUpd, "helper_code = ?"), append(args, v)
	}
	if v := upd.HelperCodeLang; v != nil {
		toUpd, args = append(toUpd, "helper_code_lang = ?"), append(args, v)
	}
	if v := upd.SubtaskString; v != nil {
		toUpd, args = append(toUpd, "subtasks = ?"), append(args, v)
	}

	if v := upd.ConsoleInput; v != nil {
		toUpd, args = append(toUpd, "console_input = ?"), append(args, v)
	}
	if v := upd.Visible; v != nil {
		toUpd, args = append(toUpd, "visible = ?"), append(args, v)
	}

	return toUpd, args
}

func NewProblemService(db *sqlx.DB) kilonova.ProblemService {
	return &ProblemService{db}
}
