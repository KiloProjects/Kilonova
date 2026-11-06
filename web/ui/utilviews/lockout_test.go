package utilviews

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

func TestUserLockout(t *testing.T) {
	as := func(name string) context.Context {
		ctx := context.WithValue(context.Background(), util.LangKey, "en")
		return context.WithValue(ctx, user.AuthedUserKey, &kilonova.UserBrief{ID: 2, Name: name})
	}
	t.Run("warns users with a valid name", func(t *testing.T) {
		must(t, render(t, UserLockout(as("bob"))), "<strong class=\"text-lg\">bob</strong>", "text-sm text-muted mb-2")
	})
	t.Run("doesn't warn legacy invalid names", func(t *testing.T) {
		mustNot(t, render(t, UserLockout(as("a b"))), "text-sm text-muted mb-2")
	})
}

func render(t *testing.T, n Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func must(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in %s", w, got)
		}
	}
}

func mustNot(t *testing.T, got string, bad ...string) {
	t.Helper()
	for _, b := range bad {
		if strings.Contains(got, b) {
			t.Fatalf("unexpected %q in %s", b, got)
		}
	}
}
