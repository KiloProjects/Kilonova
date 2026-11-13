package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func (s *API) AdminV2() http.Handler {
	humaConf := huma.DefaultConfig("Kilonova Admin", "2.0")
	humaConf.Servers = []*huma.Server{
		{URL: s.base.HostURL().JoinPath("api/admin/v2").String()},
	}
	humaConf.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		// "session": {
		//	Type: "apiKey",
		//	In:   "cookie",
		//	Name: "kn-sessionid",
		// },
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-Api-Key",
		},
		"oauth": {
			Type:             "openIdConnect",
			OpenIDConnectURL: s.base.HostURL().JoinPath(oidc.DiscoveryEndpoint).String(),
		},
	}

	r := chi.NewRouter()
	api := humachi.New(r, humaConf)
	api.UseMiddleware(s.SetupSessionV2(api), s.CheckScopes(api))
	api.UseMiddleware(s.MustBeAdminV2(api))

	huma.Register(api, huma.Operation{
		OperationID: "generate-user",
		Method:      http.MethodPost,
		Path:        "/generateUser",
		Security:    []map[string][]string{},
	}, s.generateUser)

	return r
}
