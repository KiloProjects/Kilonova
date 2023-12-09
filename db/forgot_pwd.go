package db

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) CreatePwdResetRequest(ctx context.Context, id int) (string, error) {
	// since RandomString might not always be unique, salt it with the ID
	// thus, if there is a collision, at least it will be from that user already
	vidB := md5.Sum([]byte(kilonova.RandomString(16) + strconv.Itoa(id)))
	vid := hex.EncodeToString(vidB[:])
	_, err := s.conn.Exec(ctx, `INSERT INTO pwd_reset_requests (id, user_id) VALUES ($1, $2)`, vid, id)
	return vid, err
}

func (s *DB) GetPwdResetRequest(ctx context.Context, id string) (int, error) {
	var request struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := s.conn.QueryRow(ctx, `SELECT id, created_at, user_id FROM pwd_reset_requests WHERE id = $1`, id).Scan(&request.ID, &request.CreatedAt, &request.UserID)
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
	_, err := s.conn.Exec(ctx, `DELETE FROM pwd_reset_requests WHERE id = $1`, req)
	return err
}
