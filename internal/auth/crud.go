package auth

import (
	"context"
	"crypto/rand"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *AuthStorage) CreateClient(ctx context.Context, name string, appType ApplicationType, authorID int, devMode bool, allowedRedirects []string, allowedPostLogoutRedirects []string) (uuid.UUID, string, error) {
	secret := rand.Text()
	id := uuid.Must(uuid.NewV7())

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, "", err
	}

	if _, err := s.conn.Exec(ctx, `
		INSERT INTO oauth_clients (id, app_type, author_id, secret_hash, allowed_redirects, allowed_post_logout_redirects, developer_mode, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, appType, authorID, hash, allowedRedirects, allowedPostLogoutRedirects, devMode, name); err != nil {
		return uuid.Nil, "", err
	}

	return id, secret, nil
}

func (s *AuthStorage) ApproveAuthRequest(ctx context.Context, reqID string, userID int) error {
	_, err := s.conn.Exec(ctx, `
		UPDATE oauth_requests
		SET request_done = true, user_id = $2
		WHERE id = $1
	`, reqID, userID)

	return err
}
