package config

import (
	"slices"
	"testing"
)

func TestGraderClientsFromEnv(t *testing.T) {
	got, err := GraderClientsFromEnv([]string{"PATH=/bin", "KN_GRADER_CLIENT_STAGING=def456", "KN_GRADER_LISTEN=:9000", "KN_GRADER_CLIENT_KILONOVA=abc=123"})
	if err != nil {
		t.Fatal(err)
	}
	want := []GraderClientConf{{"kilonova", "abc=123"}, {"staging", "def456"}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, bad := range [][]string{{}, {"KN_GRADER_CLIENT_=tok"}, {"KN_GRADER_CLIENT_X="}} {
		if _, err := GraderClientsFromEnv(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}
