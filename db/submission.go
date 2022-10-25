package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func (s *DB) Submission(ctx context.Context, id int) (*kilonova.Submission, error) {
	var sub kilonova.Submission
	err := s.conn.GetContext(ctx, &sub, s.conn.Rebind("SELECT * FROM submissions WHERE id = ? LIMIT 1"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &sub, err
}

const subSelectQuery = "SELECT * FROM submissions WHERE %s %s %s;"

func (s *DB) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	var subs []*kilonova.Submission
	where, args := subFilterQuery(&filter)
	query := fmt.Sprintf(subSelectQuery, strings.Join(where, " AND "), getSubmissionOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset))
	query = s.conn.Rebind(query)
	err := s.conn.SelectContext(ctx, &subs, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Submission{}, nil
	}
	return subs, err
}

func (s *DB) CountSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) (int, error) {
	where, args := subFilterQuery(&filter)
	query := "SELECT COUNT(id) FROM submissions WHERE " + strings.Join(where, " AND ")
	var cnt int
	query = s.conn.Rebind(query)
	err := s.conn.GetContext(ctx, &cnt, query, args...)
	return cnt, err
}

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, language, code) VALUES (?, ?, ?, ?) RETURNING id;"

func (s *DB) CreateSubmission(ctx context.Context, authorID int, problem *kilonova.Problem, language eval.Language, code string) (int, error) {
	if authorID <= 0 || problem == nil || language.InternalName == "" || code == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(createSubQuery), authorID, problem.ID, language.InternalName, code)
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
	if len(pbids) == 0 {
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

	args := make([]any, 0, len(pbids))
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

func (s *DB) solvedProblems(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.conn.SelectContext(ctx, &pbs, `SELECT problem_id FROM submissions WHERE score = 100 AND user_id = $1 GROUP BY problem_id ORDER BY problem_id;`, userid)
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
	default:
		return "ORDER BY id" + ord
	}
}

// FetchGraderSubmission returns a waiting submission that is not locked. Note that it must be closed
func (s *DB) FetchGraderSubmission(ctx context.Context) (eval.GraderSubmission, error) {
	tx, err := s.conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	var sub kilonova.Submission
	err = tx.GetContext(ctx, &sub, "SELECT * FROM submissions WHERE status = 'waiting' ORDER BY id DESC LIMIT 1 FOR UPDATE SKIP LOCKED")
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Rollback()
	}
	return &graderSub{ctx, &sub, tx}, nil
}

type graderSub struct {
	ctx context.Context
	sub *kilonova.Submission
	tx  *sqlx.Tx
}

func (g *graderSub) Submission() *kilonova.Submission {
	return g.sub
}

func (g *graderSub) Update(sub kilonova.SubmissionUpdate) error {
	return updateSubmission(g.tx, g.ctx, g.sub.ID, sub)
}

func (g *graderSub) Close() error {
	defer g.tx.Rollback()
	return g.tx.Commit()
}
