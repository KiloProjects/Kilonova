package problempages

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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
		got := uitest.Render(t, Statement(ctx, p))
		uitest.Contains(t, got, `href="/login?back=%2Fproblems%2F7"`, "<p>statement</p>", "sum.in", "sum.out", "64MB", "0.5s")
		uitest.NotContains(t, got, `id="sendSubForm"`, `id="variant-select"`)

		p.SubmitForm = &SubmitFormParams{Topbar: p.Topbar}
		got = uitest.Render(t, Statement(ctx, p))
		uitest.Contains(t, got, `id="sendSubForm"`)
		uitest.NotContains(t, got, "/login?back=")
	})
	t.Run("console input uses glossary entries", func(t *testing.T) {
		p := statementParams()
		p.Topbar.Problem.ConsoleInput = true
		uitest.Contains(t, uitest.Render(t, Statement(ctx, p)), `<kn-glossary name="stdin" content="stdout"></kn-glossary>`)
	})
	t.Run("variant picker only with several variants, llm warning", func(t *testing.T) {
		p := statementParams()
		a := &kilonova.StatementVariant{Language: "en", Format: "md", Type: "llm"}
		b := &kilonova.StatementVariant{Language: "ro", Format: "pdf"}
		p.Variants = []*kilonova.StatementVariant{a, b}
		p.SelectedVariant = a
		got := uitest.Render(t, Statement(ctx, p))
		uitest.Contains(t, got, `<option value="en-md-llm" selected>`, `<option value="ro-pdf-">`, "<blockquote>")
	})
	t.Run("contest sidebar replaces problem sidebar", func(t *testing.T) {
		p := statementParams()
		p.ParentLists = []ParentList{{List: &kilonova.ProblemList{ID: 3, Title: "List"}}}
		uitest.Contains(t, uitest.Render(t, Statement(ctx, p)), `href="/problem_lists/3"`)
		c := &kilonova.Contest{ID: 9, Name: "Cup"}
		p.Topbar.Contest = c
		p.ContestSidebar = &contests.SidebarParams{Info: contests.Info{Contest: c}}
		uitest.NotContains(t, uitest.Render(t, Statement(ctx, p)), `href="/problem_lists/3"`)
	})
	t.Run("parent list marks current problem and shows score only when loaded", func(t *testing.T) {
		p := statementParams()
		uid, score := 1, decimal.NewFromInt(40)
		p.ParentLists = []ParentList{{List: &kilonova.ProblemList{ID: 3}, Problems: []*kilonova.ScoredProblem{
			{Problem: kilonova.Problem{ID: 7, Name: "a<b"}, ScoreUserID: &uid, MaxScore: &score},
			{Problem: kilonova.Problem{ID: 8, Name: "other"}},
		}}}
		got := uitest.Render(t, Statement(ctx, p))
		uitest.Contains(t, got, `class="topbar-selected p-0" href="/problems/7?list_id=3">a&lt;b</a> (<span data-pbid-reload="7">40p</span>)`, `<a href="/problems/8?list_id=3">other</a></li>`)
	})
	t.Run("external resource links only for editors", func(t *testing.T) {
		p := statementParams()
		p.ShowExternalResources = true
		uitest.NotContains(t, uitest.Render(t, Statement(ctx, p)), "externalResources/create")
		p.Topbar.IsProblemEditor = true
		uitest.Contains(t, uitest.Render(t, Statement(ctx, p)), `href="/problems/7/externalResources/create"`)
	})
	t.Run("scoring strategy is a JS string", func(t *testing.T) {
		p := statementParams()
		p.Topbar.Problem.ScoringStrategy = "</script>"
		got := uitest.Render(t, Statement(ctx, p))
		uitest.Contains(t, got, "let strat = \"\\u003c/script\\u003e\";")
	})
}

func TestStatementDescription(t *testing.T) {
	pb := &kilonova.Problem{Name: "sum", SourceCredits: "ONI"}
	tags := []*kilonova.Tag{{Name: "Ana", Type: kilonova.TagTypeAuthor}, {Name: "dp", Type: kilonova.TagTypeMethod}, {Name: "Bob", Type: kilonova.TagTypeAuthor}}
	require.Equal(t, "sum - propusă la ONI - de către Ana, Bob", StatementDescription(pb, tags))
	require.Equal(t, "x", StatementDescription(&kilonova.Problem{Name: "x"}, nil))
}

func TestFormatStmtVariant(t *testing.T) {
	require.Equal(t, "🇬🇧 English - Custom - LaTeX", ui.StatementVariantName(ctx, &kilonova.StatementVariant{Language: "en", Format: "tex", Type: "custom"}))
}

