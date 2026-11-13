package modals

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestOlderSubmissions(t *testing.T) {
	subs := &sudoapi.Submissions{Submissions: []*kilonova.Submission{
		{ID: 10, Status: kilonova.StatusFinished, Score: decimal.NewFromInt(40)},
		{ID: 11, Status: kilonova.StatusWorking},
		{ID: 12, Status: kilonova.StatusWaiting},
	}}
	pb := &kilonova.Problem{ID: 8}

	t.Run("one icon or score per status", func(t *testing.T) {
		got := uitest.Render(t, OlderSubmissions(ctx, &OlderSubmissionsParams{UserID: 2, Problem: pb, Submissions: subs}))
		uitest.Contains(t, got, ">40<", "fa-cog animate-spin", "fa-clock", `hx-get="/problems/8/submissions/?user_id=2"`)
	})
	t.Run("polls every second only while grading and auto-reloading", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, OlderSubmissions(ctx, &OlderSubmissionsParams{Problem: pb, Submissions: subs, AutoReload: true})), `hx-trigger="kn-poll,every 1s"`)
		uitest.Contains(t, uitest.Render(t, OlderSubmissions(ctx, &OlderSubmissionsParams{Problem: pb, Submissions: subs, AutoReload: true, AllFinished: true})), `hx-trigger="kn-poll"`)
	})
	t.Run("contest submissions link inside the contest", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, OlderSubmissions(ctx, &OlderSubmissionsParams{Problem: pb, Contest: &kilonova.Contest{ID: 3}, Submissions: subs, NumHidden: 25})),
			`href="/contests/3/problems/8/submissions/?user_id=0"`, "See 25 more")
	})
}

func TestProblemSources(t *testing.T) {
	got := uitest.Render(t, ProblemSources(ctx, &kilonova.ProblemList{ID: 5}, []*kilonova.ScoredProblem{
		{Problem: kilonova.Problem{ID: 1, Name: "A", SourceCredits: "OJI"}},
		{Problem: kilonova.Problem{ID: 2, Name: "B"}},
	}))
	uitest.Contains(t, got, `hx-post="/problem_lists/5/updateSources"`, "<code>OJI</code>", "<span>-</span>")
}
