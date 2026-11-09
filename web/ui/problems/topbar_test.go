package problems

import (
	"context"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func TestTopbar(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	pb := &kilonova.Problem{ID: 7, Name: "<x>"}
	running := &kilonova.Contest{ID: 3, Name: "C", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour)}
	ended := &kilonova.Contest{ID: 3, Name: "C", StartTime: time.Now().Add(-2 * time.Hour), EndTime: time.Now().Add(-time.Hour)}

	t.Run("statement page escapes the name and selects its tab", func(t *testing.T) {
		got := render(t, Topbar(ctx, &TopbarParams{Problem: pb, Page: "pb_statement"}))
		must(t, got, "<b>&lt;x&gt;</b>", `<a class="p-1 topbar-selected" href="/problems/7/">`)
		mustNot(t, got, "/submit", "data-tabs-collapsible")
	})
	t.Run("submit tab only when authed", func(t *testing.T) {
		must(t, render(t, Topbar(ctx, &TopbarParams{Problem: pb, Page: "pb_statement", Authed: true})), `href="/problems/7/submit"`)
	})
	t.Run("editor menu is collapsed on viewing pages, open on edit pages", func(t *testing.T) {
		must(t, render(t, Topbar(ctx, &TopbarParams{Problem: pb, Page: "pb_statement", IsProblemEditor: true})),
			`<span data-tabs-panel="" class="hidden">`)
		must(t, render(t, Topbar(ctx, &TopbarParams{Problem: pb, Page: "desc", IsProblemEditor: true})),
			`data-tabs-open`, `<a data-tabs-toggle="" class="hidden p-1"`, `class="p-1 topbar-selected" href="/problems/7/edit/desc"`)
	})
	t.Run("contest problem links under the contest and hides submissions when it can't submit", func(t *testing.T) {
		got := render(t, Topbar(ctx, &TopbarParams{Problem: pb, Contest: running, URLPrefix: "/contests/3", Page: "pb_statement"}))
		must(t, got, "<b>C | &lt;x&gt;</b>", `href="/contests/3/problems/7/"`, "kn-comm-announcer", `data-qna-poll="10000"`)
		mustNot(t, got, "/submissions", "/leaderboard")
	})
	t.Run("contest page has no problem tabs", func(t *testing.T) {
		got := render(t, Topbar(ctx, &TopbarParams{Contest: running, URLPrefix: "/contests/3", Page: "contest_leaderboard", LeaderboardVisible: true, IsContestEditor: true}))
		must(t, got, `href="/contests/3/manage/edit"`, `class="p-1 topbar-selected" href="/contests/3/leaderboard"`, `data-qna-poll="5000"`)
		mustNot(t, got, "/problems/", "topbar-separator-lg")
	})
	t.Run("ended contest problem shows the warning", func(t *testing.T) {
		got := render(t, Topbar(ctx, &TopbarParams{Problem: pb, Contest: ended, URLPrefix: "/contests/3", Page: "pb_statement"}))
		must(t, got, "data-strip-contest", `href="/contests/3/problems/7/submissions"`)
		mustNot(t, got, "kn-comm-announcer", "data-qna-poll")
	})
}

func TestListGroup(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	list := &kilonova.ProblemList{ID: 5, Title: "L", NumProblems: 2, Description: "d",
		SubLists: []*kilonova.ShallowProblemList{{ID: 6, Title: "S", NumProblems: 1}}}

	t.Run("anonymous sees counts, sublists load lazily", func(t *testing.T) {
		got := render(t, ListGroup(ctx, ListGroupParams{List: list, Open: true, DescriptionHTML: "<p>d</p>"}))
		must(t, got, `<details class="list-group-head" open>`, "<p>d</p>", `hx-get="/problem_lists/6"`, `hx-trigger="toggle once"`)
	})
	t.Run("authed sees solved counts, -1 on a miss", func(t *testing.T) {
		got := render(t, ListGroup(ctx, ListGroupParams{List: list, Authed: true, NumSolved: map[int]int{5: 1}}))
		must(t, got, "1 / 2", "-1 / 1")
	})
}
