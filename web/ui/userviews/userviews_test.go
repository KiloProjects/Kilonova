package userviews

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/bwmarrin/discordgo"
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
		got := uitest.Render(t, Topbar(visitor, target, "submissions"))
		uitest.Contains(t, got, `<a class="p-1 topbar-selected" href="/profile/bob/submissions">`)
		uitest.NotContains(t, got, "bob@x.y", `href="/settings"`, "#5")
	})
	t.Run("self sees email and settings tab", func(t *testing.T) {
		got := uitest.Render(t, Topbar(self, target, "settings"))
		uitest.Contains(t, got, "bob@x.y", `<a class="p-1 topbar-selected" href="/settings">`, "resendEmail()")
		uitest.NotContains(t, got, "#5")
	})
	t.Run("admin sees id and email, not settings", func(t *testing.T) {
		got := uitest.Render(t, Topbar(admin, target, "overview"))
		uitest.Contains(t, got, "#5", "bob@x.y")
		uitest.NotContains(t, got, `href="/settings"`)
	})
	t.Run("display name is escaped", func(t *testing.T) {
		u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 2, Name: "a", DisplayName: "<b>"}}
		uitest.Contains(t, uitest.Render(t, Topbar(visitor, u, "overview")), "&lt;b&gt; (a)")
	})
}

func TestProfile(t *testing.T) {
	p := ProfileParams{User: target, BioHTML: "<p>hi</p>", SolvedCount: 1}
	t.Run("admin gets the manage form", func(t *testing.T) {
		got := uitest.Render(t, Profile(admin, p))
		uitest.Contains(t, got, "managerUserForm", "/user/byID/5/moderation/manage", "deleteAccount()", "<p>hi</p>", "kn-pb-solvedby")
		uitest.NotContains(t, got, "/settings#bio")
	})
	t.Run("self gets the edit bio link, not the manage form", func(t *testing.T) {
		got := uitest.Render(t, Profile(self, p))
		uitest.Contains(t, got, "/settings#bio")
		uitest.NotContains(t, got, "managerUserForm")
	})
	t.Run("history shown only with more than one entry", func(t *testing.T) {
		p := p
		p.ChangeHistory = []*kilonova.UsernameChange{{Name: "<o>"}, {Name: "bob"}}
		uitest.Contains(t, uitest.Render(t, Profile(visitor, p)), "<code>&lt;o&gt;</code>")
		p.ChangeHistory = p.ChangeHistory[:1]
		uitest.NotContains(t, uitest.Render(t, Profile(visitor, p)), "<code>")
	})
}

func TestSettings(t *testing.T) {
	u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 5, Name: "bob"}, Bio: "a &amp; <b>", PreferredLanguage: "ro", AvatarType: "discord"}
	got := uitest.Render(t, Settings(as(&u.UserBrief), SettingsParams{User: u, AvatarHelpHTML: "<p>help</p>"}))
	uitest.Contains(t, got, "a &amp; &lt;b&gt;</textarea>", `<option value="ro" selected>`, `<option value="discord" selected>`, "<p>help</p>")
	uitest.NotContains(t, got, "/settings/oauth")
}

func TestDiscordLink(t *testing.T) {
	d := &discordgo.User{ID: "99", Username: "disc"}
	uitest.Contains(t, uitest.Render(t, DiscordLink(admin, target, d)), "<code>99</code>", "/profile/bob/discordAvatar")
	uitest.NotContains(t, uitest.Render(t, DiscordLink(self, target, d)), "<code>")
	uitest.NotContains(t, uitest.Render(t, DiscordLink(self, target, nil)), "discordAvatar")
}
