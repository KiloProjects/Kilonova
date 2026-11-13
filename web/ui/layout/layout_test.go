package layout

import (
	"context"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type fakeAssets struct{}

func (fakeAssets) Asset(_ context.Context, s string) string { return "/static/misc/" + s }

func TestView(t *testing.T) {
	base := uitest.Ctx("en")
	base = context.WithValue(base, MiddlewareStartKey, time.Now())
	as := func(u *kilonova.UserFull) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }

	t.Run("renders content, defaults and footer", func(t *testing.T) {
		got := uitest.Render(t, View(base, Props{Assets: fakeAssets{}, Content: P(Text("hello"))}))
		for _, want := range []string{"<!doctype html>", `<html lang="en"`, "<title>Kilonova</title>", "<p>hello</p>", "<footer", `src="/static/misc/app.ts"`} {
			require.Contains(t, got, want)
		}
	})

	t.Run("footer is built after the content renders", func(t *testing.T) {
		if testing.Short() {
			t.Skip("sleeps during render")
		}
		slow := NodeFunc(func(io.Writer) error { time.Sleep(25 * time.Millisecond); return nil })
		got := uitest.Render(t, View(base, Props{Assets: fakeAssets{}, Content: slow}))
		m := regexp.MustCompile(`Render duration: ([^<]+)<`).FindStringSubmatch(got)
		require.NotNil(t, m, "no render duration in %s", got)
		d, err := time.ParseDuration(m[1])
		require.NoError(t, err)
		require.GreaterOrEqual(t, d, 20*time.Millisecond, "render duration doesn't cover the content render; footer built too early?")
	})

	t.Run("HideFooter drops the footer", func(t *testing.T) {
		require.NotContains(t, uitest.Render(t, View(base, Props{Assets: fakeAssets{}, HideFooter: true})), "<footer")
	})

	t.Run("unverified email banner only for unverified users", func(t *testing.T) {
		u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 2, Name: "bob"}}
		require.Contains(t, uitest.Render(t, View(as(u), Props{Assets: fakeAssets{}})), "bg-green-200", "banner missing for unverified user")
		u.VerifiedEmail = true
		require.NotContains(t, uitest.Render(t, View(as(u), Props{Assets: fakeAssets{}})), "bg-green-200", "banner shown for verified user")
	})

	t.Run("username can't break out of the platform_info script", func(t *testing.T) {
		u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 2, Name: "</script><script>alert(1)</script>"}, VerifiedEmail: true}
		require.NotContains(t, uitest.Render(t, View(as(u), Props{Assets: fakeAssets{}})), "<script>alert(1)", "username escaped the JSON script")
	})
}
