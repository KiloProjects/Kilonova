package auth

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

var _ op.AuthRequest = (*Request)(nil)

type Request struct {
	ID            uuid.UUID         `db:"id"`
	AuthTime      *time.Time        `db:"auth_time"`
	ApplicationID uuid.UUID         `db:"application_id"`
	CallbackURI   string            `db:"callback_uri"`
	TransferState string            `db:"transfer_state"`
	UserID        *int              `db:"user_id"`
	Scopes        []string          `db:"scopes"`
	ResponseType  oidc.ResponseType `db:"response_type"`
	ResponseMode  oidc.ResponseMode `db:"response_mode"`
	Nonce         string            `db:"nonce"`

	RequestDone bool    `db:"request_done"`
	RequestCode *string `db:"request_code"`

	// CodeChallenge is either:
	// - empty string - no code challenge
	// - the SHA256 hash for the code challenge
	CodeChallenge       string                   `db:"code_challenge"`
	CodeChallengeMethod oidc.CodeChallengeMethod `db:"code_challenge_method"`

	// These are all meant for the UI, not important to the flow
	CreatedAt time.Time      `db:"created_at"`
	Prompt    []string       `db:"prompt"`
	UILocales []language.Tag `db:"ui_locales"`
	LoginHint string         `db:"login_hint"`
	ExpiresAt *time.Time     `db:"expires_at"`
}

func (a *Request) GetID() string {
	return a.ID.String()
}

func (a *Request) GetACR() string {
	return "" // ACR not implemented
}

func (a *Request) GetAMR() []string {
	if a.RequestDone {
		return []string{"pwd"}
	}
	return []string{}
}

func (a *Request) GetAudience() []string {
	return []string{a.ApplicationID.String()}
}

func (a *Request) GetAuthTime() time.Time {
	if a.AuthTime == nil {
		return time.Time{}
	}
	return *a.AuthTime
}

func (a *Request) GetClientID() string {
	return a.ApplicationID.String()
}

func (a *Request) GetCodeChallenge() *oidc.CodeChallenge {
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

func (a *Request) GetNonce() string {
	return a.Nonce
}

func (a *Request) GetRedirectURI() string {
	return a.CallbackURI
}

func (a *Request) GetResponseType() oidc.ResponseType {
	return a.ResponseType
}

func (a *Request) GetResponseMode() oidc.ResponseMode {
	return a.ResponseMode
}

func (a *Request) GetScopes() []string {
	return a.Scopes
}

func (a *Request) GetState() string {
	return a.TransferState
}

func (a *Request) GetSubject() string {
	if a.UserID == nil {
		return ""
	}
	return strconv.Itoa(*a.UserID)
}

func (a *Request) Done() bool {
	return a.RequestDone
}

type OIDCCodeChallenge struct {
	Challenge string
	Method    string
}
