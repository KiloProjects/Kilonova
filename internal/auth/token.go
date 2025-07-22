package auth

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/op"
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

	AMR      []string   `db:"amr"`
	AuthTime *time.Time `db:"auth_time"`
}

var _ op.RefreshTokenRequest = (*RefreshTokenRequest)(nil)

type RefreshTokenRequest struct {
	*Token
}

func (r *RefreshTokenRequest) GetAMR() []string {
	return r.AMR
}

func (r *RefreshTokenRequest) GetAudience() []string {
	return r.Audience
}

func (r *RefreshTokenRequest) GetAuthTime() time.Time {
	if r.AuthTime == nil {
		return time.Time{}
	}
	return *r.AuthTime
}

func (r *RefreshTokenRequest) GetClientID() string {
	return r.ApplicationID.String()
}

func (r *RefreshTokenRequest) GetScopes() []string {
	return r.Scopes
}

func (r *RefreshTokenRequest) GetSubject() string {
	if r.UserID == nil {
		return ""
	}
	return strconv.Itoa(*r.UserID)
}

func (r *RefreshTokenRequest) SetCurrentScopes(scopes []string) {
	r.Scopes = scopes
}
