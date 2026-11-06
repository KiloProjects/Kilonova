package adminviews

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

func TestUsersList(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	t.Run("rows show roles and display name", func(t *testing.T) {
		got := render(t, UsersList(ctx, UsersListParams{Users: []*kilonova.UserBrief{{ID: 1, Name: "alice", DisplayName: "Al", Admin: true}}}))
		must(t, got, `href="/profile/alice"`, "alice (Al)", "✅", "❌")
		mustNot(t, got, "text-4xl")
	})
	t.Run("empty list shows the placeholder", func(t *testing.T) {
		got := render(t, UsersList(ctx, UsersListParams{}))
		must(t, got, "text-4xl")
		mustNot(t, got, "<table")
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
