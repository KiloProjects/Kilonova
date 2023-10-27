package db

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

func (s *DB) ResetSubmission(ctx context.Context, subID int) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		if err := clearSub(ctx, tx, subID); err != nil {
			return err
		}
		if err := initSub(ctx, tx, subID); err != nil {
			return err
		}
		return nil
	})
}

func (s *DB) ResetProblemSubmissions(ctx context.Context, problemID int) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		if err := clearProblemSubs(ctx, tx, problemID); err != nil {
			return err
		}
		if err := initProblemSubs(ctx, tx, problemID); err != nil {
			return err
		}
		return nil
	})
}

func (s *DB) InitSubmission(ctx context.Context, subID int) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		return initSub(ctx, tx, subID)
	})
}

// Don't forget to update clearProblemSubs
func clearSub(ctx context.Context, tx pgx.Tx, subID int) error {
	// Reset submission data:
	if _, err := tx.Exec(ctx, "UPDATE submissions SET status = 'creating', score = 0, max_time = -1, max_memory = -1, compile_error = false, compile_message = '' WHERE id = $1", subID); err != nil {
		return err
	}

	// Remove subtests
	if _, err := tx.Exec(ctx, `DELETE FROM submission_tests WHERE submission_id = $1`, subID); err != nil {
		return err
	}

	// Remove subtasks
	_, err := tx.Exec(ctx, `DELETE FROM submission_subtasks WHERE submission_id = $1`, subID)

	return err
}

// Don't forget to update initProblemSubs
func initSub(ctx context.Context, tx pgx.Tx, subID int) error {
	if subID <= 0 {
		return kilonova.ErrMissingRequired
	}

	// Set precision before everything, since subtask initialization is relying on it
	if _, err := tx.Exec(ctx, `
	UPDATE submissions SET 
		digit_precision = COALESCE((SELECT digit_precision FROM problems WHERE problems.id = problem_id), digit_precision),
		submission_type = 
			CASE (SELECT scoring_strategy FROM problems WHERE problems.id = problem_id)
				WHEN 'acm-icpc' THEN 'acm-icpc'::eval_type
				ELSE 'classic'::eval_type
			END
	WHERE id = $1`, subID); err != nil {
		return err
	}

	// Init subtests
	if _, err := tx.Exec(ctx, `
	INSERT INTO submission_tests (user_id, created_at, submission_id, contest_id, test_id, visible_id, score) 
		SELECT subs.user_id, subs.created_at AS created_at, subs.id AS submission_id, subs.contest_id, tests.id AS test_id, tests.visible_id, tests.score AS score 
		FROM submissions subs, tests 
		WHERE subs.problem_id = tests.problem_id AND subs.id = $1
`, subID); err != nil {
		return err
	}

	// Init subtasks
	if _, err := tx.Exec(ctx, `
INSERT INTO submission_subtasks 
	(user_id, created_at, submission_id, contest_id, subtask_id, problem_id, visible_id, digit_precision, score) 
	SELECT subs.user_id, subs.created_at AS created_at, subs.id AS submission_id, subs.contest_id, stks.id AS subtask_id, stks.problem_id AS problem_id, stks.visible_id, subs.digit_precision AS digit_precision, stks.score AS score 
	FROM submissions subs, subtasks stks 
	WHERE subs.problem_id = stks.problem_id AND subs.id = $1
`, subID); err != nil {
		return err
	}

	// Create link between subtests and subtasks
	_, err := tx.Exec(ctx, `
	INSERT INTO submission_subtask_subtests 
		SELECT sstk.id, st.id
		FROM subtask_tests stks
		INNER JOIN submission_subtasks sstk ON stks.subtask_id = sstk.subtask_id
		INNER JOIN submission_tests st ON stks.test_id = st.test_id AND sstk.submission_id = st.submission_id WHERE st.submission_id = $1;
	`, subID)

	// Update to waiting
	if _, err := tx.Exec(ctx, "UPDATE submissions SET status = 'waiting' WHERE id = $1", subID); err != nil {
		return err
	}

	return err
}

