package sudoapi

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// TODO: Do not embed this here and instead make web/ directly use oidc provider
func (s *BaseAPI) OIDCProvider() *op.Provider {
	return s.oidcProvider
}

func (s *BaseAPI) RegisterOIDCRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		// Things mostly copy pasted from zitadel/oidc due to not accepting receiving a router (so we don't have to mount it)
		r.Use(cors.New(cors.Options{
			AllowCredentials: true,
			AllowedHeaders: []string{
				"Origin",
				"Accept",
				"Accept-Language",
				"Authorization",
				"Content-Type",
				"X-Requested-With",
			},
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
			},
			ExposedHeaders: []string{
				"Location",
				"Content-Length",
			},
			AllowOriginFunc: func(_ string) bool {
				return true
			},
		}).Handler)
		r.Use(op.NewIssuerInterceptor(s.oidcProvider.IssuerFromRequest).Handler)
		//r.HandleFunc(healthEndpoint, healthHandler)
		//r.HandleFunc(readinessEndpoint, readyHandler(o.Probes()))
		r.HandleFunc(oidc.DiscoveryEndpoint, func(w http.ResponseWriter, r *http.Request) {
			op.Discover(w, op.CreateDiscoveryConfig(r.Context(), s.oidcProvider, s.oidcProvider.Storage()))
		})
		r.HandleFunc(s.oidcProvider.AuthorizationEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.Authorize(w, r, s.oidcProvider)
		})
		r.HandleFunc(s.oidcProvider.AuthorizationEndpoint().Relative()+"/callback", op.AuthorizeCallbackHandler(s.oidcProvider))
		r.HandleFunc(s.oidcProvider.TokenEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.Exchange(w, r, s.oidcProvider)
		})
		r.HandleFunc(s.oidcProvider.IntrospectionEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.Introspect(w, r, s.oidcProvider)
		})
		r.HandleFunc(s.oidcProvider.UserinfoEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.Userinfo(w, r, s.oidcProvider)
		})
		r.HandleFunc(s.oidcProvider.RevocationEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.Revoke(w, r, s.oidcProvider)
		})
		r.HandleFunc(s.oidcProvider.EndSessionEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.EndSession(w, r, s.oidcProvider)
		})
		r.HandleFunc(s.oidcProvider.KeysEndpoint().Relative(), func(w http.ResponseWriter, r *http.Request) {
			op.Keys(w, r, s.oidcProvider.Storage())
		})
		r.HandleFunc(s.oidcProvider.DeviceAuthorizationEndpoint().Relative(), op.DeviceAuthorizationHandler(s.oidcProvider))
	})
}

func (s *BaseAPI) ApproveAuthRequest(ctx context.Context, reqID string, userID int) error {
	return s.oidcProvider.Storage().(*auth.AuthStorage).ApproveAuthRequest(ctx, reqID, userID)
}

func (s *BaseAPI) CreateClient(ctx context.Context, name string, appType auth.ApplicationType, authorID int, devMode bool, allowedRedirects []string, allowedPostLogoutRedirects []string) (uuid.UUID, string, error) {
	return s.oidcProvider.Storage().(*auth.AuthStorage).CreateClient(ctx, name, appType, authorID, devMode, allowedRedirects, allowedPostLogoutRedirects)
}

func (s *BaseAPI) GetAuthRequest(ctx context.Context, reqID string) (*auth.Request, error) {
	req, err := s.oidcProvider.Storage().(*auth.AuthStorage).AuthRequestByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	return req.(*auth.Request), nil
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
