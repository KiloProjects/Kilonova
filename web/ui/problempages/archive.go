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
		Form(ID("problemArchiveForm"), Class("segment-panel"),
			H2(ui.T(ctx, "gen_title")),
			If(t.CanViewTests, Div(Class("block mb-2"),
				archiveCheckboxLabel(ctx, "aTests", "tests", true),
				If(p.Settings != nil && !t.IsProblemEditor && p.Settings.CheckerName != "", P(Class("text-sm text-muted"), ui.T(ctx, "gen_checker_warning"))),
			)),
			If(!t.CanViewTests, hiddenCheckbox("aTests")),
			archiveCheckbox(ctx, "aAtts", "attachments", true),
			If(t.IsProblemEditor, archiveCheckbox(ctx, "aPrivateAtts", "privateAttachments", true)),
			If(!t.IsProblemEditor, hiddenCheckbox("aPrivateAtts")),
			archiveCheckbox(ctx, "aDetails", "gen_details", true),
			archiveCheckbox(ctx, "aTags", "gen_tags", true),
			archiveCheckbox(ctx, "aEditors", "gen_editors", true),
			archiveCheckbox(ctx, "aSubs", "gen_submissions", true),
			If(t.IsProblemEditor, archiveCheckbox(ctx, "aAllSubs", "gen_all_submissions", false)),
			If(!t.IsProblemEditor, hiddenCheckbox("aAllSubs")),
			captcha(ctx, p.CaptchaID),
			Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "download")),
		),
		Script(Raw(fmt.Sprintf(archiveJS, pbID))),
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

func archiveCheckbox(ctx context.Context, id, key string, checked bool) Node {
	return Div(Class("block mb-2"), archiveCheckboxLabel(ctx, id, key, checked))
}

func archiveCheckboxLabel(ctx context.Context, id, key string, checked bool) Node {
	return Label(Class("inline-flex items-center text-lg"),
		Input(Class("form-checkbox"), ID(id), Type("checkbox"), If(checked, Checked())),
		Span(Class("ml-2"), ui.T(ctx, key)),
	)
}

func hiddenCheckbox(id string) Node {
	return Input(Class("hidden"), ID(id), Type("checkbox"))
}

func captcha(ctx context.Context, id string) Node {
	if id == "" {
		return Script(Raw(`function getCaptchaData() { return {} }`))
	}
	return Group{
		Div(ID("captcha_container"), Class("segment-panel my-2"),
			P(ui.T(ctx, "auth.captcha.title"), Text(":"), Br(),
				Img(ID("captcha_img"), Class("rounded-sm"), Src("/api/auth/captcha/"+id+".png"), Alt("Captcha image"), Width("240"), Height("80"), Style("background-color: white;")),
			),
			Input(Type("hidden"), ID("captcha_id"), Value(id)),
			Label(Class("block my-2"),
				Span(ui.T(ctx, "response")),
				Input(Type("text"), ID("captcha_response"), Class("form-input w-full"), Pattern("[0-9]*"), Value(""), AutoComplete("off")),
			),
			Button(ID("captcha_reload_btn"), Class("btn btn-blue"), Type("button"), I(Class("fas fa-fw fa-rotate")), Text(" "), ui.T(ctx, "reload")),
		),
		Script(Raw(captchaJS)),
	}
}

const captchaJS = `
function reloadCaptcha() {
	document.getElementById("captcha_img").src = ` + "`" + `/api/auth/captcha/${document.getElementById("captcha_id").value}.png?reload=${bundled.dayjs().valueOf()}` + "`" + `
}
document.getElementById("captcha_reload_btn").addEventListener("click", reloadCaptcha)
function getCaptchaData() {
	return {captcha_id: document.getElementById("captcha_id").value, captcha_response: document.getElementById("captcha_response").value}
}
`

// Placeholder: problem ID.
const archiveJS = `
var problemID = %d
function startArchiveDownload(e) {
	e.preventDefault()
	let url = new URL(` + "`" + `/assets/problem/${problemID}/problemArchive` + "`" + `, window.location)
	url.search = new URLSearchParams({
		tests: document.getElementById("aTests").checked,

		attachments: document.getElementById("aAtts").checked,
		private_attachments: document.getElementById("aPrivateAtts").checked,

		details: document.getElementById("aDetails").checked,
		tags: document.getElementById("aTags").checked,
		editors: document.getElementById("aEditors").checked,

		submissions: document.getElementById("aSubs").checked,
		all_submissions: document.getElementById("aAllSubs").checked,

		...getCaptchaData(),
	}).toString()
	window.location.assign(url.toString())
}
document.getElementById("problemArchiveForm").addEventListener("submit", startArchiveDownload)
`
