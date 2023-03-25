package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/jmoiron/sqlx"
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
	err := s.conn.GetContext(ctx, &sub, "SELECT subs.* FROM submissions subs, submission_viewers users WHERE subs.id = $1 AND users.sub_id = subs.id AND users.user_id = $2 LIMIT 1", id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s.internalToSubmission(&sub), err
}

const subSelectQuery = "SELECT * FROM submissions WHERE %s %s %s;"

func (s *DB) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	where, args := subFilterQuery(&filter)
	query := fmt.Sprintf(subSelectQuery, strings.Join(where, " AND "), getSubmissionOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset))
	query = s.conn.Rebind(query)
	err := s.conn.SelectContext(ctx, &subs, query, args...)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.Submission{}, nil
	} else if err != nil {
		zap.S().Warn(err)
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}

func (s *DB) CountSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) (int, error) {
	where, args := subFilterQuery(&filter)
	query := "SELECT COUNT(id) FROM submissions WHERE " + strings.Join(where, " AND ")
	var cnt int
	query = s.conn.Rebind(query)
	err := s.conn.GetContext(ctx, &cnt, query, args...)
	return cnt, err
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

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, contest_id, language, code) VALUES (?, ?, ?, ?, ?) RETURNING id;"

func (s *DB) CreateSubmission(ctx context.Context, authorID int, problem *kilonova.Problem, language eval.Language, code string, contestID *int) (int, error) {
	if authorID <= 0 || problem == nil || language.InternalName == "" || code == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(createSubQuery), authorID, problem.ID, contestID, language.InternalName, code)
	return id, err
}

func (s *DB) UpdateSubmission(ctx context.Context, id int, upd kilonova.SubmissionUpdate) error {
	return updateSubmission(s.conn, ctx, id, upd)
}

func updateSubmission(tx sqlx.ExecerContext, ctx context.Context, id int, upd kilonova.SubmissionUpdate) error {
	toUpd, args := subUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := sqlx.Rebind(sqlx.BindType("pgx"), fmt.Sprintf(`UPDATE submissions SET %s WHERE id = ?;`, strings.Join(toUpd, ", ")))
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}
func (s *DB) BulkUpdateSubmissions(ctx context.Context, filter kilonova.SubmissionFilter, upd kilonova.SubmissionUpdate) error {
	toUpd, args := subUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	where, whereArgs := subFilterQuery(&filter)
	args = append(args, whereArgs...)
	query := s.conn.Rebind(fmt.Sprintf(`UPDATE submissions SET %s WHERE %s;`, strings.Join(toUpd, ", "), strings.Join(where, " AND ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) DeleteSubmission(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, "DELETE FROM submissions WHERE id = $1", id)
	return err
}

func (s *DB) MaxScore(ctx context.Context, userid, problemid int) int {
	var score int

	err := s.conn.GetContext(ctx, &score, "SELECT ms.score FROM max_score_view ms WHERE ms.user_id = $1 AND ms.problem_id = $2", userid, problemid)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			zap.S().Errorw("Couldn't get max score for ", zap.Int("userid", userid), zap.Int("problemid", problemid), zap.Error(err))
		}
		return -1
	}
	return score
}

func (s *DB) SolvedProblemIDs(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.conn.SelectContext(ctx, &pbs, `SELECT problem_id FROM max_score_view WHERE score = 100 AND user_id = $1 ORDER BY problem_id;`, userid)
	return pbs, err
}

func (s *DB) AttemptedProblemsIDs(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.conn.SelectContext(ctx, &pbs, `SELECT problem_id FROM max_score_view WHERE score != 100 AND score >= 0 AND user_id = $1 ORDER BY problem_id;`, userid)
	return pbs, err
}

func subFilterQuery(filter *kilonova.SubmissionFilter) ([]string, []any) {
	where, args := []string{"1 = 1"}, []any{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.UserID; v != nil {
		where, args = append(where, "user_id = ?"), append(args, v)
	}
	if v := filter.ProblemID; v != nil {
		where, args = append(where, "problem_id = ?"), append(args, v)
	}
	if v := filter.ContestID; v != nil {
		if *v == 0 { // Allow filtering for submissions from no contest
			where, args = append(where, "contest_id IS NULL"), append(args, v)
		} else {
			where, args = append(where, "contest_id = ?"), append(args, v)
		}
	}

	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		where, args = append(where, "EXISTS (SELECT 1 FROM submission_viewers WHERE user_id = ? AND sub_id = submissions.id)"), append(args, id)
	}

	if v := filter.Status; v != kilonova.StatusNone {
		where, args = append(where, "status = ?"), append(args, v)
	}
	if v := filter.Lang; v != nil {
		where, args = append(where, "lower(language) = lower(?)"), append(args, v)
	}
	if v := filter.Score; v != nil {
		where, args = append(where, "score = ?"), append(args, v)
	}

	if v := filter.CompileError; v != nil {
		where, args = append(where, "compile_error = ?"), append(args, v)
	}

	return where, args
}

func subUpdateQuery(upd *kilonova.SubmissionUpdate) ([]string, []any) {
	toUpd, args := []string{}, []any{}
	if v := upd.Status; v != kilonova.StatusNone {
		toUpd, args = append(toUpd, "status = ?"), append(args, v)
	}
	if v := upd.Score; v != nil {
		toUpd, args = append(toUpd, "score = ?"), append(args, v)
	}

	if v := upd.CompileError; v != nil {
		toUpd, args = append(toUpd, "compile_error = ?"), append(args, v)
	}
	if v := upd.CompileMessage; v != nil {
		toUpd, args = append(toUpd, "compile_message = ?"), append(args, v)
	}

	if v := upd.MaxTime; v != nil {
		toUpd, args = append(toUpd, "max_time = ?"), append(args, v)
	}
	if v := upd.MaxMemory; v != nil {
		toUpd, args = append(toUpd, "max_memory = ?"), append(args, v)
	}

	return toUpd, args
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
