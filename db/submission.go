package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.SubmissionService = &SubmissionService{}

type SubmissionService struct {
	db *sqlx.DB
}

func (s *SubmissionService) SubmissionByID(ctx context.Context, id int) (*kilonova.Submission, error) {
	var sub kilonova.Submission
	err := s.db.GetContext(ctx, &sub, s.db.Rebind("SELECT * FROM submissions WHERE id = ? LIMIT 1"), id)
	return &sub, err
}

func (s *SubmissionService) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	var subs []*kilonova.Submission
	where, args := s.filterQueryMaker(&filter)
	query := "SELECT * FROM submissions WHERE " + strings.Join(where, " AND ") + " ORDER BY id DESC " + FormatLimitOffset(filter.Limit, filter.Offset)
	query = s.db.Rebind(query)
	err := s.db.SelectContext(ctx, &subs, query, args...)
	return subs, err
}

func (s *SubmissionService) CountSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) (int, error) {
	where, args := s.filterQueryMaker(&filter)
	query := "SELECT COUNT(id) FROM submissions WHERE " + strings.Join(where, " AND ")
	if len(where) == 0 {
		strings.ReplaceAll(query, "WHERE", "")
	}
	var cnt int
	query = s.db.Rebind(query)
	err := s.db.GetContext(ctx, &cnt, query, args...)
	return cnt, err
}

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, language, code) VALUES (?, ?, ?, ?) RETURNING id;"

func (s *SubmissionService) CreateSubmission(ctx context.Context, sub *kilonova.Submission) error {
	if sub.UserID == 0 || sub.ProblemID == 0 || sub.Language == "" || sub.Code == "" {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := s.db.GetContext(ctx, &id, s.db.Rebind(createSubQuery), sub.UserID, sub.ProblemID, sub.Language, sub.Code)
	if err == nil {
		sub.ID = id
	}
	return err
}

func (s *SubmissionService) UpdateSubmission(ctx context.Context, id int, upd kilonova.SubmissionUpdate) error {
	toUpd, args := s.updateQueryMaker(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.db.Rebind(fmt.Sprintf(`UPDATE submissions SET %s WHERE id = ?;`, strings.Join(toUpd, ", ")))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SubmissionService) BulkUpdateSubmissions(ctx context.Context, filter kilonova.SubmissionFilter, upd kilonova.SubmissionUpdate) error {
	return s.bulkUpdateSubs(ctx, &filter, &upd)
}

func (s *SubmissionService) DeleteSubmission(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind("DELETE FROM submissions WHERE id = ?"), id)
	return err
}

func (s *SubmissionService) MaxScore(ctx context.Context, userid, problemid int) int {
	var score int

	err := s.db.GetContext(ctx, &score, s.db.Rebind(`SELECT score FROM submissions
WHERE user_id = ? AND problem_id = ?
ORDER BY score DESC 
LIMIT 1;`), userid, problemid)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Println(userid, problemid, err)
		}
		return -1
	}
	return score
}

func (s *SubmissionService) MaxScores(ctx context.Context, userid int, pbids []int) map[int]int {
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
	err := s.db.SelectContext(ctx, &cols, s.db.Rebind("SELECT problem_id, MAX(score) AS score FROM submissions WHERE problem_id IN "+inClause+" AND user_id = ? GROUP BY problem_id"), args...)
	if err != nil {
		log.Println("MaxScores:", err)
		return nil
	}

	for _, col := range cols {
		rez[col.ProblemID] = col.MaxScore
	}
	return rez
}

func (s *SubmissionService) SolvedProblems(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.db.SelectContext(ctx, &pbs, s.db.Rebind(`SELECT problem_id FROM submissions
WHERE score = 100 AND user_id = ?
GROUP BY problem_id
ORDER BY problem_id;`), userid)
	return pbs, err
}

func (s *SubmissionService) bulkUpdateSubs(ctx context.Context, filter *kilonova.SubmissionFilter, upd *kilonova.SubmissionUpdate) error {
	toUpd, args := s.updateQueryMaker(upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	where, whereArgs := s.filterQueryMaker(filter)
	args = append(args, whereArgs...)
	query := s.db.Rebind(fmt.Sprintf(`UPDATE submissions SET %s WHERE %s;`, strings.Join(toUpd, ", "), strings.Join(where, " AND ")))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SubmissionService) filterQueryMaker(filter *kilonova.SubmissionFilter) ([]string, []interface{}) {
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
	if v := filter.Visible; v != nil {
		where, args = append(where, "visible = ?"), append(args, v)
	}
	if v := filter.Quality; v != nil {
		where, args = append(where, "quality = ?"), append(args, v)
	}

	return where, args
}

func (s *SubmissionService) updateQueryMaker(upd *kilonova.SubmissionUpdate) ([]string, []interface{}) {
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

	if v := upd.Visible; v != nil {
		toUpd, args = append(toUpd, "visible = ?"), append(args, v)
	}
	if v := upd.Quality; v != nil {
		toUpd, args = append(toUpd, "quality = ?"), append(args, v)
	}

	return toUpd, args
}

func NewSubmissionService(db *sqlx.DB) kilonova.SubmissionService {
	return &SubmissionService{db}
}
