package problempages

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func problem() *kilonova.Problem {
	return &kilonova.Problem{ID: 7, Name: "<sum>", TestName: "sum", TimeLimit: 0.5, MemoryLimit: 65536, ScoringStrategy: kilonova.ScoringTypeMaxSub}
}

func statementParams() StatementParams {
	pb := problem()
	return StatementParams{
		Topbar:        &problems.TopbarParams{Problem: pb, Page: "pb_statement", FullyVisible: true},
		Info:          &problems.InfoParams{Problem: pb},
		StatementHTML: "<p>statement</p>",
		ReqPath:       "/problems/7",
	}
}

func TestStatement(t *testing.T) {
	t.Run("visitor gets login prompt, user gets submit form", func(t *testing.T) {
		p := statementParams()
		got := render(t, Statement(ctx, p))
		must(t, got, `href="/login?back=%2Fproblems%2F7"`, "<p>statement</p>", "sum.in", "sum.out", "64MB", "0.5s")
		mustNot(t, got, `id="sendSubForm"`, `id="variant-select"`)

		p.SubmitForm = &SubmitFormParams{Topbar: p.Topbar}
		got = render(t, Statement(ctx, p))
		must(t, got, `id="sendSubForm"`)
		mustNot(t, got, "/login?back=")
	})
	t.Run("console input uses glossary entries", func(t *testing.T) {
		p := statementParams()
		p.Topbar.Problem.ConsoleInput = true
		must(t, render(t, Statement(ctx, p)), `<kn-glossary name="stdin" content="stdout"></kn-glossary>`)
	})
	t.Run("variant picker only with several variants, llm warning", func(t *testing.T) {
		p := statementParams()
		a := &kilonova.StatementVariant{Language: "en", Format: "md", Type: "llm"}
		b := &kilonova.StatementVariant{Language: "ro", Format: "pdf"}
		p.Variants = []*kilonova.StatementVariant{a, b}
		p.SelectedVariant = a
		got := render(t, Statement(ctx, p))
		must(t, got, `<option value="en-md-llm" selected>`, `<option value="ro-pdf-">`, "<blockquote>")
	})
	t.Run("contest sidebar replaces problem sidebar", func(t *testing.T) {
		p := statementParams()
		p.ParentLists = []ParentList{{List: &kilonova.ProblemList{ID: 3, Title: "List"}}}
		must(t, render(t, Statement(ctx, p)), `href="/problem_lists/3"`)
		c := &kilonova.Contest{ID: 9, Name: "Cup"}
		p.Topbar.Contest = c
		p.ContestSidebar = &contests.SidebarParams{Info: contests.Info{Contest: c}}
		mustNot(t, render(t, Statement(ctx, p)), `href="/problem_lists/3"`)
	})
	t.Run("parent list marks current problem and shows score only when loaded", func(t *testing.T) {
		p := statementParams()
		uid, score := 1, decimal.NewFromInt(40)
		p.ParentLists = []ParentList{{List: &kilonova.ProblemList{ID: 3}, Problems: []*kilonova.ScoredProblem{
			{Problem: kilonova.Problem{ID: 7, Name: "a<b"}, ScoreUserID: &uid, MaxScore: &score},
			{Problem: kilonova.Problem{ID: 8, Name: "other"}},
		}}}
		got := render(t, Statement(ctx, p))
		must(t, got, `class="topbar-selected p-0" href="/problems/7?list_id=3">a&lt;b</a> (<span data-pbid-reload="7">40p</span>)`, `<a href="/problems/8?list_id=3">other</a></li>`)
	})
	t.Run("external resource links only for editors", func(t *testing.T) {
		p := statementParams()
		p.ShowExternalResources = true
		mustNot(t, render(t, Statement(ctx, p)), "externalResources/create")
		p.Topbar.IsProblemEditor = true
		must(t, render(t, Statement(ctx, p)), `href="/problems/7/externalResources/create"`)
	})
	t.Run("scoring strategy is a JS string", func(t *testing.T) {
		p := statementParams()
		p.Topbar.Problem.ScoringStrategy = "</script>"
		got := render(t, Statement(ctx, p))
		must(t, got, "let strat = \"\\u003c/script\\u003e\";")
	})
}

func TestStatementDescription(t *testing.T) {
	pb := &kilonova.Problem{Name: "sum", SourceCredits: "ONI"}
	tags := []*kilonova.Tag{{Name: "Ana", Type: kilonova.TagTypeAuthor}, {Name: "dp", Type: kilonova.TagTypeMethod}, {Name: "Bob", Type: kilonova.TagTypeAuthor}}
	if got := StatementDescription(pb, tags); got != "sum - propusă la ONI - de către Ana, Bob" {
		t.Fatal(got)
	}
	if got := StatementDescription(&kilonova.Problem{Name: "x"}, nil); got != "x" {
		t.Fatal(got)
	}
}

