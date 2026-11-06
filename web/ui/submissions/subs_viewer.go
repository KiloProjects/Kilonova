// Package submissions renders a submission's page, its paste page, and the filterable
// submissions list.
package submissions

import (
	"context"
	"fmt"
	"maps"
	"math"
	"net/url"
	"slices"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SubsViewerQuery struct {
	UserID        *int
	Username      *string
	ProblemID     *int
	ProblemListID *int
	ContestID     *int
	Status        string
	Score         *decimal.Decimal
	Lang          string
	CompileError  *bool
	Ordering      string
	Ascending     bool
	Page          int
}

// Fields that are pre-set and hidden from the filter form
// TODO: Replace vibecoded solution with better
type SubsViewerOverwrites struct {
	UserID    bool
	Username  bool
	ProblemID bool
	ContestID bool
}

type SubsViewerParams struct {
	Submissions *sudoapi.Submissions
	Query       SubsViewerQuery
	Overwrites  SubsViewerOverwrites
	NumPages    int
	Title       string
	Languages   map[string]string
	IsAdmin     bool
}

func buildSortURL(q SubsViewerQuery, ordering string) string {
	ascending := false
	if q.Ordering == ordering {
		ascending = !q.Ascending
	}
	return buildQueryURL(q, map[string]string{
		"ordering":  ordering,
		"ascending": strconv.FormatBool(ascending),
		"page":      "1",
	})
}

func buildQueryURL(q SubsViewerQuery, overrides map[string]string) string {
	params := url.Values{}

	if q.UserID != nil {
		params.Set("user_id", strconv.Itoa(*q.UserID))
	}
	if q.Username != nil {
		params.Set("username", *q.Username)
	}
	if q.ProblemID != nil {
		params.Set("problem_id", strconv.Itoa(*q.ProblemID))
	}
	if q.ProblemListID != nil {
		params.Set("problem_list_id", strconv.Itoa(*q.ProblemListID))
	}
	if q.ContestID != nil {
		params.Set("contest_id", strconv.Itoa(*q.ContestID))
	}
	if q.Status != "" {
		params.Set("status", q.Status)
	}
	if q.Score != nil {
		params.Set("score", q.Score.String())
	}
	if q.Lang != "" {
		params.Set("lang", q.Lang)
	}
	if q.CompileError != nil {
		params.Set("compile_error", strconv.FormatBool(*q.CompileError))
	}
	if q.Ordering != "" && q.Ordering != "id" {
		params.Set("ordering", q.Ordering)
	}
	if q.Ascending {
		params.Set("ascending", "true")
	}
	if q.Page > 1 {
		params.Set("page", strconv.Itoa(q.Page))
	}

	for k, v := range overrides {
		if v == "" {
			params.Del(k)
		} else {
			params.Set(k, v)
		}
	}

	if len(params) == 0 {
		return ""
	}
	return "?" + params.Encode()
}

func sizeFormatter(size int) string {
	if size > 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/1024/1024)
	}
	if size > 1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%d B", size)
}

func getGradient(score, maxScore float64) string {
	if maxScore == 0 || score == 0 {
		if score == 0 {
			return "#f11722"
		}
		return "#e3dd71"
	}
	rap := score / maxScore
	switch {
	case rap == 1.0:
		return "#7fff00"
	case rap > 0.8:
		return "#67cf39"
	case rap > 0.6:
		return "#9fdd2e"
	case rap > 0.4:
		return "#d2eb19"
	case rap > 0.2:
		return "#f1d011"
	case rap > 0:
		return "#f6881e"
	}
	return "#f11722"
}

// rezStr picks the Romanian plural form for the result count; a negative count is unknown.
func rezStr(ctx context.Context, count int, truncated bool) string {
	switch {
	case count < 0:
		return "- " + ui.TS(ctx, "u20Results")
	case count == 0:
		return "0 " + ui.TS(ctx, "u20Results")
	case count == 1:
		return ui.TS(ctx, "oneResult")
	case count < 20:
		return fmt.Sprintf("%d %s", count, ui.TS(ctx, "u20Results"))
	case truncated:
		return ui.TS(ctx, "results")
	}
	return fmt.Sprintf("%d %s", count, ui.TS(ctx, "manyResults"))
}

