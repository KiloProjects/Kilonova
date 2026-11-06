// Package lists renders the problem list pages and the problem search page.
package lists

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/proposer"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// IndexParams is the root problem lists page. Others are the problems in no list.
type IndexParams struct {
	Lists  []problems.ListGroupParams
	Others []*kilonova.ScoredProblem
	Authed bool
}

// Index lists the root problem lists, followed by the problems in no list.
func Index(ctx context.Context, p IndexParams) Node {
	return Div(Class("segment-panel"),
		H2(Class("mb-2"), ui.T(ctx, "problem_lists")),
		Div(Class("list-group"),
			Map(p.Lists, func(l problems.ListGroupParams) Node { return problems.ListGroup(ctx, l) }),
			If(len(p.Others) > 0, Details(Class("list-group-head"),
				Summary(Class("pb-1 mt-1"),
					Span(ui.T(ctx, "the_rest")),
					If(p.Authed, Group{Text(" "), Span(Class("float-right badge"), ui.T(ctx, "num_solved", numSolved(p.Others), len(p.Others)))}),
				),
				Div(Class("mt-2"), problems.ProblemListGroup(ctx, &problems.ProblemListingParams{Problems: p.Others, ShowID: true, ShowPublished: true})),
			)),
		),
	)
}

func numSolved(pbs []*kilonova.ScoredProblem) int {
	var cnt int
	hundred := decimal.NewFromInt(100)
	for _, pb := range pbs {
		if pb.MaxScore != nil && pb.MaxScore.Equal(hundred) {
			cnt++
		}
	}
	return cnt
}

// ProgressIndexParams is the featured checklist picker. NumSolved is only read when Authed.
type ProgressIndexParams struct {
	RootList  int
	Lists     []*kilonova.ProblemList
	Authed    bool
	NumSolved map[int]int
}

// ProgressIndex lists the featured checklists.
func ProgressIndex(ctx context.Context, p ProgressIndexParams) Node {
	return Div(Class("segment-panel reset-list"),
		A(Class("block mb-2"), Href(fmt.Sprintf("/problem_lists/%d", p.RootList)), I(Class("fas fa-arrow-left")), Text(" "), ui.T(ctx, "viewAllLists")),
		H1(ui.T(ctx, "selectChecklist")),
		If(len(p.Lists) == 0, ui.T(ctx, "noFeaturedChecklist")),
		Map(p.Lists, func(l *kilonova.ProblemList) Node {
			return Ul(Li(
				A(Href(fmt.Sprintf("/problem_lists/%d/progress", l.ID)), Text(l.Title)),
				If(p.Authed, Textf(" (%d/%d)", p.NumSolved[l.ID], l.NumProblems)),
			))
		}),
	)
}

// ProgressParams is a problem list's checklist as seen by CheckedUser (nil for a visitor).
type ProgressParams struct {
	List        *sudoapi.FullProblemList
	CheckedUser *kilonova.UserBrief
}

// Checklist is the checklist table of a problem list, plus a form to view it as another user.
func Checklist(ctx context.Context, p ProgressParams) Node {
	l, cu := p.List, p.CheckedUser
	listHref := fmt.Sprintf("/problem_lists/%d", l.ID)
	return Group{
		Div(Class("segment-panel"),
			A(Class("block mb-2"), Href(listHref), I(Class("fas fa-arrow-left")), Text(" "), ui.T(ctx, "list.view_list")),
			Iff(cu != nil, func() Node {
				return Span(Class("block my-2"), ui.T(ctx, "list.user_view_info"), Text(" "),
					A(Href("/profile/"+cu.Name),
						Img(Class("rounded-sm inline-block mr-1"), Src("/profile/"+cu.Name+"/avatar?s=32"), Width("32"), Height("32"), Alt("Avatar")),
						Text(" "+cu.Name),
					),
					Text("."),
				)
			}),
			Table(Class("kn-table kn-table-slim"),
				THead(Tr(Th(Class("kn-table-cell"), ColSpan("9999"),
					A(Class("text-2xl white-anchor font-bold"), Href(listHref), Text(l.Title+" "),
						If(cu != nil, Textf("(%d/%d)", l.SolvedCount, l.NumProblems))),
				))),
				TBody(checklistRows(ctx, l, cu != nil)),
			),
		),
		Form(ID("otherUserForm"), Class("segment-panel"), Method("GET"),
			H2(ui.T(ctx, "list.view_as_user")),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "username")),
				Input(ID("otherUsername"), Name("username"), Type("text"), Class("form-input")),
			),
			Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "view")),
		),
		Script(Raw(otherUserScript)),
	}
}

