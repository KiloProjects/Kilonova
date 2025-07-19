package auth

import (
	"time"

	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Token struct {
	ID            uuid.UUID  `db:"id"`
	CreatedAt     time.Time  `db:"created_at"`
	ExpiresAt     time.Time  `db:"expires_at"`
	ApplicationID uuid.UUID  `db:"application_id"`
	UserID        *int       `db:"user_id"`
	Scopes        []string   `db:"scopes"`
	Audience      []string   `db:"audience"`
	ParentToken   *uuid.UUID `db:"from_token"`
	TokenType     TokenType  `db:"token_type"`
}