// Don't forget to update clearSub
func clearProblemSubs(ctx context.Context, tx pgx.Tx, problemID int) error {
	if problemID <= 0 {
		return kilonova.ErrMissingRequired
	}
	// Reset submission data:
	if _, err := tx.Exec(ctx, "UPDATE submissions SET status = 'creating', score = 0, max_time = -1, max_memory = -1, compile_error = false, compile_message = '' WHERE problem_id = $1", problemID); err != nil {
		return err
	}

	// Remove subtests
	if _, err := tx.Exec(ctx, `DELETE FROM submission_tests subtests USING submissions subs WHERE subs.problem_id = $1 AND subtests.submission_id = subs.id`, problemID); err != nil {
		return err
	}

	// Remove subtasks
	_, err := tx.Exec(ctx, `DELETE FROM submission_subtasks stks USING submissions subs WHERE subs.problem_id = $1 AND stks.submission_id = subs.id`, problemID)

	return err
}

// Don't forget to update initSub
func initProblemSubs(ctx context.Context, tx pgx.Tx, problemID int) error {
	if problemID <= 0 {
		return kilonova.ErrMissingRequired
	}

	// Set precision before everything, since subtask initialization is relying on it
	if _, err := tx.Exec(ctx, `
	UPDATE submissions SET 
		digit_precision = COALESCE((SELECT digit_precision FROM problems WHERE problems.id = problem_id), digit_precision),
		submission_type = 
			CASE (SELECT scoring_strategy FROM problems WHERE problems.id = problem_id)
				WHEN 'acm-icpc' THEN 'acm-icpc'::eval_type
				ELSE 'classic'::eval_type
			END
	WHERE problem_id = $1`, problemID); err != nil {
		return err
	}

	// Initialize subtests
	if _, err := tx.Exec(ctx, `
INSERT INTO submission_tests (user_id, created_at, submission_id, contest_id, test_id, visible_id, score) 
	SELECT subs.user_id, subs.created_at AS created_at, subs.id AS submission_id, subs.contest_id, tests.id AS test_id, tests.visible_id, tests.score AS score 
	FROM submissions subs, tests 
	WHERE subs.problem_id = tests.problem_id AND subs.problem_id = $1
`, problemID); err != nil {
		return err
	}

	// Initialize subtasks
	if _, err := tx.Exec(ctx, `
INSERT INTO submission_subtasks 
	(user_id, created_at, submission_id, contest_id, subtask_id, problem_id, visible_id, digit_precision, score) 
	SELECT subs.user_id, subs.created_at AS created_at, subs.id AS submission_id, subs.contest_id, stks.id AS subtask_id, stks.problem_id AS problem_id, stks.visible_id, subs.digit_precision AS digit_precision, stks.score AS score 
	FROM submissions subs, subtasks stks 
	WHERE subs.problem_id = stks.problem_id AND subs.problem_id = $1
`, problemID); err != nil {
		return err
	}

	// Create link between subtests and subtasks
	_, err := tx.Exec(ctx, `
INSERT INTO submission_subtask_subtests 
	SELECT sstk.id, st.id
	FROM subtask_tests stks
	INNER JOIN submission_subtasks sstk ON stks.subtask_id = sstk.subtask_id
	INNER JOIN submission_tests st ON stks.test_id = st.test_id AND sstk.submission_id = st.submission_id
	WHERE EXISTS (SELECT 1 FROM submissions subs WHERE subs.id = st.submission_id AND subs.problem_id = $1)
	`, problemID)

	// Update to waiting
	if _, err := tx.Exec(ctx, "UPDATE submissions SET status = 'waiting' WHERE problem_id = $1", problemID); err != nil {
		return err
	}

	return err
}
