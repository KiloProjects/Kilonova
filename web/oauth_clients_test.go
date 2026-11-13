package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRedirects(t *testing.T) {
	got, err := parseRedirects("https://a.com/cb\r\n\n  com.example.app:/cb  \n")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "com.example.app:/cb", got[1])
	for _, bad := range []string{"/relative", "https://a.com/cb#frag", "not a url"} {
		_, err := parseRedirects(bad)
		assert.Error(t, err, "expected %q to be rejected", bad)
	}
}
