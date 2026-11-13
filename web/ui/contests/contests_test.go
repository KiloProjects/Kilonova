package contests

import (
	"context"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
)

func TestBrief(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	future := &kilonova.Contest{ID: 2, Name: "<c>", Type: "official", StartTime: time.Now().Add(time.Hour), EndTime: time.Now().Add(3 * time.Hour)}

	t.Run("anonymous gets a login link back to the page", func(t *testing.T) {
		got := uitest.Render(t, Brief(ctx, Info{Contest: future, CanJoin: true, ReqPath: "/contests"}))
		uitest.Contains(t, got, `<a href="/contests/2">&lt;c&gt;</a>`, `href="/login?back=%2Fcontests"`, `type="before_start"`, "duration: 2h0m0s")
		uitest.NotContains(t, got, "badge-lite", "contest_type")
	})
	t.Run("authed unregistered gets the register button", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Brief(ctx, Info{Contest: future, CanJoin: true, Authed: true})), `onclick="bundled.registerForContest(2)"`)
	})
	t.Run("editor sees badges", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Brief(ctx, Info{Contest: future, IsEditor: true})), "bg-red-700")
	})
}

func TestSidebar(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	start := time.Now().Add(-time.Hour)
	usaco := &kilonova.Contest{ID: 2, Name: "c", Type: "official", PerUserTime: 3600, StartTime: start, EndTime: time.Now().Add(time.Hour),
		Editors: []*kilonova.UserBrief{{Name: "ed"}}}
	score := decimal.NewFromInt(40)
	pbs := []*kilonova.ScoredProblem{
		{Problem: kilonova.Problem{ID: 9, Name: "P", ScoringStrategy: kilonova.ScoringTypeSumSubtasks}, ScoreUserID: new(1), MaxScore: &score},
		{Problem: kilonova.Problem{ID: 10, Name: "Q"}},
	}

	t.Run("registered user can start a USACO contest", func(t *testing.T) {
		got := uitest.Render(t, Sidebar(ctx, SidebarParams{Info: Info{Contest: usaco, CanJoin: true, Authed: true, Registration: &kilonova.ContestRegistration{}}}))
		uitest.Contains(t, got, `onclick="bundled.startContestRegistration(2)"`, `href="/profile/ed"`, "1h0m0s")
	})
	t.Run("problems list highlights the current one and links the breakdown", func(t *testing.T) {
		got := uitest.Render(t, Sidebar(ctx, SidebarParams{Info: Info{Contest: usaco}, Problems: pbs, CurrentProblem: &pbs[0].Problem}))
		uitest.Contains(t, got, `class="topbar-selected p-0" href="/contests/2/problems/9"`,
			`P</a> (<a class="max_score_breakdown" href="#" data-problemid="9" data-contestid="2">40p</a>)`,
			`<a href="/contests/2/problems/10">Q</a></li>`)
		uitest.NotContains(t, got, "/profile/ed")
	})
}
