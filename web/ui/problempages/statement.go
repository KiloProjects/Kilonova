package problempages

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ParentList is a problem list containing the viewed problem, with its problems
// scored for the viewing user. Problems is nil if they couldn't be loaded.
type ParentList struct {
	List     *kilonova.ProblemList
	Problems []*kilonova.ScoredProblem
}

type StatementParams struct {
	Topbar *problems.TopbarParams
	// ContestSidebar replaces the problem sidebar when viewing the problem in a contest.
	ContestSidebar *contests.SidebarParams
	Info           *problems.InfoParams
	// FromContest shows the contest disclaimer.
	FromContest bool
	ParentLists []ParentList
	// nil hides the older submissions panel.
	OlderSubmissions *modals.OlderSubmissionsParams
	Attachments      []*kilonova.Attachment

	ShowExternalResources bool
	ExternalResources     []*kilonova.ExternalResource

	Variants        []*kilonova.StatementVariant
	SelectedVariant *kilonova.StatementVariant
	// StatementHTML is trusted markup (rendered markdown or an embed).
	StatementHTML string

	// nil when the user isn't logged in; the page then asks them to log in.
	SubmitForm *SubmitFormParams
	ReqPath    string
}

// StatementDescription is the meta description of a problem page.
func StatementDescription(pb *kilonova.Problem, tags []*kilonova.Tag) string {
	desc := pb.Name
	if pb.SourceCredits != "" {
		desc += " - propusă la " + pb.SourceCredits
	}
	var authors []string
	for _, tag := range tags {
		if tag.Type == kilonova.TagTypeAuthor {
			authors = append(authors, tag.Name)
		}
	}
	if len(authors) > 0 {
		desc += " - de către " + strings.Join(authors, ", ")
	}
	return desc
}

// Statement is the problem statement page: limits, statement, sidebar and submit form.
func Statement(ctx context.Context, p StatementParams) Node {
	pb := p.Topbar.Problem
	back := url.QueryEscape(p.ReqPath)
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"),
			Div(Class("page-content reset-list enhance-tables statement-content"),
				If(len(p.Variants) > 1, variantSelect(ctx, p)),
				Div(Class("w-full mb-6 mt-2 text-center"),
					Span(Class("block"), ui.T(ctx, "timeLimit"), Textf(": %vs", pb.TimeLimit)),
					Span(Class("block"), ui.T(ctx, "memoryLimit"), Textf(": %vMB", math.Round(float64(pb.MemoryLimit)/1024.0*100)/100.0)),
					Span(Class("block"), ui.T(ctx, "input"), Text(": "), ioFile(pb, "stdin", ".in")),
					Span(Class("block"), ui.T(ctx, "output"), Text(": "), ioFile(pb, "stdout", ".out")),
					If(!pb.DefaultPoints.IsZero(), Span(Class("block"), ui.T(ctx, "defaultPoints"), Textf(": %sp", pb.DefaultPoints))),
				),
				Article(Class("text-justify"),
					If(p.SelectedVariant != nil && p.SelectedVariant.Type == "llm", BlockQuote(ui.T(ctx, "aiTranslationWarning"))),
					Raw(p.StatementHTML),
				),
				Div(Style("clear: both")),
			),
			Aside(Class("page-sidebar"),
				sidebar(ctx, p),
				Iff(p.OlderSubmissions != nil, func() Node {
					return Section(Class("segment-panel"), modals.OlderSubmissions(ctx, p.OlderSubmissions))
				}),
				If(len(p.Attachments) > 0, attachments(ctx, pb, p.Attachments)),
				If(p.Topbar.Contest == nil && p.ShowExternalResources, externalResources(ctx, p)),
			),
		),
		Script(Raw(statementAttachmentsJS)),
		Script(Raw(fmt.Sprintf(statementScoreJS, pb.ScorePrecision, ui.JS(string(pb.ScoringStrategy)), pb.ID))),
		Iff(p.SubmitForm != nil, func() Node { return SubmitForm(ctx, *p.SubmitForm) }),
		If(p.SubmitForm == nil, Div(Class("segment-panel mt-3 mb-5"),
			H1(
				A(Href("/login?back="+back), ui.T(ctx, "auth.login")), Text(" "),
				ui.T(ctx, "submitLogin1"), Text(" "),
				A(Href("/signup?back="+back), ui.T(ctx, "auth.signup")), Text(" "),
				ui.T(ctx, "submitLogin2"),
			),
		)),
	}
}