func submissionStatus(ctx context.Context, sub *kilonova.Submission, problem *kilonova.Problem) string {
	switch sub.Status {
	case kilonova.StatusFinished:
		if sub.SubmissionType == kilonova.EvalTypeClassic {
			afterText := " " + ui.TS(ctx, "points")
			if problem != nil && math.Abs(problem.ScoreScale.InexactFloat64()-100.0) > 0.01 {
				afterText = "% " + ui.TS(ctx, "correct")
			}
			score := ui.RemoveTrailingZeros(sub.Score.StringFixed(sub.ScorePrecision))
			return fmt.Sprintf("%s: %s%s", ui.TS(ctx, "evaluated"), score, afterText)
		}
		if sub.Score.InexactFloat64() == 100 {
			return ui.TS(ctx, "accepted")
		}
		if sub.ICPCVerdict != nil {
			return icpcVerdictText(ctx, *sub.ICPCVerdict)
		}
		return ui.TS(ctx, "rejected")
	case kilonova.StatusWorking:
		return ui.TS(ctx, "evaluating")
	case kilonova.StatusReevaling:
		return ui.TS(ctx, "reevaling")
	}
	return ui.TS(ctx, "waiting")
}

func icpcVerdictText(ctx context.Context, verdict string) string {
	return icpcVerdictRe.ReplaceAllStringFunc(verdict, func(s string) string {
		return kilonova.MaybeGetText(util.LanguageContext(ctx), s)
	})
}

func contestProblemURL(contestID *int, problemID int) string {
	if contestID != nil {
		return fmt.Sprintf("/contests/%d/problems/%d", *contestID, problemID)
	}
	return fmt.Sprintf("/problems/%d", problemID)
}

// statusStyle colours a finished submission's status cell from red (0) to green (100).
func statusStyle(sub *kilonova.Submission) string {
	if sub.Status != kilonova.StatusFinished {
		return ""
	}
	score := sub.Score.InexactFloat64()
	if sub.SubmissionType == kilonova.EvalTypeICPC && score != 100 {
		score = 0
	}
	return "background-color: " + getGradient(score, 100) + ";"
}

func sortIndicator(q SubsViewerQuery, ordering string) Node {
	icon := "fa-sort"
	if q.Ordering == ordering {
		icon = "fa-sort-down"
		if q.Ascending {
			icon = "fa-sort-up"
		}
	}
	return Group{Text(" "), I(Class("fas " + icon))}
}

func SubsPage(ctx context.Context, params SubsViewerParams) Node {
	return Group{H1(ui.T(ctx, "subStatus")), SubsViewer(ctx, params)}
}

