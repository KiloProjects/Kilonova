// Package uitest holds the assertions shared by the web/ui view tests.
package uitest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	g "maragu.dev/gomponents"
)

// Render renders n to a string, failing the test on error.
func Render(t testing.TB, n g.Node) string {
	t.Helper()
	var b strings.Builder
	require.NoError(t, n.Render(&b))
	return b.String()
}

// Contains asserts that got contains every one of want.
func Contains(t testing.TB, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		assert.Contains(t, got, w)
	}
}

// NotContains asserts that got contains none of bad.
func NotContains(t testing.TB, got string, bad ...string) {
	t.Helper()
	for _, w := range bad {
		assert.NotContains(t, got, w)
	}
}
