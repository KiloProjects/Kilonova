package submissions

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
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
		must(t, render(t, Submission(ctx, SubmissionPageParams{Submission: fullSub(kilonova.StatusWorking), LanguageFormatter: lang})), `hx-trigger="kn-poll, every 1s"`)
		must(t, render(t, Submission(ctx, SubmissionPageParams{Submission: fullSub(kilonova.StatusFinished), LanguageFormatter: lang})), `hx-trigger="kn-poll"`)
	})
	t.Run("code is escaped by the highlighter", func(t *testing.T) {
		sub := fullSub(kilonova.StatusFinished)
		sub.CodeTrulyVisible = true
		got := render(t, Submission(ctx, SubmissionPageParams{Submission: sub, LanguageFormatter: lang,
			SubmissionFiles: []*kilonova.SubmissionFile{{Filename: "a.cpp", Data: []byte("</pre><script>x</script>")}}}))
		mustNot(t, got, "<script>x")
	})
}

func TestPaste(t *testing.T) {
	got := render(t, Paste(ctx, PastePageParams{
		SubmissionPageParams: SubmissionPageParams{Submission: fullSub(kilonova.StatusFinished), LanguageFormatter: lang},
		Paste:                &kilonova.SubmissionPaste{ID: "abc", Author: &kilonova.UserBrief{Name: "alice"}},
	}))
	must(t, got, `<a href="/profile/alice">alice</a>`, `<a href="/submissions/5">#5</a>`)
}

func TestSubsViewer(t *testing.T) {
	pid := 8
	subs := &sudoapi.Submissions{
		Submissions: []*kilonova.Submission{{ID: 1, ProblemID: 8, Status: kilonova.StatusFinished, SubmissionType: kilonova.EvalTypeClassic, Score: decimal.NewFromInt(100)}},
		Count:       1,
		Problems:    map[int]*kilonova.Problem{8: {ID: 8, Name: "Sum"}, 9: {ID: 9, Name: "Other"}},
	}
	t.Run("problem filter names only that problem", func(t *testing.T) {
		got := render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, Query: SubsViewerQuery{ProblemID: &pid}}))
		must(t, got, `href="/problems/8">Sum</a>`)
		mustNot(t, got, "Other", "break")
	})
	t.Run("sort links flip direction on the active column", func(t *testing.T) {
		got := render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, Query: SubsViewerQuery{Ordering: "max_time"}}))
		must(t, got, `href="?ascending=true&amp;ordering=max_time&amp;page=1"`, "fa-sort-down", `background-color: #7fff00;`)
	})
	t.Run("export link only for admins", func(t *testing.T) {
		mustNot(t, render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs})), "/assets/submissions/export")
		must(t, render(t, SubsViewer(ctx, SubsViewerParams{Submissions: subs, IsAdmin: true})), "/assets/submissions/export")
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
	for _, b := range bad {
		if strings.Contains(got, b) {
			t.Fatalf("unexpected %q in %s", b, got)
		}
	}
}