func SubsViewer(ctx context.Context, params SubsViewerParams) Node {
	q := params.Query
	// A problem filter swaps the problem column for code size.
	allProblems := q.ProblemID == nil || *q.ProblemID == 0
	subs := params.Submissions

	sortHeader := func(class, key, ordering string) Node {
		return Th(Scope("col"), Class(class),
			A(Class("block nostyle"), Href(buildSortURL(q, ordering)), ui.T(ctx, key), sortIndicator(q, ordering)),
		)
	}

	return Div(Class("page-holder mt-0"),
		Aside(Class("page-sidebar lg:order-last"),
			Div(Class("segment-panel"), H2(ui.T(ctx, "filters")), filterForm(ctx, params)),
		),
		Div(Class("page-content"),
			If(params.Title != "", H1(Text(params.Title))),
			H2(Class("inline-block"), Text(rezStr(ctx, subs.Count, subs.Truncated))),
			Div(Class("flex justify-center"), ui.Paginator(ui.PaginatorConfig{Page: q.Page, NumPages: params.NumPages, ContextSize: 2, ShowArrows: true})),
			Iff(!params.Overwrites.ProblemID && !allProblems && subs.Problems[*q.ProblemID] != nil, func() Node {
				p := subs.Problems[*q.ProblemID]
				return P(ui.T(ctx, "problemSingle"), Text(" "), A(Href(fmt.Sprintf("/problems/%d", p.ID)), Text(p.Name)))
			}),
			If(len(subs.Submissions) > 0, Div(Table(Class("kn-table"),
				THead(Tr(
					Th(Scope("col"), Class("w-20 text-center px-4 py-2"), ui.T(ctx, "id")),
					Th(Scope("col"), ui.T(ctx, "author")),
					sortHeader("cursor-pointer", "uploadDate", "id"),
					If(allProblems, Th(Scope("col"), ui.T(ctx, "problemSingle"))),
					If(!allProblems, sortHeader("cursor-pointer", "codeSize", "code_size")),
					sortHeader("cursor-pointer", "time", "max_time"),
					sortHeader("cursor-pointer", "memory", "max_mem"),
					sortHeader("w-1/6 cursor-pointer", "status", "score"),
				)),
				TBody(Map(subs.Submissions, func(sub *kilonova.Submission) Node {
					return subRow(ctx, subs, sub, allProblems)
				})),
			))),
			If(len(subs.Submissions) == 0, Div(Class("text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center"), ui.T(ctx, "noSubFound"))),
		),
		Script(Raw(`function copyQuery() {
	navigator.clipboard
		.writeText(window.location.origin + window.location.pathname + window.location.search)
		.then(function () {
			bundled.createToast({status: "success", title: bundled.getText("copied")});
		})
		.catch(function () {
			bundled.createToast({status: "error", title: bundled.getText("notCopied")});
		});
}`)),
	)
}

func subRow(ctx context.Context, subs *sudoapi.Submissions, sub *kilonova.Submission, allProblems bool) Node {
	cell := func(children ...Node) Node { return Td(Class("text-center px-2 py-1"), Group(children)) }
	author := Text("-")
	if u, ok := subs.Users[sub.UserID]; ok {
		author = A(Href("/profile/"+u.Name), Text(u.Name))
	}
	var third Node
	if allProblems {
		problem := Text("-")
		if p, ok := subs.Problems[sub.ProblemID]; ok {
			problem = A(Href(contestProblemURL(sub.ContestID, p.ID)), Text(p.Name))
		}
		third = cell(problem)
	} else {
		size := Span(Text("-"))
		if sub.CodeSize > 0 {
			size = Span(Text(sizeFormatter(sub.CodeSize)))
		}
		third = cell(size)
	}
	time, mem := Text("-"), Text("-")
	if sub.MaxTime != -1 {
		time = Textf("%.0fms", math.Floor(sub.MaxTime*1000))
	}
	if sub.MaxMemory != -1 {
		mem = Text(sizeFormatter(sub.MaxMemory * 1024))
	}
	return Tr(Class("kn-table-row"),
		Th(Scope("row"), Class("text-center px-2 py-1"), Text(strconv.Itoa(sub.ID))),
		Td(Class("px-2 py-1"), author),
		cell(ui.ServerTimestamp(sub.CreatedAt)),
		third,
		cell(time),
		cell(mem),
		Td(Class("text-center"), Style(statusStyle(sub)),
			A(Href(fmt.Sprintf("/submissions/%d", sub.ID)), Text(submissionStatus(ctx, sub, subs.Problems[sub.ProblemID]))),
		),
	)
}

