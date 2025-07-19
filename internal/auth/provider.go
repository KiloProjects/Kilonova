package auth

import (
	"crypto/sha256"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

func GetProvider(storage *AuthStorage) (http.Handler, error) {
	opConfig := &op.Config{
		CryptoKey:                sha256.Sum256([]byte(CryptoKey.Value())),
		DefaultLogoutRedirectURI: "/",
		CodeMethodS256:           true,

		AuthMethodPost:          true,
		AuthMethodPrivateKeyJWT: false,

		GrantTypeRefreshToken:  true,
		RequestObjectSupported: true,
		SupportedUILocales: []language.Tag{
			language.English,
			language.Romanian,
		},

		SupportedScopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeProfile,
			oidc.ScopeEmail,
			oidc.ScopeOfflineAccess,
		},

		BackChannelLogoutSupported:        true,
		BackChannelLogoutSessionSupported: true,
	}

	opts := []op.Option{
		op.WithLogger(slog.Default().WithGroup("oidc")),
	}
	if config.Common.Debug {
		opts = append(opts, op.WithAllowInsecure())
	}

	handler, err := op.NewProvider(opConfig, storage, op.StaticIssuer(config.Common.HostPrefix), opts...)
	if err != nil {
		return nil, err
	}

	return handler, nil
}