func TestSubmitForm(t *testing.T) {
	cs := []*kilonova.Contest{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	t.Run("in contest", func(t *testing.T) {
		tb := &problems.TopbarParams{Problem: problem(), Contest: cs[0], CanSubmitInContest: true}
		got := uitest.Render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb}))
		uitest.Contains(t, got, `<input type="hidden" id="sub_contestid" value="1">`, `class="block mb-2 form-label"`, "\nreloadRemainingAttempts()\n")
		tb.CanSubmitInContest = false
		got = uitest.Render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb}))
		uitest.Contains(t, got, `value="-1"`, `form-label hidden`)
	})
	t.Run("running contests", func(t *testing.T) {
		tb := &problems.TopbarParams{Problem: problem(), FullyVisible: true}
		got := uitest.Render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb, RunningContests: cs}))
		uitest.Contains(t, got, `<option value="-1" selected>-</option>`, `form-label hidden`)
		tb.FullyVisible = false
		got = uitest.Render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb, RunningContests: cs}))
		uitest.Contains(t, got, `<option value="" disabled selected>`)
		uitest.NotContains(t, got, "form-label hidden")
		got = uitest.Render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb, RunningContests: cs[:1]}))
		uitest.Contains(t, got, `<option value="1" selected>A</option>`)
	})
	t.Run("page name is a JS string", func(t *testing.T) {
		tb := &problems.TopbarParams{Problem: problem(), Page: "pb_submit"}
		uitest.Contains(t, uitest.Render(t, SubmitForm(ctx, SubmitFormParams{Topbar: tb})), `const problemID = 7;`, `const pageName = "pb_submit";`)
	})
}

func TestSubmitPage(t *testing.T) {
	tb := &problems.TopbarParams{Problem: problem()}
	uitest.Contains(t, uitest.Render(t, SubmitPage(ctx, SubmitPageParams{Form: SubmitFormParams{Topbar: tb}})), `class="page-content-full-wrapper min-h-0"`)
	tb.Contest = &kilonova.Contest{ID: 1}
	got := uitest.Render(t, SubmitPage(ctx, SubmitPageParams{Form: SubmitFormParams{Topbar: tb}, ContestSidebar: &contests.SidebarParams{Info: contests.Info{Contest: tb.Contest}}}))
	uitest.Contains(t, got, `class="page-content-wrapper min-h-0"`, `class="page-sidebar min-h-0"`)
}

func TestStatistics(t *testing.T) {
	subs := &sudoapi.Submissions{
		Submissions: []*kilonova.Submission{{ID: 5, UserID: 1, MaxTime: 0.25, MaxMemory: -1}, {ID: 6, UserID: 2, MaxTime: -1}},
		Users:       map[int]*kilonova.UserBrief{1: {ID: 1, Name: "alice"}},
	}
	stats := &sudoapi.ProblemStatistics{TimeLeaderboard: subs, MemoryLeaderboard: &sudoapi.Submissions{}, SizeLeaderboard: subs}
	p := StatisticsParams{Topbar: &problems.TopbarParams{Problem: problem()}, Stats: stats}

	page := uitest.Render(t, Statistics(ctx, p))
	uitest.Contains(t, page, `<h1>`, `<a href="/submissions/5">250ms</a>`, `<a href="/profile/alice">alice</a>`, "????", `<td class="kn-table-cell">2.</td>`)
	uitest.NotContains(t, page, "hx-swap-oob", "memory", "codeSize") // empty memory board, zero-size best entry

	modal := uitest.Render(t, StatisticsModal(ctx, p))
	uitest.Contains(t, modal, `<div hx-swap-oob="beforeend:#modals"><dialog class="modal-container modal-container-large">`, `<code>&lt;sum&gt;</code>`, `<div class="modal-footer"></div>`, "<h2>")
}

func TestArchive(t *testing.T) {
	tb := &problems.TopbarParams{Problem: problem(), CanViewTests: true}
	p := ArchiveParams{Topbar: tb, Settings: &kilonova.ProblemEvalSettings{CheckerName: "checker.cpp"}, Tests: []*kilonova.Test{{ID: 11, VisibleID: 1}}}
	got := uitest.Render(t, Archive(ctx, p))
	uitest.Contains(t, got, `<input class="hidden" id="aAllSubs" type="checkbox">`, "text-sm text-muted", `href="/assets/problem/7/test/11/input" download`, "function getCaptchaData() { return {} }", "var problemID = 7")
	uitest.NotContains(t, got, "captcha_img")

	tb.IsProblemEditor, tb.CanViewTests = true, false
	p.CaptchaID = "abc"
	got = uitest.Render(t, Archive(ctx, p))
	uitest.Contains(t, got, `<input class="form-checkbox" id="aAllSubs" type="checkbox">`, `<input class="hidden" id="aTests" type="checkbox">`, `src="/api/auth/captcha/abc.png"`)
	uitest.NotContains(t, got, "text-sm text-muted")
}
