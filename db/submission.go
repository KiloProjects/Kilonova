package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type dbSubmission struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UserID    int       `db:"user_id"`
	ProblemID int       `db:"problem_id"`
	Language  string    `db:"language"`
	Code      string    `db:"code"`
	CodeSize  int       `db:"code_size"`
	Status    string    `db:"status"`

	CompileError   *bool   `db:"compile_error"`
	CompileMessage *string `db:"compile_message"`

	MaxTime   float64 `db:"max_time"`
	MaxMemory int     `db:"max_memory"`

	ContestID *int `db:"contest_id"`

	Score          decimal.Decimal `db:"score"`
	ScorePrecision int32           `db:"digit_precision"`

	CompileDuration *float64 `db:"compile_duration"`

	ScoreScale decimal.Decimal `db:"leaderboard_score_scale"`

	SubmissionType kilonova.EvalType `db:"submission_type"`
	ICPCVerdict    *string           `db:"icpc_verdict"`
}

func (s *DB) Submission(ctx context.Context, id int) (*kilonova.Submission, error) {
	var sub dbSubmission
	err := Get(s.conn, ctx, &sub, "SELECT * FROM submissions WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

func (s *DB) SubmissionLookingUser(ctx context.Context, id int, user *kilonova.UserBrief) (*kilonova.Submission, error) {
	if user.IsAdmin() {
		return s.Submission(ctx, id)
	}
	var userID int = 0
	if user != nil {
		userID = user.ID
	}

	var sub dbSubmission
	err := Get(s.conn, ctx, &sub, "SELECT subs.* FROM submissions subs, visible_submissions($2) v_subs WHERE subs.id = $1 AND v_subs.sub_id = $1 LIMIT 1", id, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

func (s *DB) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)

	query := fmt.Sprintf("SELECT submissions.* FROM submissions WHERE %s %s %s", fb.Where(), getSubmissionOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset))
	err := Select(s.conn, ctx, &subs, query, fb.Args()...)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.Submission{}, nil
	} else if err != nil {
		slog.WarnContext(ctx, "Could not load submisions", slog.Any("err", err))
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}

func (s *DB) SubmissionCode(ctx context.Context, subID int) ([]byte, error) {
	var code []byte
	err := s.conn.QueryRow(ctx, "SELECT code FROM submissions WHERE id = $1 LIMIT 1", subID).Scan(&code)
	return code, err
}

func (s *DB) SubmissionCount(ctx context.Context, filter kilonova.SubmissionFilter, limit int) (int, error) {
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)
	query := "SELECT COUNT(*) FROM submissions WHERE " + fb.Where()
	if limit > 0 {
		lim := fb.FormatString("LIMIT %s", limit)
		query = fmt.Sprintf("SELECT COUNT(*) FROM (SELECT 1 FROM submissions WHERE %s %s) sbc", fb.Where(), lim)
	}
	var val int
	err := s.conn.QueryRow(ctx, query, fb.Args()...).Scan(&val)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return -1, err
		}
		val = 0
	}
	return val, nil
}

func (s *DB) LastSubmissionTime(ctx context.Context, filter kilonova.SubmissionFilter) (*time.Time, error) {
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)
	query := fmt.Sprintf("SELECT MAX(created_at) FROM submissions WHERE %s", fb.Where())
	var val *time.Time
	err := s.conn.QueryRow(ctx, query, fb.Args()...).Scan(&val)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		val = nil
	}
	return val, nil
}

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, contest_id, language, code) VALUES ($1, $2, $3, $4, $5) RETURNING id;"

func (s *DB) CreateSubmission(ctx context.Context, authorID int, problem *kilonova.Problem, langName string, code string, contestID *int) (int, error) {
	if authorID <= 0 || problem == nil || langName == "" || code == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	err := s.conn.QueryRow(ctx, createSubQuery, authorID, problem.ID, contestID, langName, code).Scan(&id)
	return id, err
}

func (s *DB) UpdateSubmission(ctx context.Context, id int, upd kilonova.SubmissionUpdate) error {
	return s.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{ID: &id}, upd)
}

func (s *DB) BulkUpdateSubmissions(ctx context.Context, filter kilonova.SubmissionFilter, upd kilonova.SubmissionUpdate) error {
	ub := newUpdateBuilder()
	subUpdateQuery(&upd, ub)
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	subFilterQuery(&filter, fb)
	_, err := s.conn.Exec(ctx, `UPDATE submissions SET `+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) ClearUserContestSubmissions(ctx context.Context, contestID, userID int) error {
	_, err := s.conn.Exec(ctx, "UPDATE submissions SET contest_id = NULL WHERE contest_id = $1 AND user_id = $2", contestID, userID)
	return err
}

func (s *DB) DeleteSubmission(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM submissions WHERE id = $1", id)
	return err
}

func (s *DB) MaxScoreSubID(ctx context.Context, userid, problemid int) (int, error) {
	var score int

	err := s.conn.QueryRow(ctx, "SELECT id FROM submissions WHERE user_id = $1 AND problem_id = $2 ORDER BY score DESC, id ASC LIMIT 1", userid, problemid).Scan(&score)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return -1, nil
		}
		return -1, err
	}
	return score, nil
}

func (s *DB) MaxScore(ctx context.Context, userid, problemid int) decimal.Decimal {
	var score decimal.Decimal

	err := s.conn.QueryRow(ctx, "SELECT ms.score FROM max_scores ms WHERE ms.user_id = $1 AND ms.problem_id = $2", userid, problemid).Scan(&score)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.ErrorContext(ctx, "Couldn't get max score", slog.Any("err", err), slog.Int("user_id", userid), slog.Int("problem_id", problemid))
		}
		return decimal.NewFromInt(-1)
	}
	return score
}

