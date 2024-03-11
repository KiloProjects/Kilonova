package db

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

// Note that ResetSubmissions only properly supports fields that might not be reset (ie. id, problem_id, user_id, etc)
// One workaround for other fields is to get the submissions that you would reset beforehand and pass them to the IDs field
func (s *DB) ResetSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		if err := clearSubs(ctx, tx, filter); err != nil {
			return err
		}
		if err := initSubs(ctx, tx, filter); err != nil {
			return err
		}
		return nil
	})
}

func (s *DB) InitSubmission(ctx context.Context, subID int) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		return initSubs(ctx, tx, kilonova.SubmissionFilter{ID: &subID})
	})
}

func clearSubs(ctx context.Context, tx pgx.Tx, filter kilonova.SubmissionFilter) error {
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)
	// Reset submission data:
	if _, err := tx.Exec(ctx, `
		UPDATE submissions 
			SET status = 'creating', score = 0, max_time = -1, max_memory = -1, compile_error = false, compile_message = '', icpc_verdict = NULL, compile_duration = NULL, leaderboard_score_scale = 100
			WHERE `+fb.Where(), fb.Args()...); err != nil {
		return err
	}

	// Remove subtests
	if _, err := tx.Exec(ctx, fmt.Sprintf(`WITH cleared_subs AS (SELECT * FROM submissions WHERE %s) DELETE FROM submission_tests WHERE EXISTS (SELECT 1 FROM cleared_subs WHERE id = submission_id)`, fb.Where()), fb.Args()...); err != nil {
		return err
	}

	// Remove subtasks
	_, err := tx.Exec(ctx, fmt.Sprintf(`WITH cleared_subs AS (SELECT * FROM submissions WHERE %s) DELETE FROM submission_subtasks WHERE EXISTS (SELECT 1 FROM cleared_subs WHERE id = submission_id)`, fb.Where()), fb.Args()...)

	return err
}

func initSubs(ctx context.Context, tx pgx.Tx, filter kilonova.SubmissionFilter) error {
	fb := newFilterBuilder()
	subFilterQuery(&filter, fb)
	// Set precision before everything, since subtask initialization is relying on it
	if _, err := tx.Exec(ctx, `
	UPDATE submissions SET 
		digit_precision = COALESCE((SELECT digit_precision FROM problems WHERE problems.id = problem_id), digit_precision),
		submission_type = 
			CASE (SELECT scoring_strategy FROM problems WHERE problems.id = problem_id)
				WHEN 'acm-icpc' THEN 'acm-icpc'::eval_type
				ELSE 'classic'::eval_type
			END,
		leaderboard_score_scale = COALESCE((SELECT leaderboard_score_scale FROM problems WHERE problems.id = problem_id), leaderboard_score_scale)
	WHERE `+fb.Where(), fb.Args()...); err != nil {
		return err
	}

	// Init subtests
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
	INSERT INTO submission_tests (created_at, submission_id, test_id, visible_id, score) 
		WITH subs_to_add AS (SELECT * FROM submissions WHERE %s)
		SELECT subs.created_at AS created_at, subs.id AS submission_id, tests.id AS test_id, tests.visible_id, tests.score AS score 
		FROM subs_to_add subs, tests 
		WHERE subs.problem_id = tests.problem_id`, fb.Where()), fb.Args()...); err != nil {
		return err
	}

	// Init subtasks
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
	INSERT INTO submission_subtasks 
	(user_id, created_at, submission_id, contest_id, subtask_id, problem_id, visible_id, digit_precision, score, leaderboard_score_scale) 
		WITH subs_to_add AS (SELECT * FROM submissions WHERE %s)
	SELECT subs.user_id, subs.created_at AS created_at, subs.id AS submission_id, subs.contest_id, stks.id AS subtask_id, stks.problem_id AS problem_id, stks.visible_id, subs.digit_precision AS digit_precision, stks.score AS score, subs.leaderboard_score_scale AS leaderboard_score_scale
	FROM subs_to_add subs, subtasks stks 
	WHERE subs.problem_id = stks.problem_id`, fb.Where()), fb.Args()...); err != nil {
		return err
	}

	// Create link between subtests and subtasks
	_, err := tx.Exec(ctx, fmt.Sprintf(`
	INSERT INTO submission_subtask_subtests 
	SELECT sstk.id, st.id
	FROM subtask_tests stks
	INNER JOIN submission_subtasks sstk ON stks.subtask_id = sstk.subtask_id
	INNER JOIN submission_tests st ON stks.test_id = st.test_id AND sstk.submission_id = st.submission_id
	WHERE EXISTS (SELECT 1 FROM submissions WHERE id = st.submission_id AND %s)`,
		fb.Where()), fb.Args()...)

	// Update to waiting
	if _, err := tx.Exec(ctx, "UPDATE submissions SET status = 'waiting' WHERE "+fb.Where(), fb.Args()...); err != nil {
		return err
	}

	return err
}