const otherUserScript = `
document.getElementById("otherUserForm").addEventListener("submit", async (e) => {
	e.preventDefault()
	const username = document.getElementById("otherUsername").value
	let res = await bundled.getCall(` + "`/user/byName/${encodeURIComponent(username)}`" + `, {})
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	var params = new URLSearchParams(window.location.search)
	params.set("username", username);
	window.location.search = params.toString()
})
`

// checklistRows renders a list as one row (its title cell spanning its sublists' rows)
// followed by its sublists' rows. Lists with nothing to show render nothing.
func checklistRows(ctx context.Context, l *sudoapi.FullProblemList, withUser bool) Node {
	span := ChecklistSpan(l)
	if span == 0 {
		return nil
	}
	href := fmt.Sprintf("/problem_lists/%d", l.ID)
	solved := l.SolvedCount
	if solved == -1 {
		solved = 0
	}
	var title Node
	switch {
	case !l.Root && l.DepthLevel != 1:
		title = Td(Class("kn-table-cell"), RowSpan(strconv.Itoa(span)),
			A(Class("white-anchor font-bold"), Href(href), Text(l.Title+":")),
			If(withUser, Group{Br(), Textf("(%d/%d)", solved, l.NumProblems)}))
	case !l.Root:
		title = Td(Class("kn-table-cell"), RowSpan(strconv.Itoa(span)),
			A(Class("white-anchor font-bold"), Href(href), Text(l.Title)), Text(" "),
			If(withUser, Textf("(%d/%d)", solved, l.NumProblems)))
	case len(l.Problems) > 0:
		title = Td(Class("kn-table-cell"), A(Class("white-anchor font-bold"), Href(href), Text(l.Title)))
	}
	return Group{
		Tr(Class("kn-table-simple"), title,
			Map(l.Problems, func(pb *kilonova.ScoredProblem) Node {
				return Td(Class("kn-table-cell score-cell"), Data("score", checklistScore(pb)),
					A(Href(fmt.Sprintf("/problems/%d?list_id=%d", pb.ID, l.ID)),
						Title(ui.TS(ctx, "score")+": "+scoreText(pb)), Text(pb.Name)))
			}),
		),
		Map(l.SubLists, func(sl *sudoapi.FullProblemList) Node { return checklistRows(ctx, sl, withUser) }),
	}
}

// ChecklistSpan is the number of table rows list takes in the checklist.
func ChecklistSpan(list *sudoapi.FullProblemList) int {
	var cnt, countedSubLists int
	for _, sublist := range list.SubLists {
		cnt1 := ChecklistSpan(sublist)
		cnt += cnt1
		if cnt1 > 0 {
			countedSubLists++
		}
	}
	if len(list.Problems) > 0 || countedSubLists > 0 {
		cnt++
	}
	return cnt
}

// checklistScore is the cell's data-score: the score, or -1 when there is none.
func checklistScore(pb *kilonova.ScoredProblem) string {
	if pb.ScoreUserID == nil || pb.MaxScore == nil || pb.MaxScore.IsNegative() {
		return "-1"
	}
	return ui.RemoveTrailingZeros(pb.MaxScore.StringFixed(pb.ScorePrecision))
}

// scoreText is ui.Score as tooltip text: empty with no score loaded, and for ICPC
// problems, whose score is an icon.
func scoreText(pb *kilonova.ScoredProblem) string {
	if pb.ScoreUserID == nil || pb.ScoringStrategy == kilonova.ScoringTypeICPC && pb.MaxScore != nil && !pb.MaxScore.IsNegative() {
		return ""
	}
	var b strings.Builder
	if err := ui.Score(&pb.Problem, pb.MaxScore, true).Render(&b); err != nil {
		return ""
	}
	return b.String()
}

