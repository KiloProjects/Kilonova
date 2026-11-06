package pbedit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type GeneralParams struct {
	Problem *kilonova.Problem
	Topbar  *problems.TopbarParams

	// Checklist must not be nil; pass a zero checklist if it couldn't be loaded.
	Checklist *kilonova.ProblemChecklist
	// Diagnostics are preformatted "LEVEL: message" lines.
	Diagnostics []string
	// Settings is nil if the eval settings couldn't be computed.
	Settings *kilonova.ProblemEvalSettings
	Tags     []*kilonova.Tag

	MaxMemMB   float64
	LLMEnabled bool
}

// General is the problem editor's general settings page, with the checklist and diagnostics.
func General(ctx context.Context, params GeneralParams) Node {
	pb := params.Problem
	isAdmin := user.UserBriefContext(ctx).IsAdmin()
	return Group{page(ctx, params.Topbar,
		Div(Class("page-content-wrapper"),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "header.edit.index")),
				updateForm(ctx, pb, params.MaxMemMB),
				Div(Class("block my-2"),
					Button(Type("button"), ID("deleteProblemButton"), Class("btn btn-red mr-2"), ui.T(ctx, "deleteProblem")),
				),
			),
			Div(Class("segment-panel"),
				H2(Class("mb-0"), ui.T(ctx, "problem_tags")),
				P(Class("text-sm text-muted mb-2"), ui.T(ctx, "problem_tags_explainer")),
				El("kn-pb-tag-edit", Attr("enc", encodeTags(params.Tags)), Attr("pbid", strconv.Itoa(pb.ID))),
			),
			checklist(ctx, pb, params.Checklist, len(params.Diagnostics) == 0, isAdmin),
			script("general.js", map[string]any{"PROBLEM": pb}),
		),
		Aside(Class("page-sidebar"),
			Div(Class("segment-panel reset-list"),
				Iff(params.Settings != nil, func() Node { return evalSettings(params.Settings) }),
				Button(Class("btn btn-red mt-2"), Attr("onclick", "reevaluateSubs()"), Text("Reevaluare submisii")),
			),
			Div(Class("segment-panel"),
				H3(ui.T(ctx, "problem_diagnostics")),
				If(len(params.Diagnostics) > 0, Ul(Map(params.Diagnostics, func(d string) Node { return Li(Text(d)) }))),
				If(len(params.Diagnostics) == 0, P(ui.T(ctx, "no_notices"))),
			),
			If(params.LLMEnabled, Div(Class("segment-panel"),
				H3(ui.T(ctx, "experimentalZone")),
				Form(Class("segment-panel"), ID("translateStatementForm"),
					H4(ui.T(ctx, "translateStatement")),
					Button(Class("btn btn-blue"), Attr("onclick", "translateStatement()"), ui.T(ctx, "button.create")),
				),
				script("general_llm.js", nil),
			)),
		),
	), script("general_bottom.js", nil)}
}

