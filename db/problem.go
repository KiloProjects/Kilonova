package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type dbProblem struct {
	ID          int        `db:"id"`
	CreatedAt   time.Time  `db:"created_at"`
	PublishedAt *time.Time `db:"published_at"`
	Name        string     `db:"name"`

	TestName     string `db:"test_name"`
	Visible      bool   `db:"visible"`
	VisibleTests bool   `db:"visible_tests"`

	DefaultPoints decimal.Decimal `db:"default_points"`

	// Limit stuff
	TimeLimit   float64 `db:"time_limit"`
	MemoryLimit int     `db:"memory_limit"`
	SourceSize  int     `db:"source_size"`

	SourceCredits string `db:"source_credits"`

	// Eval stuff
	ConsoleInput   bool  `db:"console_input"`
	DigitPrecision int32 `db:"digit_precision"`

	ScoringStrategy kilonova.ScoringType `db:"scoring_strategy"`
}

type dbScoredProblem struct {
	dbProblem
	ScoreUserID *int `db:"user_id"`
	IsEditor    bool `db:"pb_editor"`

	MaxScore  *decimal.Decimal `db:"score"`
	UNUSEDPOZ int              `db:"unused_position"`
}

func (s *DB) Problem(ctx context.Context, id int) (*kilonova.Problem, error) {
	var pb dbProblem
	err := Get(s.conn, ctx, &pb, "SELECT * FROM problems WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToProblem(&pb), nil
}

func (s *DB) ScoredProblem(ctx context.Context, problemID int, userID int) (*kilonova.ScoredProblem, error) {
	var pb dbScoredProblem
	err := Get(s.conn, ctx, &pb, `SELECT pbs.*, $2 AS user_id, COALESCE(ms.score, -1) AS score, (editors.user_id IS NOT NULL) AS pb_editor
FROM problems pbs 
	LEFT JOIN max_score_view ms ON (pbs.id = ms.problem_id AND ms.user_id = $2) 
	LEFT JOIN LATERAL (SELECT user_id FROM problem_editors editors WHERE pbs.id = editors.problem_id AND editors.user_id = $2 LIMIT 1) editors ON TRUE
WHERE pbs.id = $1 LIMIT 1`, problemID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToScoredProblem(&pb, userID)
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
	fb := newFilterBuilder()
	problemFilterQuery(&filter, fb)

	query := fmt.Sprintf("SELECT * FROM problems WHERE %s %s %s", fb.Where(), getProblemOrdering(filter.Ordering, filter.Descending), FormatLimitOffset(filter.Limit, filter.Offset))
	err := Select(s.conn, ctx, &pbs, query, fb.Args()...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Problem{}, nil
	}
	return mapper(pbs, s.internalToProblem), err
}

func (s *DB) ScoredProblems(ctx context.Context, filter kilonova.ProblemFilter, userID int) ([]*kilonova.ScoredProblem, error) {
	var pbs []*dbScoredProblem
	fb := newFilterBuilderFromPos(userID)
	problemFilterQuery(&filter, fb)

	query := `SELECT problems.*, $1 AS user_id, COALESCE(ms.score, -1) AS score, (editors.user_id IS NOT NULL) AS pb_editor
FROM problems 
	LEFT JOIN max_score_view ms ON (problems.id = ms.problem_id AND ms.user_id = $1)
	LEFT JOIN LATERAL (SELECT user_id FROM problem_editors editors WHERE problems.id = editors.problem_id AND editors.user_id = $1 LIMIT 1) editors ON TRUE
WHERE ` + fb.Where() + " " + getProblemOrdering(filter.Ordering, filter.Descending) + " " + FormatLimitOffset(filter.Limit, filter.Offset)
	err := Select(s.conn, ctx, &pbs, query, fb.Args()...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.internalToScoredProblems(pbs, userID), err
}

func (s *DB) CountProblems(ctx context.Context, filter kilonova.ProblemFilter) (int, error) {
	fb := newFilterBuilder()
	problemFilterQuery(&filter, fb)
	var val int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM problems WHERE "+fb.Where(), fb.Args()...).Scan(&val)
	return val, err
}

func (s *DB) ContestProblems(ctx context.Context, contestID int) ([]*kilonova.Problem, error) {
	var pbs []*dbProblem
	err := Select(s.conn, ctx, &pbs, "SELECT pbs.* FROM problems pbs, contest_problems cpbs WHERE cpbs.problem_id = pbs.id AND cpbs.contest_id = $1 ORDER BY cpbs.position ASC", contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Problem{}, nil
	}
	return mapper(pbs, s.internalToProblem), err
}

func (s *DB) ScoredContestProblems(ctx context.Context, contestID int, userID int, freezeTime *time.Time) ([]*kilonova.ScoredProblem, error) {
	var pbs []*dbScoredProblem
	err := Select(s.conn, ctx, &pbs, `SELECT pbs.*, cpbs.position AS unused_position, ms.user_id, ms.score, (editors.user_id IS NOT NULL) AS pb_editor
FROM (problems pbs INNER JOIN contest_problems cpbs ON cpbs.problem_id = pbs.id) 
	LEFT JOIN contest_max_scores($1, $3) ms ON (pbs.id = ms.problem_id AND ms.user_id = $2)
	LEFT JOIN LATERAL (SELECT user_id FROM problem_editors editors WHERE pbs.id = editors.problem_id AND editors.user_id = $2 LIMIT 1) editors ON TRUE
WHERE cpbs.contest_id = $1 
ORDER BY cpbs.position ASC`, contestID, userID, freezeTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.internalToScoredProblems(pbs, userID), err
}

const problemCreateQuery = `INSERT INTO problems (
	name, console_input, test_name, memory_limit, source_size, time_limit, visible, source_credits, default_points
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING id;`

func (s *DB) CreateProblem(ctx context.Context, p *kilonova.Problem, authorID int) error {
	if p.Name == "" || authorID == 0 {
		return kilonova.ErrMissingRequired
	}
	if p.TestName == "" {
		p.TestName = kilonova.MakeSlug(p.Name)
	}
	if p.MemoryLimit == 0 {
		p.MemoryLimit = 65536 // 64MB
	}
	if p.TimeLimit == 0 {
		p.TimeLimit = 1 // 1s
	}
	if p.SourceSize == 0 {
		p.SourceSize = kilonova.DefaultSourceSize
	}
	var id int
	err := s.conn.QueryRow(ctx, problemCreateQuery, p.Name, p.ConsoleInput, p.TestName, p.MemoryLimit, p.SourceSize, p.TimeLimit, p.Visible, p.SourceCredits, p.DefaultPoints).Scan(&id)
	if err == nil {
		p.ID = id
	}
	if err != nil {
		return err
	}
	return s.AddProblemEditor(ctx, id, authorID)
}

func (s *DB) UpdateProblem(ctx context.Context, id int, upd kilonova.ProblemUpdate) error {
	return s.BulkUpdateProblems(ctx, kilonova.ProblemFilter{ID: &id}, upd)
}

func (s *DB) BulkUpdateProblems(ctx context.Context, filter kilonova.ProblemFilter, upd kilonova.ProblemUpdate) error {
	ub := newUpdateBuilder()
	problemUpdateQuery(&upd, ub)
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	problemFilterQuery(&filter, fb)
	_, err := s.conn.Exec(ctx, "UPDATE problems SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) DeleteProblem(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM problems WHERE id = $1", id)
	return err
}

func problemFilterQuery(filter *kilonova.ProblemFilter, fb *filterBuilder) {
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		fb.AddConstraint("id = -1")
	}
	if v := filter.IDs; len(v) > 0 {
		fb.AddConstraint("id = ANY(%s)", v)
	}
	if v := filter.Name; v != nil {
		fb.AddConstraint("lower(name) = lower(%s)", v)
	}
	if v := filter.FuzzyName; v != nil {
		fb.AddConstraint("position(lower(unaccent(%s)) in format('#%%s %%s', id, lower(unaccent(name)))) > 0", v)
	}
	if v := filter.DeepListID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM problem_list_deep_problems WHERE list_id = %s AND problem_id = problems.id)", v)
	}
	if v := filter.ConsoleInput; v != nil {
		fb.AddConstraint("console_input = %s", v)
	}
	if v := filter.Visible; v != nil {
		fb.AddConstraint("visible = %s", v)
	}
	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		if filter.LookEditor {
			fb.AddConstraint("EXISTS (SELECT 1 FROM problem_editors WHERE user_id = %s AND problem_id = problems.id)", id)
		} else {
			fb.AddConstraint("EXISTS (SELECT 1 FROM visible_pbs(%s) WHERE problem_id = problems.id)", id)
		}
	}
	if v := filter.EditorUserID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM problem_user_access WHERE user_id = %s AND problem_id = problems.id)", v)
	}
	if v := filter.AttachmentID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM problem_attachments_m2m WHERE attachment_id = %s AND problem_id = id)", v)
	}
	if v := filter.Tags; len(v) > 0 {
		for _, group := range v {
			q := "EXISTS (SELECT 1 FROM problem_tags WHERE problem_id = problems.id AND tag_id = ANY(%s))"
			if group.Negate {
				q = "NOT " + q
			}
			fb.AddConstraint(q, group.TagIDs)
		}
	}

	if v := filter.Language; v != nil {
		fb.AddConstraint(`EXISTS (SELECT 1 FROM attachments, problem_attachments_m2m m2m WHERE attachments.name LIKE CONCAT('statement-', %s::text, '.%%') AND problem_id = problems.id AND m2m.attachment_id = attachments.id)`, filter.Language)
	}

	if v := filter.SolvedBy; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM max_score_view WHERE score = 100 AND problem_id = problems.id AND user_id = %s)", v)
	}
	if v := filter.AttemptedBy; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM max_score_view WHERE score != 100 AND score >= 0 AND problem_id = problems.id AND user_id = %s)", v)
	}

	if filter.Unassociated {
		fb.AddConstraint("NOT EXISTS (SELECT 1 FROM problem_list_problems WHERE problem_id = problems.id)")
	}
}