// ViewParams is a problem list's page. Group is the list itself, loaded open.
// CanEdit (author or admin) shows the update form, IsAdmin the bulk update and sources.
type ViewParams struct {
	List    *kilonova.ProblemList
	Group   problems.ListGroupParams
	Parents []*kilonova.ProblemList
	Author  *kilonova.UserBrief // nil when unknown
	Authed  bool
	CanEdit bool
	IsAdmin bool

	ProblemSources Node
}

// View is a problem list's page.
func View(ctx context.Context, p ViewParams) Node {
	l := p.List
	id := strconv.Itoa(l.ID)
	return Group{
		Map(p.Parents, func(pl *kilonova.ProblemList) Node {
			return A(Class("block mb-2"), Href(fmt.Sprintf("/problem_lists/%d", pl.ID)), I(Class("fas fa-arrow-left")), Text(" "+pl.Title))
		}),
		A(Class("block mb-2"), Href("/problem_lists/"+id+"/progress"), ui.T(ctx, "list.view_checklist"), Text(" "), I(Class("fas fa-arrow-right"))),
		Div(Class("list-group mb-4"), problems.ListGroup(ctx, p.Group)),
		Div(Class("segment-panel"),
			P(ui.T(ctx, "created_by"), Text(": "), author(p.Author), Text(" ("), ui.ServerTimestamp(l.CreatedAt), Text(")")),
			Div(Class("my-2"),
				A(Class("btn btn-blue mr-2"), Href("/problems?deep_list_id="+id), ui.T(ctx, "use_in_search")), Text(" "),
				A(Class("btn btn-blue mr-2"), Href("/submissions?problem_list_id="+id), ui.T(ctx, "pblist_submissions")),
			),
			Div(Class("block my-2"),
				A(Class("btn btn-blue mr-2"), Href("/problems/random?list_id="+id), ui.T(ctx, "random_problem")),
				If(p.Authed, Group{Text(" "), A(Class("btn btn-blue"), Href("/problems/random?list_id="+id+"&unsolved=true"), ui.T(ctx, "random_unsolved_problem"))}),
			),
			If(p.Authed && len(l.List) > 0, simulationForm(ctx, l)),
		),
		If(p.CanEdit, Group{
			updateForm(ctx, l),
			If(p.IsAdmin, bulkVisibilityForm(ctx, l)),
			Script(Raw(fmt.Sprintf(updateScript, l.ID, l.ID, l.ID))),
			Details(Class("segment-panel"), Open(),
				Summary(H2(Class("inline-block"), ui.T(ctx, "createNestedPbList"))),
				proposer.CreateProblemList(ctx, l),
			),
			If(p.IsAdmin, p.ProblemSources),
		}),
	}
}

func author(u *kilonova.UserBrief) Node {
	if u == nil {
		return Text("-")
	}
	return A(Href("/profile/"+u.Name),
		Img(Class("rounded-sm inline-block mx-1"), Src("/profile/"+u.Name+"/avatar?s=32"), Width("24"), Height("24"), Alt("Avatar")),
		Text(" "+u.Name))
}

func checkbox(ctx context.Context, name, key string, checked bool) Node {
	return Div(Class("block my-2"),
		Label(Class("inline-flex items-center text-lg"),
			Input(Class("form-checkbox"), Name(name), ID(name), Type("checkbox"), If(checked, Checked())),
			Span(Class("ml-2"), ui.T(ctx, key)),
		),
	)
}

func simulationForm(ctx context.Context, l *kilonova.ProblemList) Node {
	list, _ := json.Marshal(l.List) // json escapes <, > and &, so this is safe in a script
	return Group{
		Form(ID("sim_create"), Class("segment-panel"), AutoComplete("off"),
			H2(ui.T(ctx, "create_simulation")),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "sim_name"), Text(": ")),
				Input(Type("text"), ID("sim_name"), Class("form-input"), Value(ui.TS(ctx, "simulation")+" - "+l.Title)),
			),
			checkbox(ctx, "start_now", "start_now", true),
			Label(ID("dur_label"), Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "duration"), Text(": ")),
				Input(Type("number"), ID("sim_duration"), Class("form-input"), Value("180")), Text(" "), ui.T(ctx, "minutes"),
			),
			Button(ID("sim_submit"), Type("submit"), Class("btn btn-blue my-2"), ui.T(ctx, "button.create")),
		),
		Script(Raw(strings.Replace(simulationScript, "LIST_JSON", string(list), 1))),
	}
}

