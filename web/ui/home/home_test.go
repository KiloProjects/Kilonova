package home

import (
	"context"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestIndex(t *testing.T) {
	hundred := decimal.NewFromInt(100)
	pbs := []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 1, Name: "A"}, MaxScore: &hundred}, {Problem: kilonova.Problem{ID: 2, Name: "B"}}}

	t.Run("default announcement and markdown are trusted, custom flag text is markdown", func(t *testing.T) {
		got := uitest.Render(t, Index(ctx, Params{ShowAnnouncement: true, AnnouncementHTML: "<em>hi</em>"}))
		uitest.Contains(t, got, "<em>hi</em>")
		got = uitest.Render(t, Index(ctx, Params{}))
		uitest.NotContains(t, got, "text-gray-600 dark:text-gray-300 mb-1")
	})
	t.Run("login form only for visitors when enabled", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Index(ctx, Params{MainPageLogin: true})), `id="login_form"`, `postCall("/auth/login"`)
		uitest.NotContains(t, uitest.Render(t, Index(ctx, Params{MainPageLogin: true, Authed: true})), `id="login_form"`)
	})
	t.Run("unlisted problems: solved count when authed, total otherwise; open when no lists", func(t *testing.T) {
		got := uitest.Render(t, Index(ctx, Params{ShowProblems: true, OtherProblems: pbs, Authed: true}))
		uitest.Contains(t, got, `<details class="list-group-head" open>`, "1 / 2 solved")
		got = uitest.Render(t, Index(ctx, Params{ShowProblems: true, OtherProblems: pbs}))
		uitest.Contains(t, got, "2 problems")
	})
	t.Run("pinned lists and escaping", func(t *testing.T) {
		got := uitest.Render(t, Index(ctx, Params{Authed: true, PinnedLists: []*kilonova.ProblemList{{ID: 3, Title: "<b>L</b>", NumProblems: 4}}, PinnedNumSolved: map[int]int{3: 2}}))
		uitest.Contains(t, got, `href="/problem_lists/3"`, "&lt;b&gt;L&lt;/b&gt; (#3)", "2 / 4")
	})
	t.Run("sidebar contests: register link per user kind", func(t *testing.T) {
		c := &kilonova.Contest{ID: 7, Name: "C", StartTime: time.Now().Add(time.Hour), EndTime: time.Now().Add(2 * time.Hour)}
		got := uitest.Render(t, Index(ctx, Params{ContestSidebar: true, FutureContests: []contests.Info{{Contest: c, CanJoin: true, ReqPath: "/"}}}))
		uitest.Contains(t, got, `hidden lg:block`, `href="/login?back=%2F"`, `type="before_start"`)
		got = uitest.Render(t, Index(ctx, Params{ContestSidebar: true, FutureContests: []contests.Info{{Contest: c, CanJoin: true, Authed: true}}}))
		uitest.Contains(t, got, "bundled.registerForContest(7)")
		got = uitest.Render(t, Index(ctx, Params{ContestSidebar: true, FutureContests: []contests.Info{{Contest: c, Authed: true, Registration: &kilonova.ContestRegistration{}}}}))
		uitest.Contains(t, got, "- <span>Registered</span>")
	})
}
