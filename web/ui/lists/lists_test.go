package lists

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestIndex(t *testing.T) {
	pbs := []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 1, Name: "A"}}}
	must(t, render(t, Index(ctx, IndexParams{Others: pbs, Authed: true})), "0 / 1 solved", `href="/problems/1"`)
	mustNot(t, render(t, Index(ctx, IndexParams{Others: pbs})), "float-right badge")
}

func TestProgressIndex(t *testing.T) {
	lists := []*kilonova.ProblemList{{ID: 4, Title: "<i>X</i>", NumProblems: 3}}
	must(t, render(t, ProgressIndex(ctx, ProgressIndexParams{RootList: 1, Lists: lists, Authed: true, NumSolved: map[int]int{4: 2}})),
		`href="/problem_lists/1"`, `href="/problem_lists/4/progress">&lt;i&gt;X&lt;/i&gt;</a> (2/3)`)
	mustNot(t, render(t, ProgressIndex(ctx, ProgressIndexParams{Lists: lists})), "(2/3)")
}

func TestChecklist(t *testing.T) {
	score := decimal.NewFromInt(50)
	uid := 1
	list := &sudoapi.FullProblemList{
		ProblemList: kilonova.ProblemList{ID: 1, Title: "Root", NumProblems: 1}, Root: true, SolvedCount: 0,
		SubLists: []*sudoapi.FullProblemList{
			{ProblemList: kilonova.ProblemList{ID: 2, Title: "Sub", NumProblems: 1}, DepthLevel: 1, SolvedCount: -1,
				Problems: []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 9, Name: "P"}, MaxScore: &score, ScoreUserID: &uid}}},
			{ProblemList: kilonova.ProblemList{ID: 3, Title: "Empty"}, DepthLevel: 1},
		},
	}
	t.Run("rows, spans and scores", func(t *testing.T) {
		got := render(t, Checklist(ctx, ProgressParams{List: list, CheckedUser: &kilonova.UserBrief{Name: "bob"}}))
		must(t, got, `rowspan="1"`, `>Sub</a> (0/1)`, `data-score="50"`, `href="/problems/9?list_id=2" title="Score: 50p"`, `href="/profile/bob"`)
		mustNot(t, got, "Empty")
	})
	t.Run("no counts without a user", func(t *testing.T) {
		mustNot(t, render(t, Checklist(ctx, ProgressParams{List: list})), "(0/1)", "/profile/")
	})
}

func TestView(t *testing.T) {
	l := &kilonova.ProblemList{ID: 5, Title: `"Q"`, List: []int{1, 2}}
	t.Run("visitor sees no forms", func(t *testing.T) {
		got := render(t, View(ctx, ViewParams{List: l, Group: problems.ListGroupParams{List: l}}))
		must(t, got, `href="/problems?deep_list_id=5"`, ": -")
		mustNot(t, got, "sim_create", "pblist-update", "unsolved=true")
	})
	t.Run("editor gets the update form with list values, admin the bulk form", func(t *testing.T) {
		got := render(t, View(ctx, ViewParams{List: l, Group: problems.ListGroupParams{List: l}, Authed: true, CanEdit: true, Author: &kilonova.UserBrief{Name: "al"}}))
		must(t, got, `{list: [1,2]}`, `value="1,2"`, `"/problemList/5/update"`, `value="Simulation - &#34;Q&#34;"`, `href="/profile/al"`)
		mustNot(t, got, "problemBulkVizForm")
		must(t, render(t, View(ctx, ViewParams{List: l, Group: problems.ListGroupParams{List: l}, Authed: true, CanEdit: true, IsAdmin: true})), `"/problemList/5/toggleProblems"`)
	})
}

func TestProblemSearch(t *testing.T) {
	must(t, render(t, ProblemSearch(SearchParams{Count: 3})), `<kn-pb-search enc="bnVsbA==" count="3"`)
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
