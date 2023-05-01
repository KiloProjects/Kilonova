package db

import (
	"context"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) ProblemStatisticsNumSolved(ctx context.Context, problemID int) (int, error) {
	var cnt int
	err := s.pgconn.QueryRow(ctx, "SELECT COUNT(*) FROM max_score_view WHERE problem_id = $1 AND score = 100", problemID).Scan(&cnt)
	if err != nil {
		return -1, err
	}
	return cnt, nil
}

func (s *DB) ProblemStatisticsNumAttempted(ctx context.Context, problemID int) (int, error) {
	var cnt int
	err := s.pgconn.QueryRow(ctx, "SELECT COUNT(*) FROM max_score_view WHERE problem_id = $1 AND score >= 0", problemID).Scan(&cnt)
	if err != nil {
		return -1, err
	}
	return cnt, nil
}

func (s *DB) ProblemStatisticsSize(ctx context.Context, problemID int) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	err := s.conn.SelectContext(ctx, &subs, `
	WITH top_by_size AS (
		SELECT DISTINCT ON (problem_id, user_id) id, problem_id, user_id, code_size FROM submissions WHERE score = 100 AND problem_id = $1 ORDER BY problem_id, user_id, code_size ASC
	) SELECT submissions.* FROM submissions, top_by_size WHERE submissions.id = top_by_size.id ORDER BY submissions.code_size ASC LIMIT 5;
`, problemID)
	if err != nil {
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}

func (s *DB) ProblemStatisticsMemory(ctx context.Context, problemID int) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	err := s.conn.SelectContext(ctx, &subs, `
	WITH top_by_memory AS (
		SELECT DISTINCT ON (problem_id, user_id) id, problem_id, user_id, max_time FROM submissions WHERE score = 100 AND problem_id = $1 ORDER BY problem_id, user_id, max_memory ASC
	) SELECT submissions.* FROM submissions, top_by_memory WHERE submissions.id = top_by_memory.id ORDER BY submissions.max_memory LIMIT 5;
`, problemID)
	if err != nil {
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}

func (s *DB) ProblemStatisticsTime(ctx context.Context, problemID int) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	err := s.conn.SelectContext(ctx, &subs, `
	WITH top_by_time AS (
		SELECT DISTINCT ON (problem_id, user_id) id, problem_id, user_id, max_time FROM submissions WHERE score = 100 AND problem_id = $1 ORDER BY problem_id, user_id, max_time ASC
	) SELECT submissions.* FROM submissions, top_by_time WHERE submissions.id = top_by_time.id ORDER BY submissions.max_time LIMIT 5;
`, problemID)
	if err != nil {
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}
