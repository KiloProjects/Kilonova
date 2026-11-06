package pbedit

import (
	"context"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type TestsParams struct {
	Problem *kilonova.Problem
	Topbar  *problems.TopbarParams
	Tests   []*kilonova.Test

	NextVID int            // TestAdd only
	Test    *kilonova.Test // TestEdit only
	Data    TestData       // TestEdit only
}

// TestData is a test's input and output, possibly replaced by a message when too large or unreadable.
type TestData struct {
	In, Out     string
	OkIn, OkOut bool
}

func testSidebar(ctx context.Context, p TestsParams) Node {
	items := make([]sidebarItem, 0, len(p.Tests))
	for _, t := range p.Tests {
		items = append(items, sidebarItem{t.ID, t.VisibleID})
	}
	return sidebar(ctx, p.Topbar, sidebarParams{
		page: "tests", path: "test", panelClass: "segment-panel",
		headKey: "tests", addKey: "createTest", nthKey: "nthTest",
		items: items,
	})
}

// TestScores lists the tests with editable scores and the test archive upload form.
func TestScores(ctx context.Context, p TestsParams) Node {
	pb := p.Problem
	return Group{page(ctx, p.Topbar,
		testSidebar(ctx, p),
		Div(Class("page-content-wrapper"),
			If(len(p.Tests) > 0, Div(Class("segment-panel"),
				H2(Text("Actualizare teste")),
				Table(Class("kn-table my-2"),
					THead(
						Th(Scope("col"), Class("w-1/4"), Input(Class("form-checkbox"), Type("checkbox"), ID("selectAllBox"), AutoComplete("off"))),
						Th(Scope("col"), Class("w-1/4 text-center px-4 py-2"), Text("ID")),
						Th(Scope("col"), Class("w-1/2"), ui.T(ctx, "score")),
					),
					TBody(
						Map(p.Tests, func(t *kilonova.Test) Node {
							id := strconv.Itoa(t.ID)
							return Tr(ID("row-test-"+id), Class("kn-table-row"),
								Td(Class("kn-table-cell"), Input(Class("form-checkbox"), Type("checkbox"), ID("pb-test-"+id), AutoComplete("off"))),
								Td(Class("kn-table-cell"), Text(strconv.Itoa(t.VisibleID))),
								Td(Class("kn-table-cell"), Input(Class("form-input"), Type("number"), ID("score-test-"+id), Value(t.Score.String()),
									Min("0"), Max("100"), Step(scoreStep(pb)), AutoComplete("off"))),
							)
						}),
						Tr(Class("kn-table-simple"),
							Td(ColSpan("2"), Class("kn-table-cell")),
							Td(ID("scoreOutput"), Class("kn-table-cell")),
						),
					),
				),
				Div(Class("block mb-2"),
					Button(Class("btn btn-red mr-2"), Attr("onclick", "deleteTests()"), ui.T(ctx, "deleteTests")),
					Text(" "),
					Button(Class("btn btn-blue"), Attr("onclick", "updateTests()"), ui.T(ctx, "updateTestScores")),
				),
			)),
			Form(Class("segment-panel"), ID("test_add_form"),
				H2(ui.T(ctx, "testArchive"), Text(" (200MB max)")),
				Label(Class("block my-2"),
					Span(Class("mr-2 text-xl"), ui.T(ctx, "archive"), Text(":")), Text(" "),
					Input(ID("tests"), Type("file"), Class("form-input"), Accept(".zip"), Required()),
				),
				Label(Class("block my-2"),
					Span(Class("mr-2 text-xl"), ui.T(ctx, "scoreParams"), Text(":")), Text(" "),
					Input(ID("scoreParams"), Type("text"), Class("form-input"), AutoComplete("off")),
				),
				Label(Class("block my-2"),
					Input(ID("forceStringIDs"), Class("form-checkbox"), Type("checkbox")), Text(" "),
					Span(Class("form-label ml-2"), ui.T(ctx, "forceStringIDs")),
				),
				Button(Class("btn btn-blue mb-2"), ui.T(ctx, "button.upload")),
				P(Class("text-muted text-sm"), ui.TRaw(ctx, "polygonArchiveWarn")),
			),
		),
	), script("test_scores.js", map[string]any{
		"PBID":           pb.ID,
		"PRECISION":      pb.ScorePrecision,
		"DEFAULT_POINTS": pb.DefaultPoints,
		"NO_FILES":       ui.TS(ctx, "noFiles"),
	})}
}

// TestAdd is the form for creating a single test from text or files.
func TestAdd(ctx context.Context, p TestsParams) Node {
	pb := p.Problem
	ioPanel := func(typ string) Node {
		return Div(Class("segment-panel"),
			H2(ui.T(ctx, typ)),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "upload_method"), Text(":")), Text(" "),
				Select(ID(typ+"_submit_style"), Class("form-select"), AutoComplete("off"),
					Option(Value("text"), ui.T(ctx, "text_insert")),
					Option(Value("file"), ui.T(ctx, "file")),
				),
			),
			Label(ID(typ+"_textarea_label"), Class("block mb-2"),
				Textarea(Class("form-textarea"), ID(typ+"_textarea"), Rows("15"), Style("width: 80%;")),
			),
			Label(ID(typ+"_file_label"), Class("block mb-2 hidden"),
				Span(Class("form-label"), ui.T(ctx, "upload_file"), Text(":")), Text(" "),
				Input(Class("form-input"), ID(typ+"_file"), Type("file"), AutoComplete("off")),
			),
		)
	}
	return Group{page(ctx, p.Topbar,
		testSidebar(ctx, p),
		Div(Class("page-content"),
			Form(ID("test_create_form"),
				H2(ui.T(ctx, "createTest")),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "id"), Text(": ")),
					Input(ID("vID"), Type("number"), Class("form-input"), Value(strconv.Itoa(p.NextVID)), Required()),
				),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "score"), Text(": ")),
					Input(ID("score"), Type("number"), Class("form-input"), Min("0"), Max("100"), Step(scoreStep(pb)), Value("0"), Required()),
				),
				ioPanel("input"),
				ioPanel("output"),
				Button(Class("btn btn-blue mb-2"), ui.T(ctx, "button.add")),
			),
		),
	), script("test_add.js", map[string]any{"PBID": pb.ID})}
}

