package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/op"
)

func (s *BaseAPI) OIDCProvider() *op.Provider {
	return s.oidcProvider
}

func (s *BaseAPI) ApproveAuthRequest(ctx context.Context, reqID string, userID int) error {
	return s.oidcProvider.Storage().(*auth.AuthStorage).ApproveAuthRequest(ctx, reqID, userID)
}

func (s *BaseAPI) CreateClient(ctx context.Context, name string, appType auth.ApplicationType, authorID int, devMode bool, allowedRedirects []string, allowedPostLogoutRedirects []string) (uuid.UUID, string, error) {
	return s.oidcProvider.Storage().(*auth.AuthStorage).CreateClient(ctx, name, appType, authorID, devMode, allowedRedirects, allowedPostLogoutRedirects)
}

func (s *BaseAPI) GetAuthRequest(ctx context.Context, reqID string) (*auth.AuthRequest, error) {
	req, err := s.oidcProvider.Storage().(*auth.AuthStorage).AuthRequestByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	return req.(*auth.AuthRequest), nil
}

func (s *BaseAPI) GetOAuthClient(ctx context.Context, clientID uuid.UUID) (*auth.Client, error) {
	client, err := s.oidcProvider.Storage().(*auth.AuthStorage).GetClientByClientID(ctx, clientID.String())
	if err != nil {
		return nil, err
	}
	return client.(*auth.Client), nil
}

func (s *BaseAPI) GetAccessToken(ctx context.Context, tokenID uuid.UUID) (*auth.Token, error) {
	return s.oidcProvider.Storage().(*auth.AuthStorage).GetAccessToken(ctx, tokenID)
}
