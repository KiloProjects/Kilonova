// Package modals renders fragments shown in modals, sidebars and banners.
package modals

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type OlderSubmissionsParams struct {
	UserID  int
	Problem *kilonova.Problem
	Contest *kilonova.Contest
	Limit   int

	Submissions *sudoapi.Submissions
	NumHidden   int
	AllFinished bool
	AutoReload  bool
}

// OlderSubmissions lists a user's latest submissions to a problem. While any are still
// being graded and AutoReload is set, htmx re-fetches it every second.
func OlderSubmissions(ctx context.Context, params *OlderSubmissionsParams) Node {
	url := fmt.Sprintf("/problems/%d/submissions/?user_id=%d", params.Problem.ID, params.UserID)
	if params.Contest != nil {
		url = fmt.Sprintf("/contests/%d/problems/%d/submissions/?user_id=%d", params.Contest.ID, params.Problem.ID, params.UserID)
	}
	trigger := "kn-poll"
	if !params.AllFinished && params.AutoReload {
		trigger += ",every 1s"
	}
	subs := params.Submissions.Submissions

	return Details(Attr("open"),
		Summary(H2(Class("inline-block mb-2"), ui.T(ctx, "oldSubs"))),
		Div(ID("older_subs"), Attr("hx-select", "#older_subs"), Attr("hx-get", url), Attr("hx-swap", "outerHTML"), Attr("hx-trigger", trigger),
			Map(subs, func(sub *kilonova.Submission) Node {
				return A(Href(fmt.Sprintf("/submissions/%d", sub.ID)), Class("black-anchor flex justify-between items-center rounded-sm py-1 px-2 hoverable"),
					Span(Textf("#%d: ", sub.ID), El("server-timestamp", Attr("timestamp", strconv.FormatInt(sub.CreatedAt.UnixMilli(), 10)))),
					Span(Class("badge-lite text-sm"), submissionStatus(params.Problem, sub)),
				)
			}),
			If(len(subs) == 0, P(Class("px-2"), ui.T(ctx, "noSub"))),
			If(params.NumHidden > 0, A(Class("px-2"), Href(url), seeMore(ctx, params.NumHidden))),
		),
	)
}

func submissionStatus(pb *kilonova.Problem, sub *kilonova.Submission) Node {
	switch sub.Status {
	case kilonova.StatusFinished:
		return ui.Score(pb, &sub.Score, false)
	case kilonova.StatusWorking:
		return I(Class("fas fa-cog animate-spin"))
	default:
		return I(Class("fas fa-clock"))
	}
}

// seeMore picks the Romanian plural form ("încă o", "încă 5", "încă 20 de").
func seeMore(ctx context.Context, n int) Node {
	switch {
	case n == 1:
		return ui.T(ctx, "seeOne")
	case n < 20:
		return ui.T(ctx, "seeU20", n)
	default:
		return ui.T(ctx, "seeMany", n)
	}
}

func ProblemSources(ctx context.Context, problemList *kilonova.ProblemList, problems []*kilonova.ScoredProblem) Node {
	return Form(ID("change_sources_panel"), Class("segment-panel reset-list"),
		Attr("hx-post", fmt.Sprintf("/problem_lists/%d/updateSources", problemList.ID)), Attr("hx-swap", "outerHTML"),
		H2(ui.T(ctx, "bulk_update_sources")),
		Ul(
			Map(problems, func(pb *kilonova.ScoredProblem) Node {
				source := Span(Text("-"))
				if pb.SourceCredits != "" {
					source = Code(Text(pb.SourceCredits))
				}
				return Li(A(Href(fmt.Sprintf("/problems/%d", pb.ID)), Text(pb.Name)), Text(": "), source)
			}),
			If(len(problems) == 0, Div(ui.T(ctx, "noPbFound"))),
		),
		Label(Class("block mb-2"),
			Span(Class("form-label"), ui.T(ctx, "new_format"), Text(":")),
			Input(Class("form-input"), ID("new_format"), Name("new_format"), Type("text"), Attr("hx-preserve", "true"),
				Value(ui.TS(ctx, "bulk_update_templ", problemList.Title)), Attr("hx-disabled-elt", "this"), Required()),
		),
		ui.SubmitButton(ui.T(ctx, "button.update")),
	)
}

var gpcEndTime = time.Date(2026, 9, 28, 2, 59, 59, 0, time.UTC)

// GPCAnnouncement is a dismissable banner, shown until gpcEndTime.
func GPCAnnouncement() Node {
	if time.Now().After(gpcEndTime) {
		return nil
	}
	return Section(Data("flashname", "gpc_announcement"), Class("segment-panel hidden"),
		Div(Class("flex justify-between"),
			P(Style("margin: 0.1rem;"),
				I(Class("fas fa-circle-exclamation")), Text(" "),
				Span(Class("text-muted"),
					Text("Dacă eşti o fată pasionată de informatică, te poţi califica la Girls' Programming Camp după Olimpiada Naţională de Informatică! "),
					A(Href("https://sepi.ro/news/rGSJp5?utm_source=kilonova&utm_medium=alert&utm_campaign=gpc"), Text("Află mai multe!")),
				),
			),
			Button(Data("flashdiscard", "gpc_announcement"), I(Class("modal-close fas fa-xmark"))),
		),
	)
}
