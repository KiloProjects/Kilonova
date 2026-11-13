package problems

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
)

func TestInfoSidebar(t *testing.T) {
	base := context.WithValue(context.Background(), util.LangKey, "en")
	authed := context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 2, Name: "bob"})
	alice, bob := &kilonova.UserBrief{Name: "alice"}, &kilonova.UserBrief{Name: "bob"}
	hundred := decimal.NewFromInt(100)
	pb := func(s kilonova.ScoringType) *kilonova.Problem { return &kilonova.Problem{ID: 7, ScoringStrategy: s} }
	tags := []*kilonova.Tag{{ID: 1, Name: "Ana", Type: kilonova.TagTypeAuthor}, {ID: 2, Name: "dp", Type: kilonova.TagTypeMethod}}

	t.Run("single editor inline, several in a list open for editors", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(""), Editors: []*kilonova.UserBrief{alice}})), `<a href="/profile/alice">alice</a>`)
		uitest.NotContains(t, uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(""), Editors: []*kilonova.UserBrief{alice, bob}})), "<details class=\"reset-list\" open>")
		uitest.Contains(t, uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(""), Editors: []*kilonova.UserBrief{alice, bob}, IsEditor: true})), "<details class=\"reset-list\" open>")
	})
	t.Run("viewers and visibility only for editors", func(t *testing.T) {
		p := &InfoParams{Problem: pb(""), Editors: []*kilonova.UserBrief{alice}, Viewers: []*kilonova.UserBrief{bob}}
		uitest.NotContains(t, uitest.Render(t, InfoSidebar(base, p)), "bg-red-700", `href="/profile/bob"`)
		p.IsEditor = true
		uitest.Contains(t, uitest.Render(t, InfoSidebar(base, p)), "bg-red-700", `href="/profile/bob"`)
	})
	t.Run("tags split by author and start open only when solved", func(t *testing.T) {
		got := uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(""), Tags: tags}))
		uitest.Contains(t, got, "tag-author", "tag-method", `x-data="{&#34;open&#34;:false}"`)
		uitest.Contains(t, uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(""), Tags: tags, MaxScore: &hundred})), `x-data="{&#34;open&#34;:true}"`)
	})
	t.Run("score only for logged-in users, breakdown link for sum-of-subtasks", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(kilonova.ScoringTypeSumSubtasks)})), "data-pbid-reload")
		uitest.Contains(t, uitest.Render(t, InfoSidebar(authed, &InfoParams{Problem: pb(kilonova.ScoringTypeSumSubtasks), MaxScore: &hundred})), `class="max_score_breakdown"`, ">100p<")
		uitest.NotContains(t, uitest.Render(t, InfoSidebar(authed, &InfoParams{Problem: pb(kilonova.ScoringTypeMaxSub)})), "max_score_breakdown")
	})
	t.Run("archive link is prefixed inside contests", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, InfoSidebar(base, &InfoParams{Problem: pb(""), CanViewTests: true, Contest: &kilonova.Contest{ID: 4}})), `href="/contests/4/problems/7/archive"`)
	})
}
