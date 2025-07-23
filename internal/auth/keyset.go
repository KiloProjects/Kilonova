package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	RSAPrivateKey   = config.GenFlag("oidc.rsaKey.private", "", "RSA private key for the auth server")
	RSAPrivateKeyID = config.GenFlag("oidc.rsaKey.id", uuid.Must(uuid.NewV7()).String(), "RSA private key ID for the auth server")
	CryptoKey       = config.GenFlag("oidc.cryptoKey", rand.Text(), "Crypto key for the auth server")
)

func getKey() (*rsa.PrivateKey, error) {
	if RSAPrivateKey.Value() == "" {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			slog.ErrorContext(context.Background(), "Failed to generate RSA key", slog.Any("err", err))
			os.Exit(1)
		}
		privKey := x509.MarshalPKCS1PrivateKey(rsaKey)
		RSAPrivateKey.Update(base64.StdEncoding.EncodeToString(privKey))
		return rsaKey, nil
	}

	privKey, err := base64.StdEncoding.DecodeString(RSAPrivateKey.Value())
	if err != nil {
		return nil, err
	}

	return x509.ParsePKCS1PrivateKey(privKey)
}

var _ op.Key = (*publicKey)(nil)
var _ op.SigningKey = (*signingKey)(nil)

type signingKey struct {
	pkey *rsa.PrivateKey
	kid  string
}

func (s *signingKey) SignatureAlgorithm() jose.SignatureAlgorithm {
	return jose.RS256
}

func (s *signingKey) Key() any {
	return s.pkey
}

func (s *signingKey) ID() string {
	return s.kid
}

type publicKey struct {
	*signingKey
}

func (pk *publicKey) ID() string {
	return pk.kid
}

func (pk *publicKey) Algorithm() jose.SignatureAlgorithm {
	return jose.RS256
}

func (pk *publicKey) Use() string {
	return "sig"
}

func (pk *publicKey) Key() any {
	return pk.pkey.Public()
}