const simulationScript = `
document.getElementById("start_now").addEventListener("change", (e) => {
	document.getElementById("dur_label").classList.toggle("hidden", !e.currentTarget.checked)
	document.getElementById("sim_submit").innerText = (e.currentTarget.checked ? bundled.getText("button.create") : bundled.getText("create_edit"))
})
async function createSimulation(e) {
	e.preventDefault()
	let name = document.getElementById("sim_name").value;
	let startNow = document.getElementById("start_now").checked;
	let dur = document.getElementById("sim_duration").value;
	let res = await bundled.postCall("/contest/create", {
		name,
		type: "virtual"
	})
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	bundled.apiToast({status: "success", data: "Created contest. Initializing..."})
	let id = res.data
	res = await bundled.bodyCall(` + "`/contest/${id}/update/problems`" + `, {list: LIST_JSON})
	if(res.status === "error") {
		res.data = ` + "`Contest created with errors: ${res.data}. <a href=\"/contests/${id}\">View it here</a>`" + `
		bundled.apiToast(res)
		return
	}
	if(!startNow) {
		window.location.assign(` + "`/contests/${id}/manage/edit`" + `)
		return
	}
	res = await bundled.postCall(` + "`/contest/${id}/register`" + `, {});
	if (res.status === "error") {
		res.data = ` + "`Contest created with errors: ${res.data}. <a href=\"/contests/${id}\">View it here</a>`" + `
		bundled.apiToast(res);
		return;
	}
	let startTime = bundled.dayjs(), endTime = startTime.add(dur, 'minute')
	res = await bundled.postCall(` + "`/contest/${id}/update`" + `, {
		start_time: bundled.formatISO8601(startTime),
		end_time: bundled.formatISO8601(endTime)
	})
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	window.location.assign(` + "`/contests/${id}`" + `)
}
document.getElementById("sim_create").addEventListener("submit", createSimulation)
`

func intList(ids []int) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = strconv.Itoa(id)
	}
	return strings.Join(s, ",")
}

func updateForm(ctx context.Context, l *kilonova.ProblemList) Node {
	subIDs := make([]int, len(l.SubLists))
	for i, sl := range l.SubLists {
		subIDs[i] = sl.ID
	}
	return Form(ID("pblist-update"), Class("segment-panel"), AutoComplete("off"),
		H1(ui.T(ctx, "list.update")),
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "title"), Text(": ")),
			Input(Type("text"), ID("pblist-title"), Class("form-input"), Value(l.Title)),
		),
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "sublist_label"), Text(":")),
			Input(Type("text"), ID("pblist-sublists"), Class("form-input"), Pattern("([0-9]+,?)+"), Value(intList(subIDs))),
		),
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "pbs"), Text(":")),
			Input(Type("text"), ID("pblist-list"), Class("form-input"), Pattern("([0-9]+,?)+"), Value(intList(l.List))),
		),
		checkbox(ctx, "sidebar_hidable", "sidebar_hidable", l.SidebarHidable),
		checkbox(ctx, "featured_checklist", "featured_checklist", l.FeaturedChecklist),
		Label(Class("block my-2 mb-4"),
			Span(Class("form-label"), ui.T(ctx, "desc"), Text(": ")),
			Textarea(ID("desc_tarea"), Class("hidden"), Text(l.Description)),
		),
		Button(Class("btn btn-blue mr-2"), Type("submit"), ui.T(ctx, "button.update")), Text(" "),
		Button(ID("delButton"), Class("btn btn-red"), ui.T(ctx, "button.delete")),
	)
}

