package home

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestIndex(t *testing.T) {
	hundred := decimal.NewFromInt(100)
	pbs := []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 1, Name: "A"}, MaxScore: &hundred}, {Problem: kilonova.Problem{ID: 2, Name: "B"}}}

	t.Run("default announcement and markdown are trusted, custom flag text is markdown", func(t *testing.T) {
		got := render(t, Index(ctx, Params{ShowAnnouncement: true, AnnouncementHTML: "<em>hi</em>"}))
		must(t, got, "<em>hi</em>")
		got = render(t, Index(ctx, Params{}))
		mustNot(t, got, "text-gray-600 dark:text-gray-300 mb-1")
	})
	t.Run("login form only for visitors when enabled", func(t *testing.T) {
		must(t, render(t, Index(ctx, Params{MainPageLogin: true})), `id="login_form"`, `postCall("/auth/login"`)
		mustNot(t, render(t, Index(ctx, Params{MainPageLogin: true, Authed: true})), `id="login_form"`)
	})
	t.Run("unlisted problems: solved count when authed, total otherwise; open when no lists", func(t *testing.T) {
		got := render(t, Index(ctx, Params{ShowProblems: true, OtherProblems: pbs, Authed: true}))
		must(t, got, `<details class="list-group-head" open>`, "1 / 2 solved")
		got = render(t, Index(ctx, Params{ShowProblems: true, OtherProblems: pbs}))
		must(t, got, "2 problems")
	})
	t.Run("pinned lists and escaping", func(t *testing.T) {
		got := render(t, Index(ctx, Params{Authed: true, PinnedLists: []*kilonova.ProblemList{{ID: 3, Title: "<b>L</b>", NumProblems: 4}}, PinnedNumSolved: map[int]int{3: 2}}))
		must(t, got, `href="/problem_lists/3"`, "&lt;b&gt;L&lt;/b&gt; (#3)", "2 / 4")
	})
	t.Run("sidebar contests: register link per user kind", func(t *testing.T) {
		c := &kilonova.Contest{ID: 7, Name: "C", StartTime: time.Now().Add(time.Hour), EndTime: time.Now().Add(2 * time.Hour)}
		got := render(t, Index(ctx, Params{ContestSidebar: true, FutureContests: []contests.Info{{Contest: c, CanJoin: true, ReqPath: "/"}}}))
		must(t, got, `hidden lg:block`, `href="/login?back=%2F"`, `type="before_start"`)
		got = render(t, Index(ctx, Params{ContestSidebar: true, FutureContests: []contests.Info{{Contest: c, CanJoin: true, Authed: true}}}))
		must(t, got, "bundled.registerForContest(7)")
		got = render(t, Index(ctx, Params{ContestSidebar: true, FutureContests: []contests.Info{{Contest: c, Authed: true, Registration: &kilonova.ContestRegistration{}}}}))
		must(t, got, "- <span>Registered</span>")
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
	for _, w := range bad {
		if strings.Contains(got, w) {
			t.Fatalf("unexpected %q in %s", w, got)
		}
	}
}
