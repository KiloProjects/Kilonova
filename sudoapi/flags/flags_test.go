package flags

import (
	"testing"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/stretchr/testify/require"
)

// Secret flags are left out of exports by default; this pins which ones they are.
func TestSecretFlags(t *testing.T) {
	for _, name := range []string{
		"oidc.rsaKey.private", "oidc.rsaKey.id", "oidc.cryptoKey",
		"admin.important_webhook", "admin.verbose_webhook",
		"frontend.donation.bmac_webhook_secret",
	} {
		f, ok := config.GetFlag[string](name)
		require.True(t, ok, name)
		require.True(t, f.Secret(), name)
	}
	f, ok := config.GetFlag[string]("frontend.navbar.branding")
	require.True(t, ok)
	require.False(t, f.Secret())
}
