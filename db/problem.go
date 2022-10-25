package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/gosimple/slug"
)

func (s *DB) Problem(ctx context.Context, id int) (*kilonova.Problem, error) {
	var pb dbProblem
	err := s.conn.GetContext(ctx, &pb, s.conn.Rebind("SELECT * FROM problems WHERE id = ? LIMIT 1"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return internalToProblem(&pb), err
}

func (s *DB) ProblemByName(ctx context.Context, name string) (*kilonova.Problem, error) {
	var pb dbProblem
	err := s.conn.GetContext(ctx, &pb, s.conn.Rebind("SELECT * FROM problems WHERE lower(name) = lower(?) LIMIT 1"), name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return internalToProblem(&pb), err
}

func (s *DB) VisibleProblem(ctx context.Context, id int, user *kilonova.UserBrief) (*kilonova.Problem, error) {
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{ID: &id, LookingUser: user})
	if err != nil || len(pbs) == 0 {
		return nil, nil
	}
	return pbs[0], nil
}

func (s *DB) Problems(ctx context.Context, filter kilonova.ProblemFilter) ([]*kilonova.Problem, error) {
	var pbs []*dbProblem
	where, args := problemFilterQuery(&filter)
	query := s.conn.Rebind("SELECT * FROM problems WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.conn.SelectContext(ctx, &pbs, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Problem{}, nil
	}
	return internalToProblems(pbs), err
}

const problemCreateQuery = `INSERT INTO problems (
	name, description, author_id, console_input, test_name, memory_limit, source_size, time_limit, visible, source_credits, author_credits, short_description, default_points
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING id;`

func (s *DB) CreateProblem(ctx context.Context, p *kilonova.Problem) error {
	if p.Name == "" || p.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	if p.TestName == "" {
		p.TestName = slug.Make(p.Name)
	}
	if p.MemoryLimit == 0 {
		p.MemoryLimit = 65536 // 64MB
	}
	if p.TimeLimit == 0 {
		p.TimeLimit = 1 // 1s
	}
	if p.SourceSize == 0 {
		p.SourceSize = 10000
	}
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(problemCreateQuery), p.Name, p.Description, p.AuthorID, p.ConsoleInput, p.TestName, p.MemoryLimit, p.SourceSize, p.TimeLimit, p.Visible, p.SourceCredits, p.AuthorCredits, p.ShortDesc, p.DefaultPoints)
	if err == nil {
		p.ID = id
	}
	return err
}

const problemUpdateStatement = `UPDATE problems SET %s WHERE id = ?`

func (s *DB) UpdateProblem(ctx context.Context, id int, upd kilonova.ProblemUpdate) error {
	toUpd, args := problemUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.conn.Rebind(fmt.Sprintf(problemUpdateStatement, strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

const bulkProblemUpdateStatement = `UPDATE problems SET %s WHERE %s`

func (s *DB) BulkUpdateProblems(ctx context.Context, filter kilonova.ProblemFilter, upd kilonova.ProblemUpdate) error {
	toUpd, args := problemUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	where, args1 := problemFilterQuery(&filter)
	args = append(args, args1...)
	query := s.conn.Rebind(fmt.Sprintf(bulkProblemUpdateStatement, strings.Join(toUpd, ", "), strings.Join(where, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) DeleteProblem(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM problems WHERE id = ?"), id)
	return err
}

func problemFilterQuery(filter *kilonova.ProblemFilter) ([]string, []any) {
	where, args := []string{"1 = 1"}, []any{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.IDs; len(v) > 0 {
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
	if user := filter.LookingUser; user != nil {
		var id int
		if user != nil {
			id = user.ID
			if user.Admin {
				id = -1
			}
		}
		if id >= 0 {
			where, args = append(where, "(visible = true OR author_id = ?)"), append(args, id)
		}
	}
	return where, args
}

func problemUpdateQuery(upd *kilonova.ProblemUpdate) ([]string, []any) {
	toUpd, args := []string{}, []any{}
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

	if v := upd.DefaultPoints; v != nil {
		toUpd, args = append(toUpd, "default_points = ?"), append(args, v)
	}

	if v := upd.SourceCredits; v != nil {
		toUpd, args = append(toUpd, "source_credits = ?"), append(args, v)
	}
	if v := upd.AuthorCredits; v != nil {
		toUpd, args = append(toUpd, "author_credits = ?"), append(args, v)
	}

	if v := upd.ConsoleInput; v != nil {
		toUpd, args = append(toUpd, "console_input = ?"), append(args, v)
	}
	if v := upd.Visible; v != nil {
		toUpd, args = append(toUpd, "visible = ?"), append(args, v)
	}

	return toUpd, args
}

// TODO: Move to BaseAPI
func (db *DB) SolvedProblems(ctx context.Context, uid int) ([]*kilonova.Problem, error) {
	ids, err := db.solvedProblems(ctx, uid)
	if err != nil {
		return nil, err
	}
	var pbs = make([]*kilonova.Problem, 0, len(ids))
	for _, id := range ids {
		pb, err := db.Problem(ctx, id)
		if err != nil {
			log.Printf("Couldn't get solved problem %d: %s\n", id, err)
		} else {
			pbs = append(pbs, pb)
		}
	}
	return pbs, nil
}

type dbProblem struct {
	ID            int       `db:"id"`
	CreatedAt     time.Time `db:"created_at"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	ShortDesc     string    `db:"short_description"`
	TestName      string    `db:"test_name"`
	AuthorID      int       `db:"author_id"`
	Visible       bool      `db:"visible"`
	DefaultPoints int       `db:"default_points"`

	// Limit stuff
	TimeLimit   float64 `db:"time_limit"`
	MemoryLimit int     `db:"memory_limit"`
	SourceSize  int     `db:"source_size"`

	SourceCredits string `db:"source_credits"`
	AuthorCredits string `db:"author_credits"`

	// Eval stuff
	ConsoleInput bool `db:"console_input"`
}

func internalToProblems(pbs []*dbProblem) []*kilonova.Problem {
	return mapper(pbs, internalToProblem)
}

func internalToProblem(pb *dbProblem) *kilonova.Problem {
	if pb == nil {
		return nil
	}
	return &kilonova.Problem{
		ID:          pb.ID,
		CreatedAt:   pb.CreatedAt,
		Name:        pb.Name,
		Description: pb.Description,
		ShortDesc:   pb.ShortDesc,
		TestName:    pb.TestName,

		AuthorID:      pb.AuthorID,
		Visible:       pb.Visible,
		DefaultPoints: pb.DefaultPoints,

		TimeLimit:   pb.TimeLimit,
		MemoryLimit: pb.MemoryLimit,
		SourceSize:  pb.SourceSize,

		SourceCredits: pb.SourceCredits,
		AuthorCredits: pb.AuthorCredits,

		ConsoleInput: pb.ConsoleInput,
	}
}
