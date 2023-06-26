package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) ContestRegistrations(ctx context.Context, contestID int, fuzzyName *string, limit, offset int) ([]*kilonova.ContestRegistration, error) {
	var reg []*kilonova.ContestRegistration
	additionalQ := ""
	args := []any{contestID}
	if fuzzyName != nil {
		additionalQ = " AND EXISTS (SELECT 1 FROM users WHERE id = user_id AND position(lower(unaccent($2)) in format('#%s %s', id, lower(unaccent(name)))) > 0) "
		args = append(args, fuzzyName)
	}
	err := s.conn.SelectContext(ctx, &reg, "SELECT * FROM contest_registrations WHERE contest_id = $1 "+additionalQ+" ORDER BY created_at ASC "+FormatLimitOffset(limit, offset), args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*kilonova.ContestRegistration{}, nil
		}
		return nil, err
	}
	return reg, nil
}

func (s *DB) ContestRegistrationCount(ctx context.Context, contestID int) (int, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(*) FROM contest_registrations WHERE contest_id = $1", contestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, nil
		}
		return -1, err
	}
	return cnt, nil
}

func (s *DB) ContestRegistration(ctx context.Context, contestID, userID int) (*kilonova.ContestRegistration, error) {
	var reg kilonova.ContestRegistration
	err := s.conn.GetContext(ctx, &reg, "SELECT * FROM contest_registrations WHERE contest_id = $1 AND user_id = $2 LIMIT 1", contestID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &reg, nil
}

func (s *DB) InsertContestRegistration(ctx context.Context, contestID, userID int) error {
	_, err := s.pgconn.Exec(ctx, "INSERT INTO contest_registrations (user_id, contest_id) VALUES ($1, $2)", userID, contestID)
	return err
}

func (s *DB) StartContestRegistration(ctx context.Context, contestID, userID int, startTime time.Time, endTime time.Time) error {
	_, err := s.pgconn.Exec(ctx, "UPDATE contest_registrations SET individual_start_at = $1, individual_end_at = $2 WHERE contest_id = $3 AND user_id = $4", startTime, endTime, contestID, userID)
	return err
}

func (s *DB) DeleteContestRegistration(ctx context.Context, contestID, userID int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM contest_registrations WHERE user_id = $1 AND contest_id = $2", userID, contestID)
	return err
}
