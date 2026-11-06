package problems

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ListGroupParams is a problem list rendered as a collapsible group. The handler
// loads everything: NumSolved per list ID (only read when Authed), the rendered
// description, and the list's own problems.
type ListGroupParams struct {
	List            *kilonova.ProblemList
	Open            bool
	Authed          bool
	NumSolved       map[int]int
	DescriptionHTML string // trusted, rendered markdown
	Problems        []*kilonova.ScoredProblem
}

// ListGroup renders a problem list with its description, lazily-loaded sublists
// (fetched with htmx on first open) and problems.
func ListGroup(ctx context.Context, p ListGroupParams) Node {
	l := p.List
	return Details(Class("list-group-head"), If(p.Open, Open()),
		Summary(Class("pb-1 mt-1"), listSummary(ctx, p, l.ID, l.Title, l.NumProblems)),
		If(l.Description != "", Div(Class("list-group mt-2"),
			Div(Class("list-group-head statement-content"), Raw(p.DescriptionHTML)),
		)),
		If(len(l.SubLists) > 0, Div(Class("list-group mt-2"),
			Map(l.SubLists, func(sl *kilonova.ShallowProblemList) Node {
				href := fmt.Sprintf("/problem_lists/%d", sl.ID)
				return Details(Class("list-group-head"),
					Attr("hx-get", href), Attr("hx-swap", "outerHTML"), Attr("hx-trigger", "toggle once"),
					Summary(Class("pb-1 mt-1"), listSummary(ctx, p, sl.ID, sl.Title, sl.NumProblems)),
					NoScript(A(Href(href), ui.T(ctx, "sublist_noscript")), Text(" ")),
					Div(Class("text-4xl mx-auto w-full my-10 text-center"),
						Div(I(Class("fas fa-spinner animate-spin")), Text(" "), ui.T(ctx, "loading")),
					),
				)
			}),
		)),
		Iff(len(p.Problems) > 0, func() Node {
			id := l.ID
			return Div(Class("mt-2"), ProblemListGroup(ctx, &ProblemListingParams{
				Problems: p.Problems, ShowID: true, ShowPublished: true, ListID: &id,
			}))
		}),
	)
}

func listSummary(ctx context.Context, p ListGroupParams, id int, title string, numProblems int) Node {
	var badge Node
	switch {
	case numProblems <= 0:
	case p.Authed:
		solved, ok := p.NumSolved[id]
		if !ok {
			solved = -1
		}
		badge = ui.T(ctx, "num_solved", solved, numProblems)
	case numProblems == 1:
		badge = ui.T(ctx, "single_problem")
	default:
		badge = ui.T(ctx, "num_problems", numProblems)
	}
	return Group{
		Span(Text(title+" "), A(Href(fmt.Sprintf("/problem_lists/%d", id)), Textf("(#%d)", id))),
		If(badge != nil, Group{Text(" "), Span(Class("float-right badge"), badge)}),
	}
}