func filterForm(ctx context.Context, params SubsViewerParams) Node {
	q, ow := params.Query, params.Overwrites
	label := func(key string, input Node) Node {
		return Label(Class("block mb-2"), Span(Class("form-label"), ui.T(ctx, key), Text(":")), Text(" "), input)
	}
	numberInput := func(name string, val *int) Node {
		return Input(Class("form-input"), Type("number"), Min("0"), Name(name), If(val != nil, Iff(val != nil, func() Node { return Value(strconv.Itoa(*val)) })))
	}
	hidden := func(name string, val *int, on bool) Node {
		return Iff(on && val != nil, func() Node { return Input(Type("hidden"), Name(name), Value(strconv.Itoa(*val))) })
	}
	option := func(value string, selected bool, label Node) Node {
		return Option(Value(value), If(selected, Selected()), label)
	}
	acceptedOnly := q.Status == "finished" && q.Score != nil && q.Score.InexactFloat64() == 100

	return Form(ID("sub-filters"), Method("GET"), Attr("hx-boost", "true"),
		hidden("user_id", q.UserID, ow.UserID),
		hidden("problem_id", q.ProblemID, ow.ProblemID),
		hidden("contest_id", q.ContestID, ow.ContestID),
		label("language", Select(Class("form-select"), Name("lang"),
			Option(Value(""), Text("-")),
			Map(slices.Sorted(maps.Keys(params.Languages)), func(name string) Node {
				return option(name, name == q.Lang, Text(params.Languages[name]))
			}),
		)),
		Label(Class("block mb-2"),
			Input(ID("accepted_only"), Type("checkbox"), Class("form-checkbox mr-2"), Value("true"), If(acceptedOnly, Checked())),
			Span(Class("form-label"), ui.T(ctx, "acceptedSubs")),
			Script(Raw(`document.getElementById("accepted_only").addEventListener("change", (e) => {
	if (e.target.checked) {
		document.getElementById("sub_status").value = "finished";
		document.getElementById("sub_score").value = "100";
	} else {
		document.getElementById("sub_status").value = "";
		document.getElementById("sub_score").value = "";
	}
	document.getElementById("sub-filters").submit();
});`)),
		),
		Details(Class("block mb-2"),
			Summary(Class("form-label"), ui.T(ctx, "advancedOptions")),
			label("status", Select(ID("sub_status"), Class("form-select"), Name("status"),
				Option(Value(""), Text("-")),
				Map([]string{"finished", "working", "waiting", "reevaling"}, func(s string) Node {
					return option(s, s == q.Status, ui.T(ctx, s))
				}),
			)),
			If(!ow.UserID, label("userID", numberInput("user_id", q.UserID))),
			If(!ow.UserID && !ow.Username, label("username", Input(Class("form-input"), Type("text"), Name("username"),
				Iff(q.Username != nil, func() Node { return Value(*q.Username) })))),
			If(!ow.ProblemID, label("problemID", numberInput("problem_id", q.ProblemID))),
			label("problemListID", numberInput("problem_list_id", q.ProblemListID)),
			If(!ow.ContestID, label("contestID", numberInput("contest_id", q.ContestID))),
			label("score", Input(ID("sub_score"), Class("form-input"), Type("number"), Min("-1"), Max("100"), Step("0.0001"), Name("score"),
				Iff(q.Score != nil, func() Node { return Value(q.Score.String()) }))),
			label("compileErr", Select(Class("form-select"), Name("compile_error"),
				Option(Value(""), Text("-")),
				option("true", q.CompileError != nil && *q.CompileError, ui.T(ctx, "yes")),
				option("false", q.CompileError != nil && !*q.CompileError, ui.T(ctx, "no")),
			)),
		),
		ui.SubmitButton(Class("mb-4 mr-2"), ui.T(ctx, "fetch")),
		Text(" "),
		Button(Type("button"), Class("btn mb-4 mr-2"), Attr("onclick", "copyQuery()"), ui.T(ctx, "filterLink")),
		If(params.IsAdmin, Group{Text(" "), A(Class("btn"), Href("/assets/submissions/export"+buildQueryURL(q, nil)), ui.T(ctx, "exportSubs"))}),
	)
}
