package db

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) CreatePwdResetRequest(ctx context.Context, id int) (string, error) {
	vid := kilonova.RandomString(16)
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`INSERT INTO pwd_reset_requests (id, user_id) VALUES (?, ?)`), vid, id)
	return vid, err
}

func (s *DB) GetPwdResetRequest(ctx context.Context, id string) (int, error) {
	var request struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := s.conn.GetContext(ctx, &request, s.conn.Rebind(`SELECT * FROM pwd_reset_requests WHERE id = ?`), id)
	if err != nil {
		return -1, err
	}
	// 1 hour limit
	if time.Since(request.CreatedAt) > time.Hour {
		return -1, err
	}
	return request.UserID, err
}

func (s *DB) RemovePwdResetRequest(ctx context.Context, req string) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`DELETE FROM pwd_reset_requests WHERE id = ?`), req)
	return err
}