func variantSelect(ctx context.Context, p StatementParams) Node {
	return Label(Class("block my-2"),
		Span(Class("form-label"), ui.T(ctx, "statementFormat"), Text(":")),
		Select(ID("variant-select"), Class("form-select"), AutoComplete("off"),
			Map(p.Variants, func(v *kilonova.StatementVariant) Node {
				return Option(Value(v.Language+"-"+v.Format+"-"+v.Type),
					If(p.SelectedVariant != nil && p.SelectedVariant.Equals(v), Selected()),
					Text(ui.StatementVariantName(ctx, v)))
			}),
		),
	)
}

func ioFile(pb *kilonova.Problem, stream, ext string) Node {
	if pb.ConsoleInput {
		// Both streams link the "stdin" glossary entry, as in the original template.
		return El("kn-glossary", Attr("name", "stdin"), Attr("content", stream))
	}
	return Text(pb.TestName + ext)
}

func sidebar(ctx context.Context, p StatementParams) Node {
	if p.ContestSidebar != nil {
		return contests.Sidebar(ctx, *p.ContestSidebar)
	}
	return Group{
		If(p.FromContest, problems.ContestDisclaimer(ctx)),
		problems.InfoSidebar(ctx, p.Info),
		Map(p.ParentLists, func(l ParentList) Node { return parentList(l, p.Topbar.Problem.ID) }),
	}
}

func parentList(l ParentList, currentID int) Node {
	return Details(Class("segment-panel"), Attr("open"), Role("region"),
		Summary(H2(Class("font-normal inline-block"), A(Href(fmt.Sprintf("/problem_lists/%d", l.List.ID)), Text(l.List.Title)))),
		Div(Class("reset-list text-lg"),
			If(len(l.Problems) > 0, Ul(Map(l.Problems, func(pb *kilonova.ScoredProblem) Node {
				id := strconv.Itoa(pb.ID)
				return Li(
					A(If(pb.ID == currentID, Class("topbar-selected p-0")), Href(fmt.Sprintf("/problems/%d?list_id=%d", pb.ID, l.List.ID)), Text(pb.Name)),
					Iff(pb.ScoreUserID != nil, func() Node {
						score := ui.Score(&pb.Problem, pb.MaxScore, true)
						el := Span(Data("pbid-reload", id), score)
						if pb.ScoringStrategy == kilonova.ScoringTypeSumSubtasks {
							el = A(Class("max_score_breakdown"), Href("#"), Data("problemid", id), Data("pbid-reload", id), score)
						}
						return Group{Text(" ("), el, Text(")")}
					}),
				)
			}))),
		),
	)
}

func attachments(ctx context.Context, pb *kilonova.Problem, atts []*kilonova.Attachment) Node {
	return Details(Class("segment-panel"), Attr("open"), Role("region"),
		Summary(H2(Class("inline-block"), ui.T(ctx, "attachments"))),
		Div(Class("px-2"), Div(Class("list-group"),
			Map(atts, func(att *kilonova.Attachment) Node {
				return A(Style("word-wrap: anywhere;"), Href(fmt.Sprintf("/assets/problem/%d/attachment/%s", pb.ID, url.PathEscape(att.Name))), Class("list-group-item flex justify-between"),
					Span(ID(fmt.Sprintf("att-name-%d", att.ID)), Text(att.Name)),
					Span(Class("badge-lite text-sm"), ID(fmt.Sprintf("att-size-%d", att.ID)), Text(strconv.Itoa(att.Size))),
				)
			}),
		)),
	)
}