func problemUpdateQuery(upd *kilonova.ProblemUpdate, ub *updateBuilder) {
	if v := upd.Name; v != nil {
		ub.AddUpdate("name = %s", v)
	}

	if v := upd.TestName; v != nil {
		ub.AddUpdate("test_name = %s", strings.TrimSpace(*v))
	}

	if v := upd.TimeLimit; v != nil {
		ub.AddUpdate("time_limit = %s", v)
	}
	if v := upd.MemoryLimit; v != nil {
		ub.AddUpdate("memory_limit = %s", v)
	}

	if v := upd.DefaultPoints; v != nil {
		ub.AddUpdate("default_points = %s", v)
	}

	if v := upd.SourceCredits; v != nil {
		ub.AddUpdate("source_credits = %s", strings.TrimSpace(*v))
	}

	if v := upd.ConsoleInput; v != nil {
		ub.AddUpdate("console_input = %s", v)
	}
	if v := upd.Visible; v != nil {
		ub.AddUpdate("visible = %s", v)
		// if is set to visible
		if *v {
			// Published at - first time it was set visible
			ub.AddUpdate("published_at = COALESCE(published_at, NOW())")
		}
	}
	if v := upd.VisibleTests; v != nil {
		ub.AddUpdate("visible_tests = %s", v)
	}
	if v := upd.ScoringStrategy; v != kilonova.ScoringTypeNone {
		ub.AddUpdate("scoring_strategy = %s", v)
	}
	if v := upd.ScorePrecision; v != nil {
		ub.AddUpdate("digit_precision = %s", v)
	}
}