func updateForm(ctx context.Context, pb *kilonova.Problem, maxMemMB float64) Node {
	label := func(children ...Node) Node { return Label(Class("block my-2"), Group(children)) }
	field := func(key string) Node { return Group{Span(Class("form-label"), ui.T(ctx, key), Text(":")), Text(" ")} }
	option := func(value string, selected bool, key string) Node {
		return Option(Value(value), If(selected, Selected()), ui.T(ctx, key))
	}
	return Form(ID("updateProblemForm"), AutoComplete("off"),
		label(field("title"), Input(ID("problemName"), Class("form-input"), Type("text"), Value(pb.Name))),
		label(ui.T(ctx, "author_moved")),
		label(field("source"), Input(ID("sourceCredits"), Class("form-input"), Type("text"), Value(pb.SourceCredits))),
		label(
			Input(ID("consoleInput"), Class("form-checkbox"), Type("checkbox"), If(pb.ConsoleInput, Checked())), Text(" "),
			Span(Class("form-label ml-2"), ui.T(ctx, "consoleInput")),
		),
		label(field("scoring_strategy"),
			Select(ID("scoring_strategy"), Class("form-select"),
				option("max_submission", pb.ScoringStrategy == "max_submission", "max_submission_strat"),
				option("sum_subtasks", pb.ScoringStrategy == "sum_subtasks", "sum_subtasks_strat"),
				option("acm-icpc", pb.ScoringStrategy == "acm-icpc", "acm_icpc_strat"),
			),
		),
		label(field("testName"), Input(ID("testName"), Class("form-input"), Type("text"), Value(pb.TestName))),
		label(field("memoryLimit"),
			Input(ID("memoryLimit"), Type("number"), Class("form-input"), Placeholder("Limită de memorie (total)"), Min("0"),
				Step("0.1"), Max(fmt.Sprint(maxMemMB)), Pattern(`[\d]*`), Value(fmt.Sprint(kbToMB(pb.MemoryLimit))), Required()),
			Text(" "), Span(Class("ml-1 text-xl"), Text("MB")),
		),
		label(field("timeLimit"),
			Input(ID("timeLimit"), Type("number"), Class("form-input"), Placeholder("Limită de timp..."), Min("0"),
				Step("0.01"), Pattern(`[\d]*`), Value(fmt.Sprint(pb.TimeLimit)), Required()),
			Text(" "), Span(Class("ml-1 text-xl"), ui.T(ctx, "seconds")),
		),
		label(field("scorePrecision"),
			Input(ID("scorePrecision"), Class("form-input"), Type("number"), Min("0"), Max("4"), Step("1"), Pattern(`[\d]*`),
				Value(strconv.Itoa(int(pb.ScorePrecision)))),
		),
		Details(
			Summary(ui.T(ctx, "advancedOptions")),
			label(field("defaultPoints"),
				Input(ID("defaultPoints"), Class("form-input"), Type("number"), Min("0"), Max("100"), Step(scoreStep(pb)), Pattern(`[\d]*\.?[\d]*`),
					Value(pb.DefaultPoints.String())),
			),
			label(field("scoreScale"),
				Input(ID("scoreScale"), Class("form-input"), Type("number"), Min("0"), Max("10000"), Step("1"), Pattern(`[\d]*\.?[\d]*`),
					Value(pb.ScoreScale.String())),
			),
			label(field("sourceSize"),
				// 2MB should be a healthy upper limit
				Input(ID("sourceSize"), Class("form-input"), Type("number"), Min("100"), Max("2000000"), Step("1"), Pattern(`[\d]*`),
					Value(strconv.Itoa(pb.SourceSize))),
				Text(" "), Span(Class("ml-1 text-xl"), Text("Bytes")),
			),
			label(field("taskType"),
				Select(ID("taskType"), Class("form-select"),
					option("batch", pb.TaskType == "batch", "taskTypeBatch"),
					option("communication", pb.TaskType == "communication", "taskTypeCommunication"),
					option("ai", pb.TaskType == "ai", "taskTypeAI"),
				),
			),
			label(field("communicationProcesses"),
				Input(ID("communicationProcesses"), Class("form-input"), Type("number"), Min("1"), Max("5"), Step("1"), Pattern(`[\d]*`),
					Value(strconv.Itoa(pb.CommunicationProcesses))),
			),
		),
		label(
			Input(ID("visibleTests"), Class("form-checkbox"), Type("checkbox"), If(pb.VisibleTests, Checked())), Text(" "),
			Span(Class("form-label ml-2"), ui.T(ctx, "visibleTests")),
		),
		Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "updateProblem")),
	)
}

