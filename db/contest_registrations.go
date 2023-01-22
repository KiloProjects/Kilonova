package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) ContestRegistrations(ctx context.Context, contestID, limit, offset int) ([]*kilonova.ContestRegistration, error) {
	var reg []*kilonova.ContestRegistration
	err := s.conn.SelectContext(ctx, &reg, "SELECT * FROM contest_registrations WHERE contest_id = $1 "+FormatLimitOffset(limit, offset), contestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*kilonova.ContestRegistration{}, nil
		}
		return nil, err
	}
	return reg, nil
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
	_, err := s.conn.ExecContext(ctx, "INSERT INTO contest_registrations (user_id, contest_id) VALUES ($1, $2)", userID, contestID)
	return err
}
