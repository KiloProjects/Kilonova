package auth

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAuthStorageValidatesKeys(t *testing.T) {
	key, err := GeneratePrivateKey()
	require.NoError(t, err)

	lang := func(context.Context) string { return "en" }
	host := &url.URL{Scheme: "https", Host: "example.com"}
	for name, conf := range map[string]StorageConfig{
		"no key":           {KeyID: "kid", DefaultLanguage: lang, HostURL: host},
		"no key ID":        {PrivateKey: key, DefaultLanguage: lang, HostURL: host},
		"garbage key":      {PrivateKey: "not base64!", KeyID: "kid", DefaultLanguage: lang, HostURL: host},
		"no lang fallback": {PrivateKey: key, KeyID: "kid", HostURL: host},
		"no host URL":      {PrivateKey: key, KeyID: "kid", DefaultLanguage: lang},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewAuthStorage(nil, conf) // validation fails before the pool is touched
			require.Error(t, err)
		})
	}

	parsed, err := parsePrivateKey(key)
	require.NoError(t, err, "generated key must round-trip")
	require.NoError(t, parsed.Validate())
}
