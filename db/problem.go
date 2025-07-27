package db

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
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

	ScoreScale decimal.Decimal `db:"leaderboard_score_scale"`

	// Eval stuff
	ConsoleInput   bool  `db:"console_input"`
	DigitPrecision int32 `db:"digit_precision"`

	ScoringStrategy kilonova.ScoringType `db:"scoring_strategy"`

	TaskType kilonova.TaskType `db:"task_type"`

	CommunicationProcesses int `db:"communication_num_processes"`
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

func (s *DB) VisibleProblem(ctx context.Context, id int, user *kilonova.UserBrief) (*kilonova.Problem, error) {
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{ID: &id, LookingUser: user, Look: true})
	if err != nil || len(pbs) == 0 {
		return nil, nil
	}
	return pbs[0], nil
}

func (s *DB) Problems(ctx context.Context, filter kilonova.ProblemFilter) ([]*kilonova.Problem, error) {
	var pbs []*dbProblem
	sb := sq.Select("*").From("problems").Where(problemFilterQuery(&filter)).OrderBy(getProblemOrdering(filter.Ordering, filter.Descending))
	qb := LimitOffset(sb, filter.Limit, filter.Offset)
	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	err = Select(s.conn, ctx, &pbs, query, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Problem{}, nil
	}
	return mapper(pbs, s.internalToProblem), err
}

