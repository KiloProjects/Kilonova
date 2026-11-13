package problems

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func TestTag(t *testing.T) {
	t.Run("known type gets its colour", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Tag(&kilonova.Tag{ID: 1, Name: "dp", Type: kilonova.TagTypeMethod}, true, "text-sm")), `href="/tags/1"`, `class="mx-1 tag tag-method text-sm"`)
	})
	t.Run("unknown type falls back to teal", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Tag(&kilonova.Tag{ID: 2, Name: "x", Type: "weird"}, false)), `class="bg-teal-700 tag"`)
	})
}

func TestProblemListGroup(t *testing.T) {
	base := context.WithValue(context.Background(), util.LangKey, "en")
	authed := context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 2})
	pbs := []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 1, Name: "A", Visible: false}, IsEditor: true}}

	t.Run("links into the contest and list", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, ProblemListGroup(base, &ProblemListingParams{Problems: pbs, ContestIDScore: new(4), ListID: new(9)})), `href="/contests/4/problems/1?list_id=9"`)
	})
	t.Run("publish badge only for authed editors", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, ProblemListGroup(authed, &ProblemListingParams{Problems: pbs, ShowPublished: true})), "badge-red")
		uitest.NotContains(t, uitest.Render(t, ProblemListGroup(base, &ProblemListingParams{Problems: pbs, ShowPublished: true})), "badge-red")
	})
}
