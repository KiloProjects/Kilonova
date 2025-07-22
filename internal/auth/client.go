package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

type ApplicationType int

const (
	// Referenced from https://zitadel.com/docs/guides/integrate/login/oidc/login-users#oidc--oauth-flow

	// ApplicationTypeWeb is a server-side oriented application
	ApplicationTypeWeb ApplicationType = iota
	// ApplicationTypeNative is a native (ex. mobile, desktop) application
	ApplicationTypeNative
	// ApplicationTypeUserAgent is usually a single-page application (ex. SPA)
	ApplicationTypeUserAgent
)

var _ op.Client = (*Client)(nil)

type Client struct {
	ID                         uuid.UUID       `db:"id"`
	AllowedRedirects           []string        `db:"allowed_redirects"`
	AllowedPostLogoutRedirects []string        `db:"allowed_post_logout_redirects"`
	AppType                    ApplicationType `db:"app_type"`

	SecretHash string `db:"secret_hash"`

	DeveloperMode bool `db:"developer_mode"`

	CreatedAt time.Time `db:"created_at"`
	AuthorID  *int      `db:"author_id"`

	Name string `db:"name"`
}

// GetID must return the client_id
func (c *Client) GetID() string {
	return c.ID.String()
}

// RedirectURIs must return the registered redirect_uris for Code and Implicit Flow
func (c *Client) RedirectURIs() []string {
	return c.AllowedRedirects
}

// PostLogoutRedirectURIs must return the registered post_logout_redirect_uris for sign-outs
func (c *Client) PostLogoutRedirectURIs() []string {
	return c.AllowedPostLogoutRedirects
}

func (c *Client) ApplicationType() op.ApplicationType {
	switch c.AppType {
	case ApplicationTypeWeb:
		return op.ApplicationTypeWeb
	case ApplicationTypeNative:
		return op.ApplicationTypeNative
	case ApplicationTypeUserAgent:
		return op.ApplicationTypeUserAgent
	default:
		panic("invalid application type")
	}
}

func (c *Client) AuthMethod() oidc.AuthMethod {
	if c.AppType == ApplicationTypeNative {
		return oidc.AuthMethodNone
	}
	return oidc.AuthMethodBasic
}

func (c *Client) ResponseTypes() []oidc.ResponseType {
	return []oidc.ResponseType{
		oidc.ResponseTypeCode,
		oidc.ResponseTypeIDToken,
		oidc.ResponseTypeIDTokenOnly,
	}
}

func (c *Client) GrantTypes() []oidc.GrantType {
	return []oidc.GrantType{
		oidc.GrantTypeCode,
		oidc.GrantTypeRefreshToken,
	}
}

func (c *Client) LoginURL(id string) string {
	return "/login?authRequestID=" + id
}

func (c *Client) AccessTokenType() op.AccessTokenType {
	return op.AccessTokenTypeJWT
}

func (c *Client) IDTokenLifetime() time.Duration {
	return 1 * time.Hour
}

func (c *Client) DevMode() bool {
	return c.DeveloperMode
}

func (c *Client) RestrictAdditionalIdTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string {
		return scopes
	}
}

func (c *Client) RestrictAdditionalAccessTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string {
		return scopes
	}
}

func (c *Client) IsScopeAllowed(scope string) bool {
	return false // TODO: Custom scopes
}

func (c *Client) IDTokenUserinfoClaimsAssertion() bool {
	return false
}

func (c *Client) ClockSkew() time.Duration {
	return 0
}