func checklist(ctx context.Context, pb *kilonova.Problem, chk *kilonova.ProblemChecklist, noDiagnostics, isAdmin bool) Node {
	item := func(ok bool, end string, key string, args ...any) Node {
		return Li(booleanExpression(ok), Text(" "), ui.T(ctx, key, args...), Text(end))
	}
	requested := pb.ReviewRequestedAt != nil
	reviewClass := "block my-2 btn"
	if !requested {
		reviewClass += " btn-blue"
	}
	return Div(Class("segment-panel reset-list"),
		H2(ui.T(ctx, "problem_checklist")),
		H3(ui.T(ctx, "checklist_required"), Text(":")),
		Ul(
			item(chk.NumMarkdown != 0, ";", "checklist_md"),
			item(noDiagnostics, ";", "checklist_no_diagnostics"),
			item(chk.NumTests != 0, ";", "checklist_tests", chk.NumTests),
			item(chk.NumAuthorTags != 0, ";", "checklist_author_tags"),
			item(chk.NumSolutions != 0, ".", "checklist_solutions"),
		),
		H3(ui.T(ctx, "checklist_contest_required"), Text(":")),
		Ul(item(chk.NumPDF != 0, ".", "checklist_pdf")),
		H3(ui.T(ctx, "checklist_optional"), Text(":")),
		Ul(
			item(chk.NumSubtasks != 0, ";", "checklist_subtasks", chk.NumSubtasks),
			item(chk.NumOtherTags != 0, ".", "checklist_other_tags"),
		),
		Button(Type("button"), Class(reviewClass), ID("review_request_btn"), If(requested, Disabled()), ui.T(ctx, "request_review")),
		Iff(requested, func() Node {
			return Group{Text(" "), Span(Text("Review requested at: "), ui.ServerTimestamp(*pb.ReviewRequestedAt))}
		}),
		Form(Class("segment-panel"), ID("visibility_form"), AutoComplete("off"),
			H2(ui.T(ctx, "toggle_visibility")),
			If(!isAdmin, H3(ui.T(ctx, "visibility_admin"))),
			Div(Class("block my-2"),
				Label(Class("inline-flex items-center text-lg"),
					Input(Class("form-checkbox"), Type("checkbox"), ID("visibility_check"), If(!isAdmin, Disabled()), If(pb.Visible, Checked())), Text(" "),
					Span(Class("ml-2"), ui.T(ctx, "visible")),
				),
			),
			Button(Type("submit"), Class("btn btn-blue"), If(!isAdmin, Disabled()), ui.T(ctx, "button.update")),
		),
	)
}

func booleanExpression(ok bool) Node {
	if ok {
		return Span(Class("text-lg"), Text("✅"))
	}
	return Span(Class("text-lg"), Text("❌"))
}

func evalSettings(s *kilonova.ProblemEvalSettings) Node {
	orNA := func(vals []string) string {
		if len(vals) == 0 {
			return "N/A"
		}
		return strings.Join(vals, ", ")
	}
	langs := "Toate"
	if len(s.LanguageWhitelist) > 0 {
		langs = "[" + strings.Join(s.LanguageWhitelist, ", ") + "]"
	}
	checker := "Clasic/Default (verifică conținutul fișierului de ieșire)"
	if s.CheckerName != "" {
		checker = "Custom (este executat " + s.CheckerName + ")"
	}
	return Group{
		H3(Text("Pe baza atașamentelor, aceste informații vor fi transmise evaluatorului:")),
		Ul(
			Li(Text("Limbaje permise: "+langs)),
			Li(Text("Checker: "+checker)),
			Li(Text("Fișiere extra incluse: "+orNA(s.HeaderFiles))),
			Li(Text("Fișiere grader: "+orNA(s.GraderFiles))),
		),
	}
}

func kbToMB(kb int) float64 {
	return math.Round(float64(kb)/1024.0*100) / 100.0
}

// encodeTags matches the old encodeJSON FuncMap helper: base64 of the JSON.
func encodeTags(tags []*kilonova.Tag) string {
	b, _ := json.Marshal(tags)
	return base64.StdEncoding.EncodeToString(b)
}
