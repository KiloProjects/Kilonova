package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/gosimple/slug"
	"go.uber.org/zap"
)

type dbProblem struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	Name      string    `db:"name"`

	TestName      string `db:"test_name"`
	Visible       bool   `db:"visible"`
	DefaultPoints int    `db:"default_points"`

	// Limit stuff
	TimeLimit   float64 `db:"time_limit"`
	MemoryLimit int     `db:"memory_limit"`
	SourceSize  int     `db:"source_size"`

	SourceCredits string `db:"source_credits"`
	AuthorCredits string `db:"author_credits"`

	// Eval stuff
	ConsoleInput bool `db:"console_input"`
}

type dbScoredProblem struct {
	dbProblem
	ScoreUserID *int `db:"user_id"`
	MaxScore    *int `db:"score"`
}

func (s *DB) Problem(ctx context.Context, id int) (*kilonova.Problem, error) {
	var pb dbProblem
	err := s.conn.GetContext(ctx, &pb, "SELECT * FROM problems WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToProblem(ctx, &pb)
}

func (s *DB) ScoredProblem(ctx context.Context, problemID int, userID int) (*kilonova.ScoredProblem, error) {
	var pb dbScoredProblem
	err := s.conn.GetContext(ctx, &pb, `SELECT pbs.*, ms.user_id, ms.score 
FROM problems pbs LEFT JOIN max_score_view ms ON (pbs.id = ms.problem_id AND ms.user_id = $2) 
WHERE pbs.id = $1 LIMIT 1`, problemID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToScoredProblem(ctx, &pb, userID)
}

func (s *DB) VisibleProblem(ctx context.Context, id int, user *kilonova.UserBrief) (*kilonova.Problem, error) {
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{ID: &id, LookingUser: user, Look: true})
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
	return mapperCtx(ctx, pbs, s.internalToProblem), err
}

func (s *DB) ScoredProblems(ctx context.Context, filter kilonova.ProblemFilter, userID int) ([]*kilonova.ScoredProblem, error) {
	var pbs []*dbScoredProblem
	where, args := problemFilterQuery(&filter)
	query := s.conn.Rebind(`SELECT problems.*, ms.user_id, ms.score 
FROM problems LEFT JOIN max_score_view ms ON (problems.id = ms.problem_id AND ms.user_id = ?) 
WHERE ` + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.conn.SelectContext(ctx, &pbs, query, append([]any{userID}, args...)...)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.internalToScoredProblems(ctx, pbs, userID), err
}

func (s *DB) ContestProblems(ctx context.Context, contestID int) ([]*kilonova.Problem, error) {
	var pbs []*dbProblem
	err := s.conn.SelectContext(ctx, &pbs, "SELECT pbs.* FROM problems pbs, contest_problems cpbs WHERE cpbs.problem_id = pbs.id AND cpbs.contest_id = $1 ORDER BY cpbs.position ASC", contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Problem{}, nil
	}
	return mapperCtx(ctx, pbs, s.internalToProblem), err
}

func (s *DB) ScoredContestProblems(ctx context.Context, contestID int, userID int) ([]*kilonova.ScoredProblem, error) {
	var pbs []*dbScoredProblem
	err := s.conn.SelectContext(ctx, &pbs, `SELECT pbs.*, ms.user_id, ms.score 
FROM (problems pbs INNER JOIN contest_problems cpbs ON cpbs.problem_id = pbs.id) LEFT JOIN max_score_contest_view ms ON (pbs.id = ms.problem_id AND cpbs.contest_id = ms.contest_id AND ms.user_id = $2)
WHERE cpbs.contest_id = $1 
ORDER BY cpbs.position ASC`, contestID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.internalToScoredProblems(ctx, pbs, userID), err
}

const problemCreateQuery = `INSERT INTO problems (
	name, console_input, test_name, memory_limit, source_size, time_limit, visible, source_credits, author_credits, default_points
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING id;`

func (s *DB) CreateProblem(ctx context.Context, p *kilonova.Problem, authorID int) error {
	if p.Name == "" || authorID == 0 {
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
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(problemCreateQuery), p.Name, p.ConsoleInput, p.TestName, p.MemoryLimit, p.SourceSize, p.TimeLimit, p.Visible, p.SourceCredits, p.AuthorCredits, p.DefaultPoints)
	if err == nil {
		p.ID = id
	}
	if err != nil {
		return err
	}
	return s.AddProblemEditor(ctx, id, authorID)
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
	_, err := s.conn.ExecContext(ctx, "DELETE FROM problems WHERE id = $1", id)
	return err
}

func problemFilterQuery(filter *kilonova.ProblemFilter) ([]string, []any) {
	where, args := []string{"1 = 1"}, []any{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		where = append(where, "id = -1")
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
	if v := filter.ConsoleInput; v != nil {
		where, args = append(where, "console_input = ?"), append(args, v)
	}
	if v := filter.Visible; v != nil {
		where, args = append(where, "visible = ?"), append(args, v)
	}
	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		args = append(args, id)
		if filter.LookEditor {
			where = append(where, "EXISTS (SELECT 1 FROM problem_editors WHERE user_id = ? AND problem_id = problems.id)")
		} else {
			where = append(where, "EXISTS (SELECT 1 FROM problem_viewers WHERE user_id = ? AND problem_id = problems.id)")
		}
	}

	if filter.Unassociated {
		where = append(where, "NOT EXISTS (SELECT 1 FROM problem_list_problems WHERE problem_id = problems.id)")
	}
	return where, args
}

func problemUpdateQuery(upd *kilonova.ProblemUpdate) ([]string, []any) {
	toUpd, args := []string{}, []any{}
	if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}

	if v := upd.TestName; v != nil {
		toUpd, args = append(toUpd, "test_name = ?"), append(args, v)
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

// Access rights

func (s *DB) problemAccessRights(ctx context.Context, pbid int) ([]*userAccess, error) {
	return s.getAccess(ctx, "problem_user_access", "problem_id", pbid)
}

func (s *DB) AddProblemEditor(ctx context.Context, pbid int, uid int) error {
	return s.addAccess(ctx, "problem_user_access", "problem_id", pbid, uid, "editor")
}

func (s *DB) AddProblemViewer(ctx context.Context, pbid int, uid int) error {
	return s.addAccess(ctx, "problem_user_access", "problem_id", pbid, uid, "viewer")
}

func (s *DB) StripProblemAccess(ctx context.Context, pbid int, uid int) error {
	return s.removeAccess(ctx, "problem_user_access", "problem_id", pbid, uid)
}

func (s *DB) internalToProblem(ctx context.Context, pb *dbProblem) (*kilonova.Problem, error) {
	if pb == nil {
		return nil, nil
	}

	rights, err := s.problemAccessRights(ctx, pb.ID)
	if err != nil {
		return nil, err
	}

	var editors, viewers []int
	for _, right := range rights {
		switch right.Access {
		case accessEditor:
			editors = append(editors, right.UserID)
		case accessViewer:
			viewers = append(viewers, right.UserID)
		default:
			zap.S().Warn("Unknown access rank", zap.String("right", string(right.Access)))
		}
	}
	if editors == nil {
		editors = []int{}
	}
	if viewers == nil {
		viewers = []int{}
	}

	return &kilonova.Problem{
		ID:        pb.ID,
		CreatedAt: pb.CreatedAt,
		Name:      pb.Name,
		TestName:  pb.TestName,

		Visible: pb.Visible,
		Editors: editors,
		Viewers: viewers,

		DefaultPoints: pb.DefaultPoints,

		TimeLimit:   pb.TimeLimit,
		MemoryLimit: pb.MemoryLimit,
		SourceSize:  pb.SourceSize,

		SourceCredits: pb.SourceCredits,
		AuthorCredits: pb.AuthorCredits,

		ConsoleInput: pb.ConsoleInput,
	}, nil
}

func (s *DB) internalToScoredProblem(ctx context.Context, spb *dbScoredProblem, userID int) (*kilonova.ScoredProblem, error) {
	pb, err := s.internalToProblem(ctx, &spb.dbProblem)
	if err != nil {
		return nil, err
	}
	var uid *int
	if userID > 0 {
		uid = &userID
	}
	return &kilonova.ScoredProblem{
		Problem: *pb,

		ScoreUserID: uid,
		MaxScore:    spb.MaxScore,
	}, nil
}

func (s *DB) internalToScoredProblems(ctx context.Context, spbs []*dbScoredProblem, userID int) []*kilonova.ScoredProblem {
	if len(spbs) == 0 {
		return []*kilonova.ScoredProblem{}
	}
	rez := make([]*kilonova.ScoredProblem, len(spbs))
	for i := range rez {
		var err error
		rez[i], err = s.internalToScoredProblem(ctx, spbs[i], userID)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().WithOptions(zap.AddCallerSkip(1)).Warn(err)
		}
	}
	return rez
}
