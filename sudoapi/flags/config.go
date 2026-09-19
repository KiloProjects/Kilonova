package flags

import (
	"crypto/rand"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/google/uuid"
)

var (
	BMACWebhookSecret = config.GenFlag[string]("frontend.donation.bmac_webhook_secret", "", "Secret validation ID for Buy Me a Coffee notifications")
)

var FilterUserAgent = config.GenFlag[bool]("behavior.user_agent_filter", true, "Filter user agent in API (block python requests from non-admins)")

var (
	AuthRSAPrivateKey   = config.GenFlag("oidc.rsaKey.private", "", "RSA private key for the auth server")
	AuthRSAPrivateKeyID = config.GenFlag("oidc.rsaKey.id", uuid.Must(uuid.NewV7()).String(), "RSA private key ID for the auth server")
	AuthCryptoKey       = config.GenFlag("oidc.cryptoKey", rand.Text(), "Crypto key for the auth server")
)

var MossUserID = config.GenFlag("integrations.moss.user_id", -1, "User ID for MOSS Plagiarism Checker")

// OpenAI: token and models are KN_OPENAI_* environment variables (net/llm.Provider is
// built in cmd/kn); the token is a secret and must not reach templates.
