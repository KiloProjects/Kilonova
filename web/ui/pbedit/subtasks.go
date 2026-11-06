package pbedit

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SubtasksParams struct {
	Problem  *kilonova.Problem
	Topbar   *problems.TopbarParams
	Subtasks []*kilonova.SubTask // ordered by visible ID
	Tests    []*kilonova.Test

	SubTask *kilonova.SubTask // SubtaskEdit only
}

func subtaskSidebar(ctx context.Context, p SubtasksParams) Node {
	items := make([]sidebarItem, 0, len(p.Subtasks))
	for _, st := range p.Subtasks {
		items = append(items, sidebarItem{st.ID, st.VisibleID})
	}
	return sidebar(ctx, p.Topbar, sidebarParams{
		page: "subtasks", path: "subtasks", panelClass: "segment-panel reset-list",
		headKey: "subTasks", addKey: "createSubTask", nthKey: "nthSubTask",
		items: items,
	})
}

// testSubtasks lists the visible IDs of the subtasks containing test id, or "-".
func testSubtasks(sts []*kilonova.SubTask, id int) string {
	var vids []string
	for _, st := range sts {
		if slices.Contains(st.Tests, id) {
			vids = append(vids, strconv.Itoa(st.VisibleID))
		}
	}
	if len(vids) == 0 {
		return "-"
	}
	return strings.Join(vids, ", ")
}

// SubtaskIndex lists the subtasks with editable scores and each test's subtasks.
func SubtaskIndex(ctx context.Context, p SubtasksParams) Node {
	pb := p.Problem
	return Group{page(ctx, p.Topbar,
		subtaskSidebar(ctx, p),
		Div(Class("page-content-wrapper"),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "header.edit.subtask")),
				If(len(p.Subtasks) > 0, Group{
					Table(Class("kn-table my-2"), Style("table-layout: fixed"),
						THead(
							Th(Scope("col"), Class("w-1/4"), Input(Class("form-checkbox"), Type("checkbox"), ID("selectAllBox"), AutoComplete("off"))),
							Th(Scope("col"), Class("w-1/4 text-center px-4 py-2"), ui.T(ctx, "id")),
							Th(Scope("col"), Class("w-1/2"), ui.T(ctx, "score")),
						),
						TBody(
							Map(p.Subtasks, func(st *kilonova.SubTask) Node {
								id := strconv.Itoa(st.ID)
								return Tr(ID("row-subtask-"+id), Class("kn-table-row"),
									Td(Class("kn-table-cell"), Input(Class("form-checkbox"), Type("checkbox"), ID("pb-subtask-"+id), AutoComplete("off"))),
									Td(Class("kn-table-cell"), Text(strconv.Itoa(st.VisibleID))),
									Td(Class("kn-table-cell"), Input(Class("form-input"), Type("number"), ID("score-subtask-"+id), Value(st.Score.String()),
										Min("0"), Max("100"), Step(scoreStep(pb)), AutoComplete("off"))),
								)
							}),
							Tr(Class("kn-table-simple"),
								Td(ColSpan("2"), Class("kn-table-cell")),
								Td(ID("scoreOutput"), Class("kn-table-cell")),
							),
						),
					),
					Button(Class("btn btn-red mr-2"), Attr("onclick", "deleteSubTasks()"), ui.T(ctx, "deleteSubTasks")),
					Text(" "),
					Button(Class("btn btn-blue"), Attr("onclick", "updateSubTasks()"), ui.T(ctx, "updateSubTasks")),
				}),
				If(len(p.Subtasks) == 0, H3(ui.T(ctx, "noSubTasks"))),
			),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "testAssociations")),
				If(len(p.Tests) > 0, Table(Class("kn-table my-2"), Style("table-layout: fixed"),
					THead(
						Th(Scope("col"), Class("w-1/2 text-center px-4 py-2"), ui.T(ctx, "id")),
						Th(Scope("col"), Class("w-1/2"), ui.T(ctx, "subTasks")),
					),
					TBody(Map(p.Tests, func(t *kilonova.Test) Node {
						return Tr(Class("kn-table-row"),
							Td(Class("kn-table-cell"), Text(strconv.Itoa(t.VisibleID))),
							Td(Class("kn-table-cell"), Text(testSubtasks(p.Subtasks, t.ID))),
						)
					})),
				)),
				If(len(p.Tests) == 0, H3(ui.T(ctx, "noTestsError"))),
			),
		),
	), script("subtask_index.js", map[string]any{
		"PBID":           pb.ID,
		"PRECISION":      pb.ScorePrecision,
		"DEFAULT_POINTS": pb.DefaultPoints,
	})}
}