func TestFormatStmtVariant(t *testing.T) {
	if got := ui.StatementVariantName(ctx, &kilonova.StatementVariant{Language: "en", Format: "tex", Type: "custom"}); got != "🇬🇧 English - Custom - LaTeX" {
		t.Fatal(got)
	}
}

func TestSubmitForm(t *testing.T) {
	cs := []*kilonova.Contest{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	t.Run("in contest", func(t *testing.T) {
		tb := &problems.TopbarParams{Problem: problem(), Contest: cs[0], CanSubmitInContest: true}
		got := render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb}))
		must(t, got, `<input type="hidden" id="sub_contestid" value="1">`, `class="block mb-2 form-label"`, "\nreloadRemainingAttempts()\n")
		tb.CanSubmitInContest = false
		got = render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb}))
		must(t, got, `value="-1"`, `form-label hidden`)
	})
	t.Run("running contests", func(t *testing.T) {
		tb := &problems.TopbarParams{Problem: problem(), FullyVisible: true}
		got := render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb, RunningContests: cs}))
		must(t, got, `<option value="-1" selected>-</option>`, `form-label hidden`)
		tb.FullyVisible = false
		got = render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb, RunningContests: cs}))
		must(t, got, `<option value="" disabled selected>`)
		mustNot(t, got, "form-label hidden")
		got = render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb, RunningContests: cs[:1]}))
		must(t, got, `<option value="1" selected>A</option>`)
	})
	t.Run("page name is a JS string", func(t *testing.T) {
		tb := &problems.TopbarParams{Problem: problem(), Page: "pb_submit"}
		must(t, render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb})), `const problemID = 7;`, `const pageName = "pb_submit";`)
	})
}

func TestSubmitPage(t *testing.T) {
	tb := &problems.TopbarParams{Problem: problem()}
	must(t, render(t, SubmitPage(ctx, SubmitPageParams{Form: SubmitFormParams{Topbar: tb}})), `class="page-content-full-wrapper min-h-0"`)
	tb.Contest = &kilonova.Contest{ID: 1}
	got := render(t, SubmitPage(ctx, SubmitPageParams{Form: SubmitFormParams{Topbar: tb}, ContestSidebar: &contests.SidebarParams{Info: contests.Info{Contest: tb.Contest}}}))
	must(t, got, `class="page-content-wrapper min-h-0"`, `class="page-sidebar min-h-0"`)
}

func TestStatistics(t *testing.T) {
	subs := &sudoapi.Submissions{
		Submissions: []*kilonova.Submission{{ID: 5, UserID: 1, MaxTime: 0.25, MaxMemory: -1}, {ID: 6, UserID: 2, MaxTime: -1}},
		Users:       map[int]*kilonova.UserBrief{1: {ID: 1, Name: "alice"}},
	}
	stats := &sudoapi.ProblemStatistics{TimeLeaderboard: subs, MemoryLeaderboard: &sudoapi.Submissions{}, SizeLeaderboard: subs}
	p := StatisticsParams{Topbar: &problems.TopbarParams{Problem: problem()}, Stats: stats}

	page := render(t, Statistics(ctx, p))
	must(t, page, `<h1>`, `<a href="/submissions/5">250ms</a>`, `<a href="/profile/alice">alice</a>`, "????", `<td class="kn-table-cell">2.</td>`)
	mustNot(t, page, "hx-swap-oob", "memory", "codeSize") // empty memory board, zero-size best entry

	modal := render(t, StatisticsModal(ctx, p))
	must(t, modal, `<div hx-swap-oob="beforeend:#modals"><dialog class="modal-container modal-container-large">`, `<code>&lt;sum&gt;</code>`, `<div class="modal-footer"></div>`, "<h2>")
}

func TestArchive(t *testing.T) {
	tb := &problems.TopbarParams{Problem: problem(), CanViewTests: true}
	p := ArchiveParams{Topbar: tb, Settings: &kilonova.ProblemEvalSettings{CheckerName: "checker.cpp"}, Tests: []*kilonova.Test{{ID: 11, VisibleID: 1}}}
	got := render(t, Archive(ctx, p))
	must(t, got, `<input class="hidden" id="aAllSubs" type="checkbox">`, "text-sm text-muted", `href="/assets/problem/7/test/11/input" download`, "function getCaptchaData() { return {} }", "var problemID = 7")
	mustNot(t, got, "captcha_img")

	tb.IsProblemEditor, tb.CanViewTests = true, false
	p.CaptchaID = "abc"
	got = render(t, Archive(ctx, p))
	must(t, got, `<input class="form-checkbox" id="aAllSubs" type="checkbox">`, `<input class="hidden" id="aTests" type="checkbox">`, `src="/api/auth/captcha/abc.png"`)
	mustNot(t, got, "text-sm text-muted")
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
