package problems

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ContestDisclaimer is a dismissable warning shown on problems that came from a contest.
func ContestDisclaimer(ctx context.Context) Node {
	return Section(Data("flashname", "contest_disclaimer"), Class("segment-panel hidden"),
		Div(Class("modal-header"),
			H2(Style("margin: 0.25rem;"), I(Class("fas fa-triangle-exclamation")), Text(" "), ui.T(ctx, "contestDisclaimerTitle")),
			Button(Data("flashdiscard", "contest_disclaimer"), I(Class("modal-close fas fa-xmark"))),
		),
		Div(Class("px-2 py-2"), ui.T(ctx, "contestDisclaimerBody")),
	)
}

type InfoParams struct {
	Problem  *kilonova.Problem
	IsEditor bool
	Editors  []*kilonova.UserBrief
	Viewers  []*kilonova.UserBrief
	Tags     []*kilonova.Tag
	// If viewing user is not authed, MaxScore should be nil
	MaxScore     *decimal.Decimal
	CanViewTests bool
	Contest      *kilonova.Contest // for URL prefix
}

// InfoSidebar shows a problem's editors, visibility, authors, source, tags and the
// viewer's score. Non-author tags start collapsed unless the viewer solved the problem.
func InfoSidebar(ctx context.Context, params *InfoParams) Node {
	pb := params.Problem
	var authorTags, otherTags []*kilonova.Tag
	for _, tag := range params.Tags {
		if tag.Type == kilonova.TagTypeAuthor {
			authorTags = append(authorTags, tag)
		} else {
			otherTags = append(otherTags, tag)
		}
	}
	tagList := func(tags []*kilonova.Tag) Node {
		return Map(tags, func(t *kilonova.Tag) Node { return Tag(t, false, "text-sm") })
	}
	label := func(key string) Node { return Group{ui.T(ctx, key), Text(": ")} }

	return Section(Class("segment-panel font-semibold"),
		H2(Class("font-normal"), ui.T(ctx, "problemInfo")),
		Div(Class("px-2"),
			P(label("id"), Text(strconv.Itoa(pb.ID))),
			editors(ctx, params),
			If(params.IsEditor && len(params.Viewers) > 0, userList(ctx, "viewers", params.Viewers, false)),
			If(params.IsEditor, P(label("visibility"), visibility(ctx, pb.Visible))),
			If(len(authorTags) > 0, P(label("author"), tagList(authorTags))),
			If(pb.SourceCredits != "", P(label("source"), Text(pb.SourceCredits))),
			If(len(otherTags) > 0, P(Attr("x-data", fmt.Sprintf(`{"open":%t}`, params.MaxScore != nil && params.MaxScore.Equal(decimal.NewFromInt(100)))),
				label("tags"),
				Span(Attr("x-show", "open"), Attr("x-cloak"),
					tagList(otherTags), Text(" "),
					A(Href("#"), Attr("@click.prevent", "open = !open"), ui.T(ctx, "hide")),
				),
				Text(" "),
				A(Href("#"), Attr("@click.prevent", "open = !open"), Attr("x-show", "!open"), Attr("x-cloak"), ui.T(ctx, "view")),
			)),
			If(user.UserBriefContext(ctx).IsAuthed(), P(scoreLabel(ctx, pb), Text(": "), score(params))),
			If(params.CanViewTests, A(Href(archiveURL(params)), ui.T(ctx, "downloadArchive"))),
		),
	)
}

func editors(ctx context.Context, params *InfoParams) Node {
	if len(params.Editors) == 1 {
		return P(ui.T(ctx, "editor"), Text(": "), profileLink(params.Editors[0]))
	}
	return userList(ctx, "editors", params.Editors, params.IsEditor)
}

func userList(ctx context.Context, key string, users []*kilonova.UserBrief, open bool) Node {
	return Details(Class("reset-list"), If(open, Attr("open")),
		Summary(ui.T(ctx, key), Text(":")),
		Ul(Map(users, func(u *kilonova.UserBrief) Node { return Li(profileLink(u)) })),
	)
}

func profileLink(u *kilonova.UserBrief) Node {
	return A(Href("/profile/"+u.Name), Text(u.Name))
}

func visibility(ctx context.Context, visible bool) Node {
	if visible {
		return Span(Class("badge-lite bg-green-700 text-sm"), ui.T(ctx, "published"))
	}
	return Span(Class("badge-lite bg-red-700 text-sm"), ui.T(ctx, "unpublished"))
}

func scoreLabel(ctx context.Context, pb *kilonova.Problem) Node {
	if pb.ScoringStrategy == kilonova.ScoringTypeICPC {
		return ui.T(ctx, "verdict")
	}
	return ui.T(ctx, "score")
}

// score is reloaded client-side via data-pbid-reload; sum-of-subtasks problems also
// link to a per-subtask breakdown.
func score(params *InfoParams) Node {
	id := strconv.Itoa(params.Problem.ID)
	s := Span(Data("pbid-reload", id), ui.Score(params.Problem, params.MaxScore, true))
	if params.Problem.ScoringStrategy == kilonova.ScoringTypeSumSubtasks {
		return A(Class("max_score_breakdown"), Href("#"), Data("problemid", id), s)
	}
	return s
}

func archiveURL(params *InfoParams) string {
	url := fmt.Sprintf("/problems/%d/archive", params.Problem.ID)
	if params.Contest != nil {
		url = fmt.Sprintf("/contests/%d%s", params.Contest.ID, url)
	}
	return url
}