// TestEdit edits one test's ID, score and data.
func TestEdit(ctx context.Context, p TestsParams) Node {
	pb, t := p.Problem, p.Test
	assetURL := "/assets/problem/" + strconv.Itoa(pb.ID) + "/test/" + strconv.Itoa(t.ID)
	ioArea := func(key, id, val string) Node {
		return Div(Class("mb-3"),
			H3(ui.T(ctx, key), Text(":")),
			Textarea(Class("form-textarea"), Style("width:80%;"), Rows("15"), ID(id), Text(val)),
		)
	}
	reupload := func(name, key string) Node {
		return Div(Class("block my-2"),
			Input(Class("form-checkbox"), ID(name+"Checkbox"), Type("checkbox")), Text(" "),
			Span(ID(name+"Label"), Class("form-label text-muted"), ui.T(ctx, key), Text(":")), Text(" "),
			Input(Class("form-input"), ID(name), Type("file"), Disabled()),
		)
	}
	// The original value is only compared against when it was replaced by a message.
	var origIn, origOut *string
	if !p.Data.OkIn {
		origIn = &p.Data.In
	}
	if !p.Data.OkOut {
		origOut = &p.Data.Out
	}
	return Group{page(ctx, p.Topbar,
		testSidebar(ctx, p),
		Div(Class("page-content-wrapper"),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "updateTest", t.VisibleID)),
				Form(ID("test_id_edit_form"),
					Label(Class("block my-2"),
						Span(Class("mr-2 text-xl"), Text("ID: ")),
						Input(ID("vID"), Type("number"), Class("form-input"), Value(strconv.Itoa(t.VisibleID)), Min("0"), Required()),
					),
					Label(Class("block my-2"),
						Span(Class("mr-2 text-xl"), ui.T(ctx, "score"), Text(": ")),
						Input(ID("score"), Type("number"), Class("form-input"), Value(t.Score.String()), Min("0"), Max("100"), Step(scoreStep(pb)), Required()),
					),
					Button(Class("btn btn-blue mr-2"), ui.T(ctx, "button.update")),
					Text(" "),
					Button(ID("test_del_button"), Type("button"), Class("btn btn-red"), ui.T(ctx, "button.delete")),
				),
				Div(Class("my-2"),
					A(Class("btn btn-blue mr-2"), Href(assetURL+"/input"), Attr("download"), Text("Download input")),
					Text(" "),
					A(Class("btn btn-blue"), Href(assetURL+"/output"), Attr("download"), Text("Download output")),
				),
				Form(ID("test_edit_form"),
					ioArea("input", "input", p.Data.In),
					ioArea("output", "output", p.Data.Out),
					Button(Class("btn btn-blue mb-2"), ui.T(ctx, "button.update")),
				),
			),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "updateTestFiles")),
				Form(ID("test_reupload_form"), AutoComplete("off"),
					reupload("newInput", "input"),
					reupload("newOutput", "output"),
					Button(Class("btn btn-blue"), ui.T(ctx, "button.update")),
				),
			),
		),
	), script("test_edit.js", map[string]any{
		"URL_PREFIX": p.Topbar.URLPrefix,
		"PBID":       pb.ID,
		"TEST_ID":    t.ID,
		"TEST_VID":   t.VisibleID,
		"OK_IN":      p.Data.OkIn,
		"OK_OUT":     p.Data.OkOut,
		"ORIG_IN":    origIn,
		"ORIG_OUT":   origOut,
	})}
}
