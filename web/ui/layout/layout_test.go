package layout

import (
	"context"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type fakeAssets struct{}

func (fakeAssets) Asset(_ context.Context, s string) string { return "/static/misc/" + s }

func TestView(t *testing.T) {
	base := context.WithValue(context.Background(), util.LangKey, "en")
	base = context.WithValue(base, MiddlewareStartKey, time.Now())
	as := func(u *kilonova.UserFull) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }

	t.Run("renders content, defaults and footer", func(t *testing.T) {
		got := render(t, View(base, Props{Assets: fakeAssets{}, Content: P(Text("hello"))}))
		for _, want := range []string{"<!doctype html>", `<html lang="en"`, "<title>Kilonova</title>", "<p>hello</p>", "<footer", `src="/static/misc/app.ts"`} {
			if !strings.Contains(got, want) {
				t.Fatalf("missing %q in %s", want, got)
			}
		}
	})

	t.Run("footer is built after the content renders", func(t *testing.T) {
		if testing.Short() {
			t.Skip("sleeps during render")
		}
		slow := NodeFunc(func(io.Writer) error { time.Sleep(25 * time.Millisecond); return nil })
		got := render(t, View(base, Props{Assets: fakeAssets{}, Content: slow}))
		m := regexp.MustCompile(`Render duration: ([^<]+)<`).FindStringSubmatch(got)
		if m == nil {
			t.Fatalf("no render duration in %s", got)
		}
		if d, err := time.ParseDuration(m[1]); err != nil || d < 20*time.Millisecond {
			t.Fatalf("render duration %q doesn't cover the content render; footer built too early?", m[1])
		}
	})

	t.Run("HideFooter drops the footer", func(t *testing.T) {
		if got := render(t, View(base, Props{Assets: fakeAssets{}, HideFooter: true})); strings.Contains(got, "<footer") {
			t.Fatal("footer rendered")
		}
	})

	t.Run("unverified email banner only for unverified users", func(t *testing.T) {
		u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 2, Name: "bob"}}
		if got := render(t, View(as(u), Props{Assets: fakeAssets{}})); !strings.Contains(got, "bg-green-200") {
			t.Fatal("banner missing for unverified user")
		}
		u.VerifiedEmail = true
		if got := render(t, View(as(u), Props{Assets: fakeAssets{}})); strings.Contains(got, "bg-green-200") {
			t.Fatal("banner shown for verified user")
		}
	})

	t.Run("username can't break out of the platform_info script", func(t *testing.T) {
		u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{ID: 2, Name: "</script><script>alert(1)</script>"}, VerifiedEmail: true}
		if got := render(t, View(as(u), Props{Assets: fakeAssets{}})); strings.Contains(got, "<script>alert(1)") {
			t.Fatal("username escaped the JSON script")
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
