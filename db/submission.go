package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/jackc/pgx/v5"
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

	CompileError   sql.NullBool   `db:"compile_error"`
	CompileMessage sql.NullString `db:"compile_message"`

	MaxTime   float64 `db:"max_time"`
	MaxMemory int     `db:"max_memory"`

	ContestID *int `db:"contest_id"`

	Score int `db:"score"`

	// Optimization so Submissions() also returns the total count
	FullCount *int `db:"full_count"`
}

func (s *DB) Submission(ctx context.Context, id int) (*kilonova.Submission, error) {
	var sub dbSubmission
	err := s.conn.GetContext(ctx, &sub, "SELECT * FROM submissions WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

func (s *DB) SubmissionLookingUser(ctx context.Context, id int, userID int) (*kilonova.Submission, error) {
	var sub dbSubmission
	err := s.conn.GetContext(ctx, &sub, "SELECT subs.* FROM submissions subs, visible_submissions($2) users WHERE subs.id = $1 AND users.sub_id = subs.id LIMIT 1", id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

const subSelectQuery = "SELECT *, COUNT(*) OVER() AS full_count FROM submissions WHERE %s %s %s;"

func (s *DB) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, int, error) {
	var subs []*dbSubmission
	where, args := subFilterQuery(&filter, []any{})
	query := fmt.Sprintf(subSelectQuery, where, getSubmissionOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.conn.SelectContext(ctx, &subs, query, args...)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.Submission{}, 0, nil
	} else if err != nil {
		zap.S().Warn(err)
		return []*kilonova.Submission{}, 0, err
	}
	cnt := -1
	if len(subs) > 0 {
		cnt = *subs[0].FullCount
	}
	return mapper(subs, s.internalToSubmission), cnt, nil
}

func (s *DB) RemainingSubmissionCount(ctx context.Context, contest *kilonova.Contest, problemID, userID int) (int, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(*) FROM submissions WHERE contest_id = $1 AND problem_id = $2 AND user_id = $3", contest.ID, problemID, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return -1, err
		}
		cnt = 0
	}
	return contest.MaxSubs - cnt, nil
}

func (s *DB) WaitingSubmissionCount(ctx context.Context, userID int) (int, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(*) FROM submissions WHERE status <> 'finished' AND user_id = $1", userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
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
	err := s.conn.GetContext(ctx, &id, createSubQuery, authorID, problem.ID, contestID, language.InternalName, code)
	return id, err
}

func (s *DB) UpdateSubmission(ctx context.Context, id int, upd kilonova.SubmissionUpdate) error {
	toUpd, args, err := subUpdateQuery(&upd)
	if err != nil {
		return err
	}
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE submissions SET %s WHERE id = $%d;`, toUpd, len(args))
	_, err = s.pgconn.Exec(ctx, query, args...)
	return err
}

func (s *DB) BulkUpdateSubmissions(ctx context.Context, filter kilonova.SubmissionFilter, upd kilonova.SubmissionUpdate) error {
	toUpd, args, err := subUpdateQuery(&upd)
	if err != nil {
		return err
	}
	where, whereArgs := subFilterQuery(&filter, args)
	_, err = s.conn.ExecContext(ctx, fmt.Sprintf(`UPDATE submissions SET %s WHERE %s;`, toUpd, where), whereArgs...)
	return err
}

func (s *DB) DeleteSubmission(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, "DELETE FROM submissions WHERE id = $1", id)
	return err
}

func (s *DB) MaxScoreSubID(ctx context.Context, userid, problemid int) (int, error) {
	var score int

	err := s.pgconn.QueryRow(ctx, "SELECT id FROM submissions WHERE user_id = $1 AND problem_id = $2 ORDER BY score DESC, id ASC LIMIT 1", userid, problemid).Scan(&score)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return -1, nil
		}
		return -1, err
	}
	return score, nil
}

func (s *DB) MaxScore(ctx context.Context, userid, problemid int) int {
	var score int

	err := s.pgconn.QueryRow(ctx, "SELECT ms.score FROM max_score_view ms WHERE ms.user_id = $1 AND ms.problem_id = $2", userid, problemid).Scan(&score)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			zap.S().Errorw("Couldn't get max score for ", zap.Int("userid", userid), zap.Int("problemid", problemid), zap.Error(err))
		}
		return -1
	}
	return score
}

func (s *DB) ContestMaxScore(ctx context.Context, userid, problemid, contestid int) int {
	var score int

	err := s.pgconn.QueryRow(ctx, "SELECT ms.score FROM max_score_contest_view ms WHERE ms.user_id = $1 AND ms.problem_id = $2 AND ms.contest_id = $3", userid, problemid, contestid).Scan(&score)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			zap.S().Errorw("Couldn't get contest max score for ", zap.Int("userid", userid), zap.Int("problemid", problemid), zap.Int("contestid", contestid), zap.Error(err))
		}
		return -1
	}
	return score
}

func (s *DB) SolvedProblemIDs(ctx context.Context, userid int) ([]int, error) {
	rows, err := s.pgconn.Query(ctx, `SELECT problem_id FROM max_score_view WHERE score = 100 AND user_id = $1 ORDER BY problem_id;`, userid)
	if err != nil {
		return []int{}, err
	}
	pbs, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return []int{}, err
	}
	return pbs, err
}

func (s *DB) AttemptedProblemsIDs(ctx context.Context, userid int) ([]int, error) {
	rows, err := s.pgconn.Query(ctx, `SELECT problem_id FROM max_score_view WHERE score != 100 AND score >= 0 AND user_id = $1 ORDER BY problem_id;`, userid)
	if err != nil {
		return []int{}, err
	}
	pbs, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return []int{}, err
	}
	return pbs, err
}

func subFilterQuery(filter *kilonova.SubmissionFilter, initialArgs []any) (string, []any) {
	qb := newFilterBuilderFromPos(initialArgs)
	if v := filter.ID; v != nil {
		qb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		qb.AddConstraint("id = -1")
	}
	if v := filter.IDs; len(v) > 0 {
		qb.AddConstraint("id = ANY(%s)", v)
	}
	if v := filter.UserID; v != nil {
		qb.AddConstraint("user_id = %s", v)
	}
	if v := filter.ProblemID; v != nil {
		qb.AddConstraint("problem_id = %s", v)
	}
	if v := filter.ContestID; v != nil {
		if *v == 0 { // Allow filtering for submissions from no contest
			qb.AddConstraint("contest_id IS NULL")
		} else {
			qb.AddConstraint("contest_id = %s", v)
		}
	}

	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		qb.AddConstraint("EXISTS (SELECT 1 FROM visible_submissions(%s) WHERE sub_id = submissions.id)", id)
	}

	if v := filter.Status; v != kilonova.StatusNone {
		qb.AddConstraint("status = %s", v)
	}
	if v := filter.Lang; v != nil {
		qb.AddConstraint("lower(language) = lower(%s)", v)
	}
	if v := filter.Score; v != nil {
		qb.AddConstraint("score = %s", v)
	}

	if v := filter.CompileError; v != nil {
		qb.AddConstraint("compile_error = %s", v)
	}

	return qb.Where(), qb.Args()
}

func subUpdateQuery(upd *kilonova.SubmissionUpdate) (string, []any, error) {
	b := newUpdateBuilder()
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

	return b.ToUpdate(), b.Args(), b.CheckUpdates()
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
	var cErr *bool = nil
	if sub.CompileError.Valid {
		cErr = &sub.CompileError.Bool
	}
	var cMsg *string = nil
	if sub.CompileMessage.Valid {
		cMsg = &sub.CompileMessage.String
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
		CompileError:   cErr,
		CompileMessage: cMsg,
		MaxTime:        sub.MaxTime,
		MaxMemory:      sub.MaxMemory,
		ContestID:      sub.ContestID,
		Score:          sub.Score,
	}
}
