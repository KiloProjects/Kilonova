package flags

import (
	"crypto/rand"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/google/uuid"
)

var (
	BMACWebhookSecret = config.GenFlag[string]("frontend.donation.bmac_webhook_secret", "", "Secret validation ID for Buy Me a Coffee notifications")
)

var FilterUserAgent = config.GenFlag[bool]("behavior.user_agent_filter", true, "Filter user agent in API (block python requests from non-admins)")

var (
	ListenHost = config.GenFlag[string]("server.listen.host", "localhost", "Host to listen to")
	ListenPort = config.GenFlag[int]("server.listen.port", 8070, "Port to listen on")
)

var (
	AuthRSAPrivateKey   = config.GenFlag("oidc.rsaKey.private", "", "RSA private key for the auth server")
	AuthRSAPrivateKeyID = config.GenFlag("oidc.rsaKey.id", uuid.Must(uuid.NewV7()).String(), "RSA private key ID for the auth server")
	AuthCryptoKey       = config.GenFlag("oidc.cryptoKey", rand.Text(), "Crypto key for the auth server")
)

var (
	LogDBQueries   = config.GenFlag("behavior.db.log_sql", false, "Log SQL Requests (for debugging purposes)")
	CountDBQueries = config.GenFlag("behavior.db.count_queries", false, "Count SQL Queries (for debugging purposes)")
)

var MaxMindPath = config.GenFlag("integrations.maxmind.db_path", "/usr/share/GeoIP/GeoLite2-City.mmdb", "Path to MaxMind GeoLite2-City database for IPs.")

var MossUserID = config.GenFlag("integrations.moss.user_id", -1, "User ID for MOSS Plagiarism Checker")

var OtelEnabled = config.GenFlag("integrations.otel.enabled", false, "Enable OpenTelemetry collectors")

// openai
var (
	OpenAIBaseURL      = config.GenFlag("integrations.openai.base_url", "", "Base URL for OpenAI API (`https://openrouter.ai/api/v1` can be used for OpenRouter)")
	OpenAIToken        = config.GenFlag("integrations.openai.token", "", "API Key for OpenAI access (used in translating statements)")
	OpenAIDefaultModel = config.GenFlag("integrations.openai.default_model", "gpt-4o", "Default model for LLM translations")
)
