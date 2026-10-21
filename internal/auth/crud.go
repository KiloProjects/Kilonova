package auth

import (
	"context"
	"crypto/rand"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		SET request_done = true, user_id = $2, auth_time = NOW()
		WHERE id = $1
	`, reqID, userID)

	return err
}

func (s *AuthStorage) GetAccessToken(ctx context.Context, tokenID uuid.UUID) (*Token, error) {
	return s.getAccessToken(ctx, tokenID)
}

// ListClients returns all clients, or only those authored by authorID if it is not nil
func (s *AuthStorage) ListClients(ctx context.Context, authorID *int) ([]*Client, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_clients WHERE ($1::bigint IS NULL OR author_id = $1) ORDER BY created_at DESC", authorID)
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Client])
}

func (s *AuthStorage) DeleteClient(ctx context.Context, id uuid.UUID) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM oauth_clients WHERE id = $1", id)
	return err
}

// ListAuthorizedClients returns the clients that still hold live tokens for the user
func (s *AuthStorage) ListAuthorizedClients(ctx context.Context, userID int) ([]*Client, error) {
	rows, _ := s.conn.Query(ctx, `
		SELECT * FROM oauth_clients WHERE id IN (
			SELECT application_id FROM oauth_tokens WHERE user_id = $1 AND expires_at > NOW()
		) ORDER BY name`, userID)
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Client])
}

// RevokeClientTokens deletes all of the user's tokens for the client
func (s *AuthStorage) RevokeClientTokens(ctx context.Context, userID int, clientID uuid.UUID) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM oauth_tokens WHERE user_id = $1 AND application_id = $2", userID, clientID)
	return err
}
