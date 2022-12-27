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
	err := s.conn.GetContext(ctx, &sub, s.conn.Rebind("SELECT * FROM submissions WHERE id = ? LIMIT 1"), id)
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
	if errors.Is(err, sql.ErrNoRows) {
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

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, contest_id, language, code) VALUES (?, ?, ?, ?, ?) RETURNING id;"

func (s *DB) CreateSubmission(ctx context.Context, authorID int, problem *kilonova.Problem, language eval.Language, code string, contest *kilonova.Contest) (int, error) {
	if authorID <= 0 || problem == nil || language.InternalName == "" || code == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var contestID *int
	if contest != nil {
		contestID = &contest.ID
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
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM submissions WHERE id = ?"), id)
	return err
}

func (s *DB) MaxScore(ctx context.Context, userid, problemid int) int {
	var score int

	err := s.conn.GetContext(ctx, &score, s.conn.Rebind(`SELECT score FROM submissions
WHERE user_id = ? AND problem_id = ?
ORDER BY score DESC 
LIMIT 1;`), userid, problemid)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			zap.S().Errorw("Couldn't get max score for ", zap.Int("userid", userid), zap.Int("problemid", problemid), zap.Error(err))
		}
		return -1
	}
	return score
}

func (s *DB) MaxScores(ctx context.Context, userid int, pbids []int) map[int]int {
	if pbids == nil || len(pbids) == 0 {
		return nil
	}

	cols := []struct {
		ProblemID int `db:"problem_id"`
		MaxScore  int `db:"score"`
	}{}

	inClause := "(?"
	for i := 1; i < len(pbids); i++ {
		inClause += ", ?"
	}
	inClause += ")"

	args := make([]interface{}, 0, len(pbids))
	for _, id := range pbids {
		args = append(args, id)
	}
	args = append(args, userid)

	rez := make(map[int]int)
	err := s.conn.SelectContext(ctx, &cols, s.conn.Rebind("SELECT problem_id, MAX(score) AS score FROM submissions WHERE problem_id IN "+inClause+" AND user_id = ? GROUP BY problem_id"), args...)
	if err != nil {
		zap.S().Error(zap.Error(err))
		return nil
	}

	for _, col := range cols {
		rez[col.ProblemID] = col.MaxScore
	}
	return rez
}

func (s *DB) SolvedProblemIDs(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.conn.SelectContext(ctx, &pbs, `SELECT problem_id FROM max_score_view WHERE score = 100 AND user_id = $1 ORDER BY problem_id;`, userid)
	return pbs, err
}

func (s *DB) AttemptedProblemsIDs(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.conn.SelectContext(ctx, &pbs, `SELECT problem_id FROM max_score_view WHERE score != 100 AND user_id = $1 ORDER BY problem_id;`, userid)
	return pbs, err
}

func subFilterQuery(filter *kilonova.SubmissionFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
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
			where, args = append(where, "contest_id = NULL"), append(args, v)
		} else {
			where, args = append(where, "contest_id = ?"), append(args, v)
		}
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

func subUpdateQuery(upd *kilonova.SubmissionUpdate) ([]string, []interface{}) {
	toUpd, args := []string{}, []interface{}{}
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
