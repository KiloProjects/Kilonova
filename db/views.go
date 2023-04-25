package db

import "context"

// Functions that interact with views and functions

func (s *DB) IsProblemViewer(ctx context.Context, problemID, userID int) (bool, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(*) FROM visible_pbs($2) WHERE problem_id = $1", problemID, userID)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (s *DB) IsProblemEditor(ctx context.Context, problemID, userID int) (bool, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(*) FROM problem_editors WHERE problem_id = $1 AND user_id = $2", problemID, userID)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (s *DB) IsContestViewer(ctx context.Context, contestID, userID int) (bool, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(*) FROM contest_visibility WHERE contest_id = $1 AND user_id = $2", contestID, userID)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}
