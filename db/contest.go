package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) Contest(ctx context.Context, id int) (*kilonova.Contest, error) {
	var contest kilonova.Contest
	err := s.conn.GetContext(ctx, &contest, "SELECT * FROM contests WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &contest, err
}

func (s *DB) CreateContest(ctx context.Context, name string, description string) (int, error) {
	var id int
	err := s.conn.GetContext(ctx, &id, "INSERT INTO contests (name, description) VALUES ($1, $2)", name, description)
	if err != nil {
		return -1, err
	}
	return id, nil
}