func externalResources(ctx context.Context, p StatementParams) Node {
	list := Div(Class("list-group-head"), ui.T(ctx, "no_resources"))
	if len(p.ExternalResources) > 0 {
		list = Map(p.ExternalResources, func(res *kilonova.ExternalResource) Node {
			return A(Style("text-overflow: ellipsis;"), Href(resourceURL(res.URL)), Class("list-group-item flex justify-between"),
				Title(ui.TS(ctx, "external_resource.type."+string(res.Type))),
				Span(I(Class("fas fa-fw "+res.Type.FontAwesomeIcon())), Text(" "+res.Name)),
			)
		})
	}
	prefix := fmt.Sprintf("%s/problems/%d/externalResources", p.Topbar.URLPrefix, p.Topbar.Problem.ID)
	return Details(Class("segment-panel"), If(len(p.ExternalResources) > 0, Attr("open")), Role("region"),
		Summary(H2(Class("inline-block"), ui.T(ctx, "external_resources"))),
		Div(Class("px-2"), Div(Class("list-group mb-2"), list)),
		If(p.Topbar.IsProblemEditor, Div(Class("grid grid-cols-1 lg:grid-cols-2 justify-between"),
			A(Href(prefix), ui.T(ctx, "view_details")),
			A(Class("lg:text-right"), Href(prefix+"/create"), ui.T(ctx, "suggest_resource")),
		)),
	)
}

// resourceURL tags an external resource link with UTM parameters.
func resourceURL(orgURL string) string {
	u, err := url.Parse(orgURL)
	if err != nil {
		return orgURL
	}
	q := u.Query()
	q.Set("utm_source", "resource")
	q.Set("utm_medium", "external_resource")
	q.Set("utm_campaign", "kilonova")
	u.RawQuery = q.Encode()
	return u.String()
}

const statementAttachmentsJS = `
(() => {
	for (let el of document.querySelectorAll("[id^='att-name-']")) {
		el.innerHTML = ` + "`" + `<i class="fas ${bundled.getFileIcon(el.innerText)} fa-fw"></i> ${el.innerHTML}` + "`" + `
	}
	for (let el of document.querySelectorAll("[id^='att-size-']")) {
		el.innerText = bundled.sizeFormatter(Number(el.innerText))
	}
})()
`

// Placeholders: score precision, scoring strategy (JS string), problem ID (twice).
const statementScoreJS = `
document.getElementById("variant-select")?.addEventListener("change", e => {
	var params = new URLSearchParams(window.location.search);
	params.set("var", e.target.value.replace(/-+$/, ""))
	window.location.search = params.toString()
})
let prec = %[1]d;
let strat = %[2]s;

function formatMaxScore(maxScore) {
	let score = parseInt(maxScore);
	if (isNaN(score)) {
		console.error("Unknown value for maxScore")
		return "ERR";
	}
	if (maxScore < 0) {
		return "-"
	}
	if (strat == "acm-icpc") {
		if (maxScore == 100) {
			return ` + "`" + `<i class="fas fa-fw fa-check"></i>` + "`" + `;
		}
		return ` + "`" + `<i class="fas fa-fw fa-xmark"></i>` + "`" + `;
	}
	return bundled.formatScoreStr(maxScore.toFixed(prec)) + "p";
}

const reloaders = document.querySelectorAll(` + "`" + `[data-pbid-reload="%[3]d"]` + "`" + `)
if (reloaders.length > 0) {
	document.addEventListener("kn-poll", async () => {
		// Update max score on poll
		const res = await bundled.getCall("/problem/%[3]d/maxScore", {})
		if (res.status === "error") {
			console.error(res.data)
			return
		}
		let val = formatMaxScore(res.data)
		for (let reloader of reloaders) {
			if ("contest_id" in reloader.dataset) {
				// TODO
			} else {
				reloader.innerHTML = val
			}
		}
	})
}
`