func bulkVisibilityForm(ctx context.Context, l *kilonova.ProblemList) Node {
	opt := func(value, key string) Node { return Option(Value(value), ui.T(ctx, key)) }
	placeholder := Option(Value(""), Selected(), Disabled(), Attr("hidden"), Text("-"))
	return Group{
		Form(ID("problemBulkVizForm"), Class("segment-panel"), AutoComplete("off"),
			Span(Class("text-lg"), ui.T(ctx, "updateProblems1")), Text(" "),
			// Both selects share an id, as in the original template.
			Select(Class("form-select"), Name("deep"), ID("updProblemsDeep"), Required(), AutoComplete("off"),
				placeholder,
				opt("false", "updateProblemsShallow"),
				opt("true", "updateProblemsRecursive"),
			), Text(" "),
			Span(Class("text-lg"), ui.T(ctx, "updateProblems2")), Text(" "),
			Select(Class("form-select"), Name("property"), ID("updProblemsDeep"), Required(), AutoComplete("off"),
				placeholder,
				opt("visible.false", "updateProblemsHidden"),
				opt("visible.true", "updateProblemsVisible"),
				opt("visibleTests.false", "updateProblemTestsHidden"),
				opt("visibleTests.true", "updateProblemTestsVisible"),
				opt("reviewRequested.true", "updateProblemReviewRequested"),
				opt("reviewRequested.false", "updateProblemReviewRescinded"),
			), Text(" "),
			Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "button.update")),
		),
		Script(Rawf(bulkScript, l.ID)),
	}
}

const bulkScript = `
async function updateProblemVisibilities(e) {
	e.preventDefault();
	let data = new FormData(e.currentTarget)
	let [propertyName, propertyVal] = data.get("property").split('.', 2);
	const res = await bundled.postCall("/problemList/%d/toggleProblems", {
		deep: data.get("deep"),
		[propertyName]: propertyVal
	})
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	window.location.reload()
}

document.getElementById("problemBulkVizForm").addEventListener("submit", updateProblemVisibilities);
`

// updateScript takes the list ID three times.
const updateScript = `
let problemListID = %d;
var cm = new vendored.KNEditor({
	textArea: document.getElementById("desc_tarea"),
	language: "md",

	dynamicSize: true
});

async function updateProblemList(e) {
	e.preventDefault();
	let data = {
		title: document.getElementById("pblist-title").value,
		description: cm.getText(),
		list: bundled.stringIntToNumber(document.getElementById("pblist-list").value.split(',')),
		sublists: bundled.stringIntToNumber(document.getElementById("pblist-sublists").value.split(',')),
		sidebar_hidable: document.getElementById("sidebar_hidable").checked,
		featured_checklist: document.getElementById("featured_checklist").checked,
	};
	let res = await bundled.bodyCall("/problemList/%d/update", data)
	if(res.status === "success") {
		window.location.reload();
		return
	}
	bundled.apiToast(res)
}
async function deleteProblemList(e) {
	e.preventDefault();
	if(!(await bundled.confirm(bundled.getText("pblistDeleteConfirm")))) {
		return
	}
	let res = await bundled.postCall("/problemList/%d/delete", {})
	if(res.status === "success") {
		window.location.assign("/problem_lists");
		return
	}
	bundled.apiToast(res);
}
document.getElementById("pblist-update").addEventListener("submit", updateProblemList)
document.getElementById("delButton").addEventListener("click", deleteProblemList);
`

// SearchParams feeds the problem search web component.
type SearchParams struct {
	ProblemList *kilonova.ProblemList
	Results     []*sudoapi.FullProblem
	Groups      []*kilonova.TagGroup
	GroupTags   []*kilonova.Tag
	Count       int
}

// ProblemSearch mounts the problem search component.
func ProblemSearch(p SearchParams) Node {
	return El("kn-pb-search",
		Attr("enc", ui.EncodeJSON(p.Results)),
		Attr("count", strconv.Itoa(p.Count)),
		Attr("groupEnc", ui.EncodeJSON(p.Groups)),
		Attr("tagEnc", ui.EncodeJSON(p.GroupTags)),
		Attr("pblistEnc", ui.EncodeJSON(p.ProblemList)),
	)
}
