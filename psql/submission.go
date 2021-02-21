package psql

import (
	"context"
	"fmt"
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
	err := s.db.GetContext(ctx, &sub, "SELECT * FROM submissions WHERE id = $1 LIMIT 1", id)
	return &sub, err
}

func (s *SubmissionService) Submissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	var subs []*kilonova.Submission
	where, args := s.filterQueryMaker(&filter)
	query := "SELECT * FROM submissions WHERE " + strings.Join(where, " AND ") + " ORDER BY id DESC " + FormatLimitOffset(filter.Limit, filter.Offset)
	if len(where) == 0 {
		strings.ReplaceAll(query, "WHERE", "")
	}
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

const createSubQuery = "INSERT INTO submissions (user_id, problem_id, language, code) VALUES ($1, $2, $3, $4) RETURNING id;"

func (s *SubmissionService) CreateSubmission(ctx context.Context, sub *kilonova.Submission) error {
	if sub.UserID == 0 || sub.ProblemID == 0 || sub.Language == "" || sub.Code == "" {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := s.db.GetContext(ctx, &id, createSubQuery, sub.UserID, sub.ProblemID, sub.Language, sub.Code)
	if err == nil {
		sub.ID = id
	}
	return err
}

func (s *SubmissionService) UpdateSubmission(ctx context.Context, id int, upd kilonova.SubmissionUpdate) error {
	// TODO: test it 100% works
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
	_, err := s.db.ExecContext(ctx, "DELETE FROM submissions WHERE id = $1", id)
	return err
}

func (s *SubmissionService) MaxScore(ctx context.Context, userid, problemid int) int {
	var score int

	err := s.db.GetContext(ctx, &score, `SELECT score FROM submissions
WHERE user_id = $1 AND problem_id = $2
ORDER BY score desc
LIMIT 1;`, userid, problemid)
	if err != nil {
		return -1
	}
	return score
}

func (s *SubmissionService) SolvedProblems(ctx context.Context, userid int) ([]int, error) {
	var pbs []int
	err := s.db.SelectContext(ctx, &pbs, `SELECT problem_id FROM submissions
WHERE score = 100 AND user_id = $1
GROUP BY problem_id
ORDER BY problem_id;`, userid)
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

	return toUpd, args
}