// subtaskForm is the shared ID/score/test picker of the add and edit pages. st is nil when adding.
func subtaskForm(ctx context.Context, p SubtasksParams, st *kilonova.SubTask, submit Node) Node {
	if len(p.Tests) == 0 {
		return H3(ui.T(ctx, "noTestsError"))
	}
	vid, score, extra := []Node{}, []Node{Value("0")}, []Node{}
	headCell := Th
	if st != nil {
		vid = []Node{Value(strconv.Itoa(st.VisibleID)), AutoComplete("off")}
		score = []Node{Value(st.Score.String())}
		extra = []Node{AutoComplete("off")}
		headCell = Td // the edit page's template used <td> here
	}
	return Group{
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "id"), Text(": ")),
			Input(Class("form-input"), ID("subtask-id"), Type("number"), Min("0"), Group(vid), Required()),
		),
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "score"), Text(": ")),
			Input(Class("form-input"), ID("subtask-score"), Type("number"), Min("0"), Max("100"), Group(score), Step(scoreStep(p.Problem)), Group(extra), Required()),
		),
		Table(Class("kn-table my-2"), Style("table-layout: fixed"),
			THead(
				Th(Scope("col"), Class("w-1/2"), Input(Class("form-checkbox"), Type("checkbox"), ID("selectAllBox"))),
				headCell(Scope("col"), Class("w-1/2 text-center px-4 py-2"), ui.T(ctx, "testID")),
			),
			TBody(Map(p.Tests, func(t *kilonova.Test) Node {
				id := strconv.Itoa(t.ID)
				return Tr(ID("row-test-"+id), Class("kn-table-row"),
					Td(Class("kn-table-cell"), Input(Class("form-checkbox"), Type("checkbox"), ID("pb-test-"+id), AutoComplete("off"),
						If(st != nil && slices.Contains(st.Tests, t.ID), Checked()))),
					Td(Class("kn-table-cell"), Text(strconv.Itoa(t.VisibleID))),
				)
			})),
		),
		submit,
	}
}

// SubtaskAdd is the form for creating a subtask.
func SubtaskAdd(ctx context.Context, p SubtasksParams) Node {
	return Group{page(ctx, p.Topbar,
		subtaskSidebar(ctx, p),
		Div(Class("page-content"),
			H2(ui.T(ctx, "header.edit.subtask_add")),
			Form(ID("subtask-create"), AutoComplete("off"),
				subtaskForm(ctx, p, nil, Button(Class("btn btn-blue block my-2"), ui.T(ctx, "create"))),
			),
		),
	), script("subtask_add.js", map[string]any{"PBID": p.Problem.ID})}
}

// SubtaskEdit edits a subtask's ID, score and tests.
func SubtaskEdit(ctx context.Context, p SubtasksParams) Node {
	st := p.SubTask
	return Group{page(ctx, p.Topbar,
		subtaskSidebar(ctx, p),
		Div(Class("page-content"),
			Form(ID("subtask-update"),
				H2(ui.T(ctx, "header.edit.subtask_edit", st.VisibleID)),
				If(len(st.Tests) == 0, Group{Text(" "), ui.T(ctx, "warning.emptySubTask"), Text(" ")}),
				subtaskForm(ctx, p, st, Button(Class("btn btn-blue block my-2"), Type("submit"), ui.T(ctx, "button.update"))),
			),
		),
	), script("subtask_edit.js", map[string]any{
		"PBID":       p.Problem.ID,
		"URL_PREFIX": p.Topbar.URLPrefix,
		"SUBTASK_ID": st.ID,
	})}
}
