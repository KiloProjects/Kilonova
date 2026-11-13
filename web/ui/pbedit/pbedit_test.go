package pbedit

import (
	"context"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func adminCtx() context.Context {
	return context.WithValue(ctx, user.AuthedUserKey, &kilonova.UserBrief{ID: 1, Admin: true})
}

func problem() *kilonova.Problem {
	return &kilonova.Problem{ID: 7, Name: `</script><b>x</b>`, ScorePrecision: 2, DefaultPoints: decimal.NewFromInt(10)}
}

func topbar(pb *kilonova.Problem, page string, id int) *problems.TopbarParams {
	return &problems.TopbarParams{Problem: pb, URLPrefix: "/contests/3", Page: page, PageID: id}
}

func TestGeneral(t *testing.T) {
	pb := problem()
	params := func() GeneralParams {
		return GeneralParams{Problem: pb, Topbar: topbar(pb, "general", -1), Checklist: &kilonova.ProblemChecklist{NumTests: 3}}
	}

	t.Run("escapes the problem name in attributes and the script", func(t *testing.T) {
		got := uitest.Render(t, General(ctx, params()))
		uitest.Contains(t, got, `value="&lt;/script&gt;&lt;b&gt;x&lt;/b&gt;"`, `let problem = {"id":7,`, `</script>`, `step="0.01"`, `"default_points":10`)
		uitest.NotContains(t, got, "<b>x", "__PROBLEM__")
	})
	t.Run("checklist reflects counts and diagnostics", func(t *testing.T) {
		p := params()
		p.Diagnostics = []string{"WARN: <i>bad</i>"}
		got := uitest.Render(t, General(ctx, p))
		uitest.Contains(t, got, `<span class="text-lg">✅</span> Some tests (3 currently);`, "WARN: &lt;i&gt;bad", `<span class="text-lg">❌</span> `)
		uitest.NotContains(t, got, "<i>bad")
	})
	t.Run("visibility is editable only by admins", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, General(adminCtx(), params())), `id="visibility_check" disabled`)
		uitest.Contains(t, uitest.Render(t, General(ctx, params())), `id="visibility_check" disabled`)
	})
	t.Run("review request disables the button and shows when", func(t *testing.T) {
		p := params()
		uitest.Contains(t, uitest.Render(t, General(ctx, p)), `class="block my-2 btn btn-blue" id="review_request_btn">`)
		at := time.UnixMilli(1234)
		pb2 := *pb
		pb2.ReviewRequestedAt = &at
		p.Problem = &pb2
		uitest.Contains(t, uitest.Render(t, General(ctx, p)), `id="review_request_btn" disabled`, `<server-timestamp timestamp="1234">`)
	})
	t.Run("optional sidebar sections", func(t *testing.T) {
		got := uitest.Render(t, General(ctx, params()))
		uitest.NotContains(t, got, "translateStatementForm", "Limbaje permise")
		p := params()
		p.LLMEnabled = true
		p.Settings = &kilonova.ProblemEvalSettings{LanguageWhitelist: []string{"cpp", "py"}, CheckerName: "checker.cpp"}
		uitest.Contains(t, uitest.Render(t, General(ctx, p)), "translateStatementForm", "Limbaje permise: [cpp, py]", "Custom (este executat checker.cpp)", "Fișiere grader: N/A")
	})
}

func TestAccess(t *testing.T) {
	pb := problem()
	got := uitest.Render(t, Access(ctx, pb, topbar(pb, "access", -1)))
	uitest.Contains(t, got, `id="editors_list"`, `const problem_id = 7;`)
}

