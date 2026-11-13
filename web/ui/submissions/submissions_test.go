package submissions

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

func fullSub(status kilonova.Status) *kilonova.FullSubmission {
	return &kilonova.FullSubmission{
		Submission: kilonova.Submission{ID: 5, Status: status, SubmissionType: kilonova.EvalTypeClassic},
		Author:     &kilonova.UserBrief{Name: "bob"},
		Problem:    &kilonova.Problem{ID: 8, Name: "Sum"},
	}
}

func lang(s string) string { return s }

func TestSubmission(t *testing.T) {
	t.Run("polls every second until finished", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Submission(ctx, SubmissionPageParams{Submission: fullSub(kilonova.StatusWorking), LanguageFormatter: lang})), `hx-trigger="kn-poll, every 1s"`)
		uitest.Contains(t, uitest.Render(t, Submission(ctx, SubmissionPageParams{Submission: fullSub(kilonova.StatusFinished), LanguageFormatter: lang})), `hx-trigger="kn-poll"`)
	})
	t.Run("code is escaped by the highlighter", func(t *testing.T) {
		sub := fullSub(kilonova.StatusFinished)
		sub.CodeTrulyVisible = true
		got := uitest.Render(t, Submission(ctx, SubmissionPageParams{Submission: sub, LanguageFormatter: lang,
			SubmissionFiles: []*kilonova.SubmissionFile{{Filename: "a.cpp", Data: []byte("</pre><script>x</script>")}}}))
		uitest.NotContains(t, got, "<script>x")
	})
}

func TestPaste(t *testing.T) {
	got := uitest.Render(t, Paste(ctx, PastePageParams{
		SubmissionPageParams: SubmissionPageParams{Submission: fullSub(kilonova.StatusFinished), LanguageFormatter: lang},
		Paste:                &kilonova.SubmissionPaste{ID: "abc", Author: &kilonova.UserBrief{Name: "alice"}},
	}))
	uitest.Contains(t, got, `<a href="/profile/alice">alice</a>`, `<a href="/submissions/5">#5</a>`)
}

func TestSubsViewer(t *testing.T) {
	pid := 8
	subs := &sudoapi.Submissions{
		Submissions: []*kilonova.Submission{{ID: 1, ProblemID: 8, Status: kilonova.StatusFinished, SubmissionType: kilonova.EvalTypeClassic, Score: decimal.NewFromInt(100)}},
		Count:       1,
		Problems:    map[int]*kilonova.Problem{8: {ID: 8, Name: "Sum"}, 9: {ID: 9, Name: "Other"}},
	}
	t.Run("problem filter names only that problem", func(t *testing.T) {
		got := uitest.Render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, Query: SubsViewerQuery{ProblemID: &pid}}))
		uitest.Contains(t, got, `href="/problems/8">Sum</a>`)
		uitest.NotContains(t, got, "Other", "break")
	})
	t.Run("in-place mode swaps only the viewer", func(t *testing.T) {
		got := uitest.Render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, Query: SubsViewerQuery{ProblemID: &pid}, InPlace: true}))
		uitest.Contains(t, got, `id="subs-viewer"`, `data-inplace`, `hx-target="this"`, `hx-select="#subs-viewer"`, `hx-replace-url="true"`, `class="block nostyle" href="?`)
		uitest.Contains(t, got, `hx-boost="true"`)
		uitest.NotContains(t, uitest.Render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, Query: SubsViewerQuery{ProblemID: &pid}})), "data-inplace", `hx-target="this"`)
	})
	t.Run("sort links flip direction on the active column", func(t *testing.T) {
		got := uitest.Render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, Query: SubsViewerQuery{Ordering: "max_time"}}))
		uitest.Contains(t, got, `href="?ascending=true&amp;ordering=max_time&amp;page=1"`, "fa-sort-down", `background-color: #7fff00;`)
	})
	t.Run("export link only for admins", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs})), "/assets/submissions/export")
		uitest.Contains(t, uitest.Render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, IsAdmin: true})), "/assets/submissions/export")
	})
}
