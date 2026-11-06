// Package problems renders problem tags and problem listings.
package problems

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

var tagTypes = map[kilonova.TagType]string{
	kilonova.TagTypeAuthor:  "tag-author",
	kilonova.TagTypeContest: "tag-contest",
	kilonova.TagTypeMethod:  "tag-method",
	kilonova.TagTypeOther:   "tag-other",
}

// Tag links to the tag's page, coloured by its type; wide adds horizontal margin.
func Tag(tag *kilonova.Tag, wide bool, classes ...string) Node {
	tagType, ok := tagTypes[tag.Type]
	cls := Classes{"tag": true, "mx-1": wide, tagType: ok, "bg-teal-700": !ok}
	for _, c := range classes {
		cls[c] = true
	}
	return A(Href(fmt.Sprintf("/tags/%d", tag.ID)), Span(cls, Text(tag.Name)))
}

type ProblemListingParams struct {
	Problems []*kilonova.ScoredProblem
	ShowID   bool

	ShowPublished bool

	ContestIDScore *int

	ListID *int
}

func ProblemListGroup(ctx context.Context, params *ProblemListingParams) Node {
	showPublished := params.ShowPublished && user.UserBriefContext(ctx).IsAuthed()
	return Div(Class("list-group grid grid-cols-1"),
		Map(params.Problems, func(pb *kilonova.ScoredProblem) Node {
			url := fmt.Sprintf("/problems/%d", pb.ID)
			if params.ContestIDScore != nil {
				url = fmt.Sprintf("/contests/%d%s", *params.ContestIDScore, url)
			}
			if params.ListID != nil {
				url += fmt.Sprintf("?list_id=%d", *params.ListID)
			}
			return A(Href(url), Class("list-group-item flex justify-between"),
				Span(Text(pb.Name+" "), If(params.ShowID, Textf("(#%d)", pb.ID))),
				Div(
					If(showPublished && pb.IsEditor && pb.Visible, Span(Class("badge badge-green"), ui.T(ctx, "published"))),
					If(showPublished && pb.IsEditor && !pb.Visible, Span(Class("badge badge-red"), ui.T(ctx, "unpublished"))),
					If(pb.ScoreUserID != nil, Span(Class("badge"), ui.Score(&pb.Problem, pb.MaxScore, false))),
				),
			)
		}),
	)
}