func TestTests(t *testing.T) {
	pb := problem()
	tests := []*kilonova.Test{{ID: 40, VisibleID: 1, Score: decimal.NewFromInt(45)}, {ID: 41, VisibleID: 2}}

	t.Run("scores page with tests and selected sidebar head", func(t *testing.T) {
		got := uitest.Render(t, TestScores(ctx, TestsParams{Problem: pb, Topbar: topbar(pb, "tests", -2), Tests: tests}))
		uitest.Contains(t, got,
			`class="list-group-head col-span-full list-group-selected" href="/contests/3/problems/7/edit/test"`,
			`href="/contests/3/problems/7/edit/test/41"`,
			`id="score-test-40" value="45"`,
			`var precision = 2`, `var defaultPoints = 10;`, `title: "No files`,
			`"/problem/7/update/processTestArchive"`)
		uitest.NotContains(t, got, "__")
	})
	t.Run("scores page hides the table without tests", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, TestScores(ctx, TestsParams{Problem: pb, Topbar: topbar(pb, "tests", -2)})), `id="selectAllBox"`)
	})
	t.Run("add page", func(t *testing.T) {
		got := uitest.Render(t, TestAdd(ctx, TestsParams{Problem: pb, Topbar: topbar(pb, "tests", -1), Tests: tests, NextVID: 3}))
		uitest.Contains(t, got, `id="vID" type="number" class="form-input" value="3"`, `class="list-group-item col-span-full list-group-selected"`, `"/problem/7/update/addTest"`)
	})
	t.Run("edit page compares against the original only when truncated", func(t *testing.T) {
		p := TestsParams{Problem: pb, Topbar: topbar(pb, "tests", 40), Tests: tests, Test: tests[0],
			Data: TestData{In: "1 2</textarea>", OkIn: true, Out: "too large", OkOut: false}}
		got := uitest.Render(t, TestEdit(ctx, p))
		uitest.Contains(t, got, `class="list-group-item list-group-selected" href="/contests/3/problems/7/edit/test/40"`,
			"1 2&lt;/textarea&gt;", `if(true || inputVal !== null)`, `if(false || outputVal !== "too large")`,
			`const pref = "/contests/3";`, `"/problem/7/update/test/1/info"`, `"/problem/7/update/test/40/data"`, `href="/assets/problem/7/test/40/input" download`)
		uitest.NotContains(t, got, "__")
	})
}

func TestSubtasks(t *testing.T) {
	pb := problem()
	tests := []*kilonova.Test{{ID: 40, VisibleID: 1}, {ID: 41, VisibleID: 2}}
	sts := []*kilonova.SubTask{{ID: 5, VisibleID: 1, Tests: []int{40, 41}}, {ID: 6, VisibleID: 2, Tests: []int{40}}}

	t.Run("index lists test associations", func(t *testing.T) {
		got := uitest.Render(t, SubtaskIndex(ctx, SubtasksParams{Problem: pb, Topbar: topbar(pb, "subtasks", -2), Subtasks: sts, Tests: tests}))
		uitest.Contains(t, got, `<td class="kn-table-cell">1, 2</td>`, `<td class="kn-table-cell">1</td>`, `id="row-subtask-6"`, `href="/contests/3/problems/7/edit/subtasks/6"`)
		uitest.NotContains(t, got, "__")
	})
	t.Run("index without subtasks or tests", func(t *testing.T) {
		got := uitest.Render(t, SubtaskIndex(ctx, SubtasksParams{Problem: pb, Topbar: topbar(pb, "subtasks", -2)}))
		uitest.NotContains(t, got, `id="row-subtask-`, "kn-table-row")
	})
	t.Run("add page needs tests", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, SubtaskAdd(ctx, SubtasksParams{Problem: pb, Topbar: topbar(pb, "subtasks", -1), Subtasks: sts})), `id="subtask-score"`)
		uitest.Contains(t, uitest.Render(t, SubtaskAdd(ctx, SubtasksParams{Problem: pb, Topbar: topbar(pb, "subtasks", -1), Tests: tests})), `id="subtask-score"`, `id="pb-test-41" autocomplete="off">`)
	})
	t.Run("edit page checks the subtask's tests", func(t *testing.T) {
		got := uitest.Render(t, SubtaskEdit(ctx, SubtasksParams{Problem: pb, Topbar: topbar(pb, "subtasks", 6), Subtasks: sts, Tests: tests, SubTask: sts[1]}))
		uitest.Contains(t, got, `id="pb-test-40" autocomplete="off" checked`, `id="pb-test-41" autocomplete="off">`, `subtask_id: 6,`, `value="2"`)
		empty := &kilonova.SubTask{ID: 9, VisibleID: 3}
		got = uitest.Render(t, SubtaskEdit(ctx, SubtasksParams{Problem: pb, Topbar: topbar(pb, "subtasks", 9), Tests: tests, SubTask: empty}))
		uitest.NotContains(t, got, `" checked`)
	})
}
