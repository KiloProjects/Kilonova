package config

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

// Guards the list-typed flags (frontend.banned_hot_problems) through the JSON
// persistence path and the KN_FLAG_OVERRIDES path.
func TestSliceFlagRoundTrip(t *testing.T) {
	ctx := context.Background()
	f := GenFlag("test.slice", []int{}, "test slice flag")
	f.Update([]int{1, 2})

	p := filepath.Join(t.TempDir(), "flags.json")
	if err := SaveConfigV2(ctx, p); err != nil {
		t.Fatal(err)
	}
	f.Update([]int{})
	if err := LoadConfigV2(ctx, p, true); err != nil {
		t.Fatal(err)
	}
	if got := f.Value(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("after load: got %v, want [1 2]", got)
	}

	t.Setenv("KN_FLAG_OVERRIDES", "test.slice=[3]")
	if err := LoadConfigV2(ctx, p, true); err != nil {
		t.Fatal(err)
	}
	if got := f.Value(); !slices.Equal(got, []int{3}) {
		t.Fatalf("after override: got %v, want [3]", got)
	}
}
