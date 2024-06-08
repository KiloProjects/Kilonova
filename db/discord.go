package db

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

func (s *DB) CreateDiscordState(ctx context.Context, id int) (string, error) {
	vid, err := uuid.NewV7()
	if err != nil {
		slog.Warn("Could not generate UUID", slog.Any("err", err))
		return "", err
	}
	_, err = s.conn.Exec(ctx, `INSERT INTO discord_oauth_secrets (id, user_id) VALUES ($1, $2)`, vid.String(), id)
	return vid.String(), err
}

func (s *DB) GetDiscordState(ctx context.Context, id string) (int, error) {
	var request struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := s.conn.QueryRow(ctx, "SELECT id, created_at, user_id FROM discord_oauth_secrets WHERE id = $1", id).Scan(&request.ID, &request.CreatedAt, &request.UserID)
	if err != nil {
		return -1, err
	}
	// 1 hour limit
	if time.Since(request.CreatedAt) > time.Hour {
		return -1, err
	}
	return request.UserID, err
}

func (s *DB) RemoveDiscordState(ctx context.Context, id string) error {
	_, err := s.conn.Exec(ctx, `DELETE FROM discord_oauth_secrets WHERE id = $1`, id)
	return err
}
