package sudoapi

import (
	"context"
	"log/slog"

	"uuid"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// TODO: Do not embed this here and instead make web/ directly use oidc provider
func (s *BaseAPI) OIDCProvider() *op.Provider {
	return s.oidcProvider
}

// RegisterOIDCRoutes hands the provider's own paths to it. The provider is already a
// router carrying CORS and the issuer interceptor, so nothing is re-declared here;
// it is not mounted because its paths live at the root, next to the pages.
func (s *BaseAPI) RegisterOIDCRoutes(r chi.Router) {
	for _, route := range s.oidcProvider.Handler.(chi.Routes).Routes() {
		if route.Pattern == "/healthz" || route.Pattern == "/ready" {
			continue // probes are served by cmd/kn
		}
		r.Handle(route.Pattern, s.oidcProvider)
	}
}

func (s *BaseAPI) ApproveAuthRequest(ctx context.Context, reqID string, userID int) error {
	return s.authStorage.ApproveAuthRequest(ctx, reqID, userID)
}

func (s *BaseAPI) CreateClient(ctx context.Context, name string, appType auth.ApplicationType, authorID int, devMode bool, allowedRedirects []string, allowedPostLogoutRedirects []string) (uuid.UUID, string, error) {
	id, secret, err := s.authStorage.CreateClient(ctx, name, appType, authorID, devMode, allowedRedirects, allowedPostLogoutRedirects)
	if err != nil {
		return uuid.Nil(), "", err
	}
	s.LogUserAction(ctx, "Created OAuth client", slog.String("id", id.String()), slog.String("name", name))
	return id, secret, nil
}

// OAuthClients returns all clients, or only those authored by authorID if it is not nil
func (s *BaseAPI) OAuthClients(ctx context.Context, authorID *int) ([]*auth.Client, error) {
	return s.authStorage.ListClients(ctx, authorID)
}

func (s *BaseAPI) DeleteOAuthClient(ctx context.Context, id uuid.UUID) error {
	if err := s.authStorage.DeleteClient(ctx, id); err != nil {
		return err
	}
	s.LogUserAction(ctx, "Deleted OAuth client", slog.String("id", id.String()))
	return nil
}

func (s *BaseAPI) GetAuthRequest(ctx context.Context, reqID string) (*auth.Request, error) {
	req, err := s.authStorage.AuthRequestByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	return req.(*auth.Request), nil
}

func (s *BaseAPI) GetOAuthClient(ctx context.Context, clientID uuid.UUID) (*auth.Client, error) {
	client, err := s.authStorage.GetClientByClientID(ctx, clientID.String())
	if err != nil {
		return nil, err
	}
	return client.(*auth.Client), nil
}

func (s *BaseAPI) GetAccessToken(ctx context.Context, tokenID uuid.UUID) (*auth.Token, error) {
	return s.authStorage.GetAccessToken(ctx, tokenID)
}

func (s *BaseAPI) AuthorizedOAuthClients(ctx context.Context, userID int) ([]*auth.Client, error) {
	return s.authStorage.ListAuthorizedClients(ctx, userID)
}

func (s *BaseAPI) RevokeOAuthClient(ctx context.Context, userID int, clientID uuid.UUID) error {
	return s.authStorage.RevokeClientTokens(ctx, userID, clientID)
}
