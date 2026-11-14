package problempages

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type ArchiveParams struct {
	Topbar   *problems.TopbarParams
	Tests    []*kilonova.Test
	Settings *kilonova.ProblemEvalSettings // may be nil
	// CaptchaID is set when the user must solve a captcha before downloading.
	CaptchaID string
}

// Archive lets the user download the problem archive, choosing what goes in it,
// and lists individual test downloads for users who can view tests.
func Archive(ctx context.Context, p ArchiveParams) Node {
	t := p.Topbar
	pbID := t.Problem.ID
	return Group{
		problems.Topbar(ctx, t),
		Form(ID("problemArchiveForm"), Class("segment-panel"), Method("GET"), Action(fmt.Sprintf("/assets/problem/%d/problemArchive", pbID)),
			H2(ui.T(ctx, "gen_title")),
			If(t.CanViewTests, Div(Class("block mb-2"),
				archiveCheckboxLabel(ctx, "tests", "tests", true),
				If(p.Settings != nil && !t.IsProblemEditor && p.Settings.CheckerName != "", P(Class("text-sm text-muted"), ui.T(ctx, "gen_checker_warning"))),
			)),
			archiveCheckbox(ctx, "attachments", "attachments", true),
			If(t.IsProblemEditor, archiveCheckbox(ctx, "private_attachments", "privateAttachments", true)),
			archiveCheckbox(ctx, "details", "gen_details", true),
			archiveCheckbox(ctx, "tags", "gen_tags", true),
			archiveCheckbox(ctx, "editors", "gen_editors", true),
			archiveCheckbox(ctx, "submissions", "gen_submissions", true),
			If(t.IsProblemEditor, archiveCheckbox(ctx, "all_submissions", "gen_all_submissions", false)),
			Iff(p.CaptchaID != "", func() Node { return ui.Captcha(ctx, p.CaptchaID) }),
			Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "download")),
		),
		If(len(p.Tests) > 0, Div(Class("segment-panel"),
			H2(ui.T(ctx, "individual_tests")),
			Table(Class("kn-table kn-table-slim"),
				THead(Tr(
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "id")),
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "input")),
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "output")),
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "score")),
				)),
				TBody(Map(p.Tests, func(test *kilonova.Test) Node {
					url := fmt.Sprintf("/assets/problem/%d/test/%d/", pbID, test.ID)
					return Tr(Class("kn-table-row"),
						Th(Class("kn-table-cell"), Scope("row"), Textf("%d.", test.VisibleID)),
						Td(Class("kn-table-cell"), A(Class("btn btn-blue mr-2"), Href(url+"input"), Download(""), Text("Download input"))),
						Td(Class("kn-table-cell"), A(Class("btn btn-blue"), Href(url+"output"), Download(""), Text("Download output"))),
						Td(Class("kn-table-cell"), Text(test.Score.String())),
					)
				})),
			),
		)),
	}
}

// archiveCheckbox is one archive option; name is its query parameter, left out when unchecked.
func archiveCheckbox(ctx context.Context, name, key string, checked bool) Node {
	return Div(Class("block mb-2"), archiveCheckboxLabel(ctx, name, key, checked))
}

func archiveCheckboxLabel(ctx context.Context, name, key string, checked bool) Node {
	return Label(Class("inline-flex items-center text-lg"),
		Input(Class("form-checkbox"), Name(name), Value("true"), Type("checkbox"), If(checked, Checked())),
		Span(Class("ml-2"), ui.T(ctx, key)),
	)
}
