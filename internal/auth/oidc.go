package auth

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

var _ op.AuthRequest = (*AuthRequest)(nil)

type AuthRequest struct {
	ID            uuid.UUID         `db:"id"`
	AuthTime      time.Time         `db:"auth_time"`
	ApplicationID uuid.UUID         `db:"application_id"`
	CallbackURI   string            `db:"callback_uri"`
	TransferState string            `db:"transfer_state"`
	UserID        int               `db:"user_id"`
	Scopes        []string          `db:"scopes"`
	ResponseType  oidc.ResponseType `db:"response_type"`
	ResponseMode  oidc.ResponseMode `db:"response_mode"`
	Nonce         string            `db:"nonce"`

	RequestDone bool   `db:"request_done"`
	RequestCode string `db:"request_code"`

	// CodeChallenge is either:
	// - empty string - no code challenge
	// - the SHA256 hash for the code challenge
	CodeChallenge       string                   `db:"code_challenge"`
	CodeChallengeMethod oidc.CodeChallengeMethod `db:"code_challenge_method"`

	// These are all meant for the UI, not important to the flow
	CreatedAt time.Time      `db:"created_at"`
	Prompt    []string       `db:"prompt"`
	UiLocales []language.Tag `db:"ui_locales"`
	LoginHint string         `db:"login_hint"`
	ExpiresAt *time.Time     `db:"expires_at"`
}

func (a *AuthRequest) GetID() string {
	return a.ID.String()
}

func (a *AuthRequest) GetACR() string {
	return "" // ACR not implemented
}

func (a *AuthRequest) GetAMR() []string {
	if a.RequestDone {
		return []string{"pwd"}
	}
	return []string{}
}

func (a *AuthRequest) GetAudience() []string {
	return []string{a.ApplicationID.String()}
}

func (a *AuthRequest) GetAuthTime() time.Time {
	return a.AuthTime
}

func (a *AuthRequest) GetClientID() string {
	return a.ApplicationID.String()
}

func (a *AuthRequest) GetCodeChallenge() *oidc.CodeChallenge {
	if a.CodeChallenge == "" {
		return nil
	}
	method := oidc.CodeChallengeMethodPlain
	if a.CodeChallengeMethod == oidc.CodeChallengeMethodS256 {
		method = oidc.CodeChallengeMethodS256
	}
	return &oidc.CodeChallenge{
		Challenge: a.CodeChallenge,
		Method:    method,
	}
}

func (a *AuthRequest) GetNonce() string {
	return a.Nonce
}

func (a *AuthRequest) GetRedirectURI() string {
	return a.CallbackURI
}

func (a *AuthRequest) GetResponseType() oidc.ResponseType {
	return a.ResponseType
}

func (a *AuthRequest) GetResponseMode() oidc.ResponseMode {
	return a.ResponseMode
}

func (a *AuthRequest) GetScopes() []string {
	return a.Scopes
}

func (a *AuthRequest) GetState() string {
	return a.TransferState
}

func (a *AuthRequest) GetSubject() string {
	return strconv.Itoa(a.UserID)
}

func (a *AuthRequest) Done() bool {
	return a.RequestDone
}

type OIDCCodeChallenge struct {
	Challenge string
	Method    string
}
