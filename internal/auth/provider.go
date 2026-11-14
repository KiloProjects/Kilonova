package auth

import (
	"crypto/sha256"
	"errors"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

type OauthProviderConfig struct {
	Storage    *AuthStorage
	Debug      bool
	HostPrefix string

	CryptoKey []byte
}

func GetProvider(cfg OauthProviderConfig) (*op.Provider, error) {
	if len(cfg.CryptoKey) == 0 {
		return nil, errors.New("OIDC provider needs a crypto key")
	}
	opConfig := &op.Config{
		CryptoKey:                sha256.Sum256(cfg.CryptoKey),
		DefaultLogoutRedirectURI: "/",
		CodeMethodS256:           true,

		AuthMethodPost:          true,
		AuthMethodPrivateKeyJWT: false,

		GrantTypeRefreshToken: true,
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
	}

	var opts []op.Option
	if cfg.Debug {
		opts = append(opts, op.WithAllowInsecure())
	}

	handler, err := op.NewProvider(opConfig, cfg.Storage, op.StaticIssuer(cfg.HostPrefix), opts...)
	if err != nil {
		return nil, err
	}

	return handler, nil
}