func (s *DB) ScoredProblems(ctx context.Context, filter kilonova.ProblemFilter, scoreUID, editorUID int) ([]*kilonova.ScoredProblem, error) {
	var pbs []*dbScoredProblem
	sb := sq.Select("problems.*").Column(sq.Alias(sq.Expr("?::integer", scoreUID), "user_id")).Column("COALESCE(ms.score, -1) AS score").Column("(editors.user_id IS NOT NULL) AS pb_editor")
	sb = sb.From("problems").LeftJoin("max_scores ms ON (problems.id = ms.problem_id AND ms.user_id = ?)", scoreUID).LeftJoin("LATERAL (SELECT user_id FROM problem_editors editors WHERE problems.id = editors.problem_id AND editors.user_id = ? LIMIT 1) editors ON TRUE", editorUID)
	sb = sb.Where(problemFilterQuery(&filter)).OrderBy(getProblemOrdering(filter.Ordering, filter.Descending))
	sb = LimitOffset(sb, filter.Limit, filter.Offset)
	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	err = Select(s.conn, ctx, &pbs, query, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.internalToScoredProblems(ctx, pbs, scoreUID), err
}

func (s *DB) CountProblems(ctx context.Context, filter kilonova.ProblemFilter) (int, error) {
	sb := sq.Select("COUNT(*)").From("problems").Where(problemFilterQuery(&filter))
	query, args, err := sb.ToSql()
	if err != nil {
		return 0, err
	}
	var val int
	err = s.conn.QueryRow(ctx, query, args...).Scan(&val)
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
	return s.internalToScoredProblems(ctx, pbs, userID), err
}

const problemCreateQuery = `INSERT INTO problems (
	name, console_input, test_name, memory_limit, source_size, time_limit, visible, source_credits, default_points, task_type
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
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
		p.SourceSize = kilonova.DefaultSourceSize.Value()
	}
	if p.TaskType == kilonova.TaskTypeNone {
		p.TaskType = kilonova.TaskTypeBatch
	}
	var id int
	err := s.conn.QueryRow(ctx, problemCreateQuery, p.Name, p.ConsoleInput, p.TestName, p.MemoryLimit, p.SourceSize, p.TimeLimit, p.Visible, p.SourceCredits, p.DefaultPoints, p.TaskType).Scan(&id)
	if err == nil {
		p.ID = id
	}
	if err != nil {
		return err
	}
	return s.AddProblemEditor(ctx, id, authorID)
}

func (s *DB) UpdateProblem(ctx context.Context, id int, upd kilonova.ProblemUpdate) ([]int, error) {
	return s.BulkUpdateProblems(ctx, kilonova.ProblemFilter{ID: &id}, upd)
}

type problemUpdateInfo struct {
	ID          int        `db:"id"`
	Visible     bool       `db:"visible"`
	PublishedAt *time.Time `db:"published_at"`
	Now         time.Time  `db:"now"`
}

func (s *DB) BulkUpdateProblems(ctx context.Context, filter kilonova.ProblemFilter, upd kilonova.ProblemUpdate) ([]int, error) {
	ub := sq.Update("problems").Where(problemFilterQuery(&filter)).Suffix("RETURNING id, visible, published_at, NOW() as now")
	ub = problemUpdateQuery(&upd, ub)
	query, args, err := ub.ToSql()
	if err != nil {
		if err.Error() == "update statements must have at least one Set clause" {
			return nil, kilonova.ErrNoUpdates
		}
		return nil, err
	}
	rows, _ := s.conn.Query(ctx, query, args...)
	updatedInfo, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[problemUpdateInfo])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var updatedProblems = make([]int, 0, len(updatedInfo))
	for _, pb := range updatedInfo {
		if pb.Visible && pb.PublishedAt != nil && pb.PublishedAt.Equal(pb.Now) {
			updatedProblems = append(updatedProblems, pb.ID)
		}
	}
	slices.Sort(updatedProblems)

	return updatedProblems, nil
}

func (s *DB) DeleteProblem(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM problems WHERE id = $1", id)
	return err
}

func problemFilterQuery(filter *kilonova.ProblemFilter) sq.And {
	sb := sq.And{}
	if v := filter.ID; v != nil {
		sb = append(sb, sq.Eq{"id": v})
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		sb = append(sb, sq.Expr("0 = 1"))
	}
	if v := filter.IDs; len(v) > 0 {
		sb = append(sb, sq.Eq{"id": v})
	}
	if v := filter.Name; v != nil {
		sb = append(sb, sq.Expr("lower(name) = lower(?)", v))
	}
	if v := filter.FuzzyName; v != nil {
		sb = append(sb, sq.Expr("position(lower(unaccent(?)) in format('#%s %s', id, lower(unaccent(name)))) > 0", v))
	}
	if v := filter.DeepListID; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM problem_list_deep_problems WHERE list_id = ? AND problem_id = problems.id)", v))
	}
	if v := filter.ConsoleInput; v != nil {
		sb = append(sb, sq.Eq{"console_input": v})
	}
	if v := filter.Visible; v != nil {
		sb = append(sb, sq.Eq{"visible": v})
	}
	if filter.Look {
		var id int
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		if filter.LookEditor {
			sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM problem_editors WHERE user_id = ? AND problem_id = problems.id)", id))
		} else if filter.LookFullyVisible {
			sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM persistently_visible_pbs(?) WHERE problem_id = problems.id)", id))
		} else {
			sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM visible_pbs(?) WHERE problem_id = problems.id)", id))
		}
	}
	if v := filter.EditorUserID; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM problem_user_access WHERE user_id = ? AND problem_id = problems.id)", v))
	}
	if v := filter.AttachmentID; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM problem_attachments_m2m WHERE attachment_id = ? AND problem_id = problems.id)", v))
	}
	if v := filter.ContestID; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM contest_problems WHERE contest_id = ? AND problem_id = problems.id)", v))

	}
	if v := filter.Tags; len(v) > 0 {
		for _, group := range v {
			prefix := ""
			if group.Negate {
				prefix = "NOT "
			}
			sb = append(sb, sq.Expr(prefix+"EXISTS (SELECT 1 FROM problem_tags WHERE problem_id = problems.id AND tag_id = ANY(?))", group.TagIDs))
		}
	}

	if v := filter.Language; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM attachments, problem_attachments_m2m m2m WHERE attachments.name LIKE CONCAT('statement-', ?::text, '%') AND problem_id = problems.id AND m2m.attachment_id = attachments.id)", filter.Language))
	}

	if v := filter.SolvedBy; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM max_scores WHERE score = 100 AND problem_id = problems.id AND user_id = ?)", v))
	}
	// Negated SolvedBy
	if v := filter.UnsolvedBy; v != nil {
		sb = append(sb, sq.Expr("NOT EXISTS (SELECT 1 FROM max_scores WHERE score = 100 AND problem_id = problems.id AND user_id = ?)", v))
	}
	if v := filter.AttemptedBy; v != nil {
		sb = append(sb, sq.Expr("EXISTS (SELECT 1 FROM max_scores WHERE score != 100 AND score >= 0 AND problem_id = problems.id AND user_id = ?)", v))
	}

	if filter.Unassociated {
		sb = append(sb, sq.Expr("NOT EXISTS (SELECT 1 FROM problem_list_problems WHERE problem_id = problems.id)"))
	}
	return sb
}

func problemUpdateQuery(upd *kilonova.ProblemUpdate, ub sq.UpdateBuilder) sq.UpdateBuilder {
	if v := upd.Name; v != nil {
		ub = ub.Set("name", v)
	}

	if v := upd.TestName; v != nil {
		ub = ub.Set("test_name", strings.TrimSpace(*v))
	}

	if v := upd.TimeLimit; v != nil && !math.IsNaN(*v) {
		ub = ub.Set("time_limit", v)
	}
	if v := upd.MemoryLimit; v != nil {
		ub = ub.Set("memory_limit", v)
	}

	if v := upd.DefaultPoints; v != nil {
		ub = ub.Set("default_points", v)
	}
	if v := upd.ScoreScale; v != nil {
		ub = ub.Set("leaderboard_score_scale", v)
	}

	if v := upd.SourceSize; v != nil {
		ub = ub.Set("source_size", v)
	}

	if v := upd.SourceCredits; v != nil {
		ub = ub.Set("source_credits", strings.TrimSpace(*v))
	}

	if v := upd.ConsoleInput; v != nil {
		ub = ub.Set("console_input", v)
	}
	if v := upd.Visible; v != nil {
		ub = ub.Set("visible", v)
		// if is set to visible
		if *v {
			// Published at - first time it was set visible
			ub = ub.Set("published_at", sq.Expr("COALESCE(published_at, NOW())"))
		}
	}
	if v := upd.VisibleTests; v != nil {
		ub = ub.Set("visible_tests", v)
	}
	if v := upd.ScoringStrategy; v != kilonova.ScoringTypeNone {
		ub = ub.Set("scoring_strategy", v)
	}
	if v := upd.ScorePrecision; v != nil {
		ub = ub.Set("digit_precision", v)
	}
	if v := upd.TaskType; v != kilonova.TaskTypeNone {
		ub = ub.Set("task_type", v)
	}
	if v := upd.CommunicationProcesses; v != nil {
		ub = ub.Set("communication_num_processes", v)
	}
	return ub
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

func (s *DB) ProblemEditors(ctx context.Context, pbid int) ([]*kilonova.UserFull, error) {
	return s.getAccessUsers(ctx, "problem_user_access", "problem_id", pbid, accessEditor)
}

func (s *DB) ProblemViewers(ctx context.Context, pbid int) ([]*kilonova.UserFull, error) {
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

		ScoreScale: pb.ScoreScale,

		ConsoleInput:   pb.ConsoleInput,
		ScorePrecision: pb.DigitPrecision,

		PublishedAt:     pb.PublishedAt,
		ScoringStrategy: pb.ScoringStrategy,

		TaskType: pb.TaskType,

		CommunicationProcesses: pb.CommunicationProcesses,
	}
}

func (s *DB) internalToScoredProblem(spb *dbScoredProblem, scoreUID int) (*kilonova.ScoredProblem, error) {
	pb := s.internalToProblem(&spb.dbProblem)
	var uid *int
	if scoreUID > 0 {
		uid = &scoreUID
	}
	return &kilonova.ScoredProblem{
		Problem: *pb,

		ScoreUserID: uid,
		MaxScore:    spb.MaxScore,
		IsEditor:    spb.IsEditor,
	}, nil
}

func (s *DB) internalToScoredProblems(ctx context.Context, spbs []*dbScoredProblem, scoreUID int) []*kilonova.ScoredProblem {
	if len(spbs) == 0 {
		return []*kilonova.ScoredProblem{}
	}
	rez := make([]*kilonova.ScoredProblem, len(spbs))
	for i := range rez {
		var err error
		rez[i], err = s.internalToScoredProblem(spbs[i], scoreUID)
		if err != nil {
			slog.WarnContext(ctx, "Could not convert to scored problem", slog.Any("err", err))
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
		return "name" + ord + ", id ASC"
	case "published_at":
		return "published_at" + ord + " NULLS LAST, created_at " + ord + ", id ASC"
	case "hot":
		return "(SELECT hot_cnt FROM hot_problems WHERE problem_id = id) " + ord + " NULLS LAST, published_at" + ord + " NULLS LAST, id ASC"
	default:
		return "id" + ord
	}
}
