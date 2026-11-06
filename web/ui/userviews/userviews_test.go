package userviews

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/bwmarrin/discordgo"
	. "maragu.dev/gomponents"
)

var (
	base    = context.WithValue(context.Background(), util.LangKey, "en")
	as      = func(u *kilonova.UserBrief) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }
	target  = &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 5, Name: "bob"}, Email: "bob@x.y", Bio: "hi"}
	admin   = as(&kilonova.UserBrief{ID: 1, Name: "adm", Admin: true})
	self    = as(&kilonova.UserBrief{ID: 5, Name: "bob"})
	visitor = base
)

func TestTopbar(t *testing.T) {
	t.Run("visitor sees no email, settings or id", func(t *testing.T) {
		got := render(t, Topbar(visitor, target, "submissions"))
		must(t, got, `<a class="p-1 topbar-selected" href="/profile/bob/submissions">`)
		mustNot(t, got, "bob@x.y", `href="/settings"`, "#5")
	})
	t.Run("self sees email and settings tab", func(t *testing.T) {
		got := render(t, Topbar(self, target, "settings"))
		must(t, got, "bob@x.y", `<a class="p-1 topbar-selected" href="/settings">`, "resendEmail()")
		mustNot(t, got, "#5")
	})
	t.Run("admin sees id and email, not settings", func(t *testing.T) {
		got := render(t, Topbar(admin, target, "overview"))
		must(t, got, "#5", "bob@x.y")
		mustNot(t, got, `href="/settings"`)
	})
	t.Run("display name is escaped", func(t *testing.T) {
		u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 2, Name: "a", DisplayName: "<b>"}}
		must(t, render(t, Topbar(visitor, u, "overview")), "&lt;b&gt; (a)")
	})
}

func TestProfile(t *testing.T) {
	p := ProfileParams{User: target, BioHTML: "<p>hi</p>", SolvedCount: 1}
	t.Run("admin gets the manage form", func(t *testing.T) {
		got := render(t, Profile(admin, p))
		must(t, got, "managerUserForm", "/user/byID/5/moderation/manage", "deleteAccount()", "<p>hi</p>", "kn-pb-solvedby")
		mustNot(t, got, "/settings#bio")
	})
	t.Run("self gets the edit bio link, not the manage form", func(t *testing.T) {
		got := render(t, Profile(self, p))
		must(t, got, "/settings#bio")
		mustNot(t, got, "managerUserForm")
	})
	t.Run("history shown only with more than one entry", func(t *testing.T) {
		p := p
		p.ChangeHistory = []*kilonova.UsernameChange{{Name: "<o>"}, {Name: "bob"}}
		must(t, render(t, Profile(visitor, p)), "<code>&lt;o&gt;</code>")
		p.ChangeHistory = p.ChangeHistory[:1]
		mustNot(t, render(t, Profile(visitor, p)), "<code>")
	})
}

func TestSettings(t *testing.T) {
	u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 5, Name: "bob"}, Bio: "a &amp; <b>", PreferredLanguage: "ro", AvatarType: "discord"}
	got := render(t, Settings(as(&u.UserBrief), SettingsParams{User: u, AvatarHelpHTML: "<p>help</p>"}))
	must(t, got, "a &amp; &lt;b&gt;</textarea>", `<option value="ro" selected>`, `<option value="discord" selected>`, "<p>help</p>")
	mustNot(t, got, "/settings/oauth")
}

func TestDiscordLink(t *testing.T) {
	d := &discordgo.User{ID: "99", Username: "disc"}
	must(t, render(t, DiscordLink(admin, target, d)), "<code>99</code>", "/profile/bob/discordAvatar")
	mustNot(t, render(t, DiscordLink(self, target, d)), "<code>")
	mustNot(t, render(t, DiscordLink(self, target, nil)), "discordAvatar")
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