// Access rights

func (s *DB) AddProblemEditor(ctx context.Context, pbid int, uid int) error {
	return s.addAccess(ctx, "problem_user_access", "problem_id", pbid, uid, "editor")
}

func (s *DB) AddProblemViewer(ctx context.Context, pbid int, uid int) error {
	return s.addAccess(ctx, "problem_user_access", "problem_id", pbid, uid, "viewer")
}

func (s *DB) StripProblemAccess(ctx context.Context, pbid int, uid int) error {
	return s.removeAccess(ctx, "problem_user_access", "problem_id", pbid, uid)
}

func (s *DB) ProblemEditors(ctx context.Context, pbid int) ([]*User, error) {
	return s.getAccessUsers(ctx, "problem_user_access", "problem_id", pbid, accessEditor)
}

func (s *DB) ProblemViewers(ctx context.Context, pbid int) ([]*User, error) {
	return s.getAccessUsers(ctx, "problem_user_access", "problem_id", pbid, accessViewer)
}

func (s *DB) internalToProblem(pb *dbProblem) *kilonova.Problem {
	if pb == nil {
		return nil
	}

	return &kilonova.Problem{
		ID:        pb.ID,
		CreatedAt: pb.CreatedAt,
		Name:      pb.Name,
		TestName:  pb.TestName,

		Visible:      pb.Visible,
		VisibleTests: pb.VisibleTests,

		DefaultPoints: pb.DefaultPoints,

		TimeLimit:   pb.TimeLimit,
		MemoryLimit: pb.MemoryLimit,
		SourceSize:  pb.SourceSize,

		SourceCredits: pb.SourceCredits,

		ConsoleInput:   pb.ConsoleInput,
		ScorePrecision: pb.DigitPrecision,

		PublishedAt:     pb.PublishedAt,
		ScoringStrategy: pb.ScoringStrategy,
	}
}

func (s *DB) internalToScoredProblem(spb *dbScoredProblem, userID int) (*kilonova.ScoredProblem, error) {
	pb := s.internalToProblem(&spb.dbProblem)
	var uid *int
	if userID > 0 {
		uid = &userID
	}
	return &kilonova.ScoredProblem{
		Problem: *pb,

		ScoreUserID: uid,
		MaxScore:    spb.MaxScore,
		IsEditor:    spb.IsEditor,
	}, nil
}

func (s *DB) internalToScoredProblems(spbs []*dbScoredProblem, userID int) []*kilonova.ScoredProblem {
	if len(spbs) == 0 {
		return []*kilonova.ScoredProblem{}
	}
	rez := make([]*kilonova.ScoredProblem, len(spbs))
	for i := range rez {
		var err error
		rez[i], err = s.internalToScoredProblem(spbs[i], userID)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().WithOptions(zap.AddCallerSkip(1)).Warn(err)
		}
	}
	return rez
}

func getProblemOrdering(ordering string, descending bool) string {
	ord := " ASC"
	if descending {
		ord = " DESC"
	}
	switch ordering {
	case "name":
		return "ORDER BY name" + ord + ", id ASC"
	case "published_at":
		return "ORDER BY published_at" + ord + " NULLS LAST, id ASC"
	case "hot":
		return "ORDER BY (SELECT hot_cnt FROM hot_problems WHERE problem_id = id) " + ord + " NULLS LAST, published_at" + ord + " NULLS LAST, id ASC"
	default:
		return "ORDER BY id" + ord
	}
}
