package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"

	"github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// GeneratePrivateKey returns a new RSA signing key in the encoding StorageConfig expects.
func GeneratePrivateKey() (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(key)), nil
}

func parsePrivateKey(encoded string) (*rsa.PrivateKey, error) {
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS1PrivateKey(der)
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
