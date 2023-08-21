package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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

	Score decimal.Decimal `db:"score"`

	ScorePrecision int32 `db:"digit_precision"`
}

func (s *DB) Submission(ctx context.Context, id int) (*kilonova.Submission, error) {
	var sub dbSubmission
	err := Get(s.conn, ctx, &sub, "SELECT * FROM submissions WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

func (s *DB) SubmissionLookingUser(ctx context.Context, id int, userID int) (*kilonova.Submission, error) {
	var sub dbSubmission
	err := Get(s.conn, ctx, &sub, "SELECT subs.* FROM submissions subs, visible_submissions($2) users WHERE subs.id = $1 AND users.sub_id = subs.id LIMIT 1", id, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

func (s *DB) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)

	query := fmt.Sprintf("SELECT * FROM submissions WHERE %s %s %s", fb.Where(), getSubmissionOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset))
	err := Select(s.conn, ctx, &subs, query, fb.Args()...)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.Submission{}, nil
	} else if err != nil {
		zap.S().Warn(err)
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}

func (s *DB) SubmissionCount(ctx context.Context, filter kilonova.SubmissionFilter) (int, error) {
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)
	var val int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM submissions WHERE "+fb.Where(), fb.Args()...).Scan(&val)
	if err != nil {
		return -1, err
	}
	return val, nil
}

func (s *DB) RemainingSubmissionCount(ctx context.Context, contest *kilonova.Contest, problemID, userID int) (int, error) {
	var cnt int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM submissions WHERE contest_id = $1 AND problem_id = $2 AND user_id = $3", contest.ID, problemID, userID).Scan(&cnt)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return -1, err
		}
		cnt = 0
	}
	return contest.MaxSubs - cnt, nil
}

func (s *DB) WaitingSubmissionCount(ctx context.Context, userID int) (int, error) {
	var cnt int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM submissions WHERE status <> 'finished' AND user_id = $1", userID).Scan(&cnt)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return -1, err
		}
		cnt = 0
	}
	return cnt, nil
}

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, contest_id, language, code) VALUES ($1, $2, $3, $4, $5) RETURNING id;"

func (s *DB) CreateSubmission(ctx context.Context, authorID int, problem *kilonova.Problem, language eval.Language, code string, contestID *int) (int, error) {
	if authorID <= 0 || problem == nil || language.InternalName == "" || code == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	err := s.conn.QueryRow(ctx, createSubQuery, authorID, problem.ID, contestID, language.InternalName, code).Scan(&id)
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

	err := s.conn.QueryRow(ctx, "SELECT ms.score FROM max_score_view ms WHERE ms.user_id = $1 AND ms.problem_id = $2", userid, problemid).Scan(&score)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			zap.S().Errorw("Couldn't get max score for ", zap.Int("userid", userid), zap.Int("problemid", problemid), zap.Error(err))
		}
		return decimal.NewFromInt(-1)
	}
	return score
}

func (s *DB) ContestMaxScore(ctx context.Context, userid, problemid, contestid int) decimal.Decimal {
	var score decimal.Decimal

	err := s.conn.QueryRow(ctx, "SELECT ms.score FROM max_score_contest_view ms WHERE ms.user_id = $1 AND ms.problem_id = $2 AND ms.contest_id = $3", userid, problemid, contestid).Scan(&score)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			zap.S().Errorw("Couldn't get contest max score for ", zap.Int("userid", userid), zap.Int("problemid", problemid), zap.Int("contestid", contestid), zap.Error(err))
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
		fb.AddConstraint("id = -1")
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
	if v := filter.ContestID; v != nil {
		if *v == 0 { // Allow filtering for submissions from no contest
			fb.AddConstraint("contest_id IS NULL")
		} else {
			fb.AddConstraint("contest_id = %s", v)
		}
	}

	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		fb.AddConstraint("EXISTS (SELECT 1 FROM visible_submissions(%s) WHERE sub_id = submissions.id)", id)
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
	if v := filter.Lang; v != nil {
		fb.AddConstraint("lower(language) = lower(%s)", v)
	}
	if v := filter.Score; v != nil {
		fb.AddConstraint("score = %s", v)
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

	if v := upd.CompileError; v != nil {
		b.AddUpdate("compile_error = %s", v)
	}
	if v := upd.CompileMessage; v != nil {
		b.AddUpdate("compile_message = %s", v)
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
		ID:             sub.ID,
		CreatedAt:      sub.CreatedAt,
		UserID:         sub.UserID,
		ProblemID:      sub.ProblemID,
		Language:       sub.Language,
		Code:           sub.Code,
		CodeSize:       sub.CodeSize,
		Status:         kilonova.Status(sub.Status),
		CompileError:   sub.CompileError,
		CompileMessage: sub.CompileMessage,
		MaxTime:        sub.MaxTime,
		MaxMemory:      sub.MaxMemory,
		ContestID:      sub.ContestID,
		Score:          sub.Score,
		ScorePrecision: sub.ScorePrecision,
	}
}
