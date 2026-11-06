package navbar

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

func TestView(t *testing.T) {
	base := context.WithValue(context.Background(), util.LangKey, "en")
	as := func(u *kilonova.UserBrief) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }

	t.Run("visitor gets login with back link and no dropdown", func(t *testing.T) {
		got := render(t, View(base, false, "/problems/1"))
		if !strings.Contains(got, `href="/login?back=/problems/1"`) || strings.Contains(got, "profile-dropdown") {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("user gets escaped name, no admin links", func(t *testing.T) {
		got := render(t, View(as(&kilonova.UserBrief{ID: 2, Name: "<b>"}), false, "/"))
		if !strings.Contains(got, "&lt;b&gt;") || strings.Contains(got, `href="/admin"`) || strings.Contains(got, "/login") {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("admin gets admin and proposer links", func(t *testing.T) {
		got := render(t, View(as(&kilonova.UserBrief{ID: 1, Name: "a", Admin: true}), false, "/"))
		if !strings.Contains(got, `href="/admin"`) || !strings.Contains(got, `href="/proposer"`) {
			t.Fatalf("got %s", got)
		}
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
