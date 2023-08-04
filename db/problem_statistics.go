package db

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

type ProblemStats struct {
	ProblemID      int `db:"problem_id"`
	NumSolvedBy    int `db:"num_solved"`
	NumAttemptedBy int `db:"num_attempted"`
}

func (s *DB) ProblemsStatistics(ctx context.Context, problemIDs []int) (map[int]*ProblemStats, error) {
	rows, _ := s.conn.Query(ctx, "SELECT problem_id, num_attempted, num_solved FROM problem_statistics WHERE problem_id = ANY($1)", problemIDs)
	stats, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[ProblemStats])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	mp := make(map[int]*ProblemStats)
	for _, id := range problemIDs {
		mp[id] = &ProblemStats{ProblemID: id, NumSolvedBy: 0, NumAttemptedBy: 0}
	}
	for _, stat := range stats {
		stat := stat
		mp[stat.ProblemID] = stat
	}
	return mp, nil
}

func (s *DB) ProblemStatisticsSize(ctx context.Context, problemID int) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	err := Select(s.conn, ctx, &subs, `
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
	err := Select(s.conn, ctx, &subs, `
	WITH top_by_memory AS (
		SELECT DISTINCT ON (problem_id, user_id) id, problem_id, user_id, max_time FROM submissions WHERE score = 100 AND problem_id = $1 AND max_memory >= 0 ORDER BY problem_id, user_id, max_memory ASC
	) SELECT submissions.* FROM submissions, top_by_memory WHERE submissions.id = top_by_memory.id ORDER BY submissions.max_memory LIMIT 5;
`, problemID)
	if err != nil {
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}

func (s *DB) ProblemStatisticsTime(ctx context.Context, problemID int) ([]*kilonova.Submission, error) {
	var subs []*dbSubmission
	err := Select(s.conn, ctx, &subs, `
	WITH top_by_time AS (
		SELECT DISTINCT ON (problem_id, user_id) id, problem_id, user_id, max_time FROM submissions WHERE score = 100 AND problem_id = $1 AND max_time >= 0 ORDER BY problem_id, user_id, max_time ASC
	) SELECT submissions.* FROM submissions, top_by_time WHERE submissions.id = top_by_time.id ORDER BY submissions.max_time LIMIT 5;
`, problemID)
	if err != nil {
		return []*kilonova.Submission{}, err
	}
	return mapper(subs, s.internalToSubmission), nil
}
