package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraderClientsFromEnv(t *testing.T) {
	got, err := GraderClientsFromEnv([]string{"PATH=/bin", "KN_GRADER_CLIENT_STAGING=def456", "KN_GRADER_LISTEN=:9000", "KN_GRADER_CLIENT_KILONOVA=abc=123"})
	require.NoError(t, err)
	want := []GraderClientConf{{"kilonova", "abc=123"}, {"staging", "def456"}}
	require.Equal(t, want, got)
	for _, bad := range [][]string{{}, {"KN_GRADER_CLIENT_=tok"}, {"KN_GRADER_CLIENT_X="}} {
		_, err := GraderClientsFromEnv(bad)
		assert.Error(t, err, "%q: expected error", bad)
	}
}