func (s *DB) ContestMaxScore(ctx context.Context, userid, problemid, contestid int, freezeTime *time.Time) decimal.Decimal {
	var score decimal.Decimal

	err := s.conn.QueryRow(ctx, "SELECT ms.score FROM contest_max_scores($3, $4) ms WHERE ms.user_id = $1 AND ms.problem_id = $2", userid, problemid, contestid, freezeTime).Scan(&score)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.ErrorContext(ctx, "Couldn't get contest max score", slog.Any("err", err), slog.Int("user_id", userid), slog.Int("problem_id", problemid), slog.Int("contest_id", contestid))
		}
		return decimal.NewFromInt(-1)
	}
	return score
}

func subFilterQuery(filter *kilonova.SubmissionFilter, fb *filterBuilder) {
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		fb.AddConstraint("0 = 1")
	}
	if v := filter.IDs; len(v) > 0 {
		fb.AddConstraint("id = ANY(%s)", v)
	}
	if v := filter.UserID; v != nil {
		fb.AddConstraint("user_id = %s", v)
	}
	if v := filter.ProblemID; v != nil {
		fb.AddConstraint("problem_id = %s", v)
	}
	if v := filter.ProblemListID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM problem_list_deep_problems WHERE list_id = %s AND problem_id = submissions.problem_id)", v)
	}
	if v := filter.ContestID; v != nil {
		if *v == 0 { // Allow filtering for submissions from no contest
			fb.AddConstraint("contest_id IS NULL")
		} else {
			fb.AddConstraint("contest_id = %s", v)
		}
	}

	// Should keep in mind to sync with lateralVisibleSubs
	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		if !filter.LookingUser.IsAdmin() {
			fb.AddConstraint("EXISTS (SELECT 1 FROM visible_submissions_ex(%s, %s, %s) WHERE sub_id = submissions.id)", id, filter.ProblemID, filter.UserID)
		}

	}

	if filter.FromAuthors {
		fb.AddConstraint(`
		EXISTS (
			SELECT 1 FROM problem_user_access 
				WHERE access = 'editor' 
					AND problem_id = submissions.problem_id 
					AND user_id = submissions.user_id
		)`)
	}

	if v := filter.Status; v != kilonova.StatusNone {
		fb.AddConstraint("status = %s", v)
	}
	if filter.Waiting {
		fb.AddConstraint("(status = %s OR status = %s OR status = %s)",
			kilonova.StatusCreating,
			kilonova.StatusWaiting,
			kilonova.StatusWorking,
		)
	}
	if v := filter.Lang; v != nil {
		fb.AddConstraint("lower(language) = lower(%s)", v)
	}
	if v := filter.Score; v != nil {
		fb.AddConstraint("score = %s", v)
	}

	if v := filter.Since; v != nil {
		fb.AddConstraint("created_at > %s", v)
	}

	if v := filter.CompileError; v != nil {
		fb.AddConstraint("compile_error = %s", v)
	}
}

func subUpdateQuery(upd *kilonova.SubmissionUpdate, b *updateBuilder) {
	if v := upd.Status; v != kilonova.StatusNone {
		b.AddUpdate("status = %s", v)
	}
	if v := upd.Score; v != nil {
		b.AddUpdate("score = %s", v)
	}

	if v := upd.ICPCVerdict; upd.ChangeVerdict {
		b.AddUpdate("icpc_verdict = %s", v)
	}

	if v := upd.CompileError; v != nil {
		b.AddUpdate("compile_error = %s", v)
	}
	if v := upd.CompileMessage; v != nil {
		b.AddUpdate("compile_message = %s", v)
	}
	if v := upd.CompileTime; v != nil {
		b.AddUpdate("compile_duration = %s", v)
	}

	if v := upd.MaxTime; v != nil {
		b.AddUpdate("max_time = %s", v)
	}
	if v := upd.MaxMemory; v != nil {
		b.AddUpdate("max_memory = %s", v)
	}
}

func getSubmissionOrdering(ordering string, ascending bool) string {
	ord := " DESC"
	if ascending {
		ord = " ASC"
	}
	switch ordering {
	case "max_time":
		return "ORDER BY max_time" + ord + ", id DESC"
	case "max_mem":
		return "ORDER BY max_memory" + ord + ", id DESC"
	case "score":
		return "ORDER BY score" + ord + ", id DESC"
	case "code_size":
		return "ORDER BY code_size" + ord + ", id DESC"
	default:
		return "ORDER BY id" + ord
	}
}

func (s *DB) internalToSubmission(sub *dbSubmission) *kilonova.Submission {
	if sub == nil {
		return nil
	}

	return &kilonova.Submission{
		ID:        sub.ID,
		CreatedAt: sub.CreatedAt,
		UserID:    sub.UserID,
		ProblemID: sub.ProblemID,
		Language:  sub.Language,
		// Code:           sub.Code,
		CodeSize:       sub.CodeSize,
		Status:         kilonova.Status(sub.Status),
		CompileError:   sub.CompileError,
		CompileMessage: sub.CompileMessage,
		MaxTime:        sub.MaxTime,
		MaxMemory:      sub.MaxMemory,
		ContestID:      sub.ContestID,
		Score:          sub.Score,
		ScorePrecision: sub.ScorePrecision,
		ScoreScale:     sub.ScoreScale,

		CompileTime: sub.CompileDuration,

		SubmissionType: sub.SubmissionType,
		ICPCVerdict:    sub.ICPCVerdict,
	}
}
