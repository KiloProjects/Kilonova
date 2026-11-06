package contests

import (
	"context"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ViewParams is the contest's main page. The sidebar carries the contest, the viewer's
// registration and the contest problems visible to them.
type ViewParams struct {
	Topbar          *problems.TopbarParams
	DescriptionHTML string // rendered markdown, trusted
	Sidebar         SidebarParams
}

// View is the contest's main page: description, problems and sidebar.
func View(ctx context.Context, p ViewParams) Node {
	c := p.Sidebar.Contest
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"),
			Div(Class("page-content-wrapper"),
				If(c.Description != "", Div(Class("segment-panel reset-list enhance-tables statement-content"), Raw(p.DescriptionHTML))),
				Div(Class("segment-panel"), viewProblems(ctx, p.Sidebar)),
			),
			Aside(Class("page-sidebar"), Sidebar(ctx, p.Sidebar)),
		),
	}
}

func viewProblems(ctx context.Context, p SidebarParams) Node {
	c, reg := p.Contest, p.Registration
	switch {
	case len(p.Problems) > 0:
		return Group{
			H2(ui.T(ctx, "problems")),
			problems.ProblemListGroup(ctx, &problems.ProblemListingParams{
				Problems: p.Problems, ShowID: p.IsEditor || c.Ended(), ShowPublished: true, ContestIDScore: &c.ID,
			}),
		}
	case !c.Started():
		return P(ui.T(ctx, "contest_problems_soon"))
	case c.Ended():
		return P(ui.T(ctx, "contest_no_problems"))
	case reg == nil:
		return P(ui.T(ctx, "contest_problems_participants"))
	case usacoStyle(c) && reg.IndividualStartTime == nil:
		return P(ui.T(ctx, "usaco_helper_text", usacoDuration(c)))
	case usacoStyle(c) && reg.IndividualEndTime.Before(time.Now()):
		return P(ui.T(ctx, "usaco_expired_helper"))
	case !usacoStyle(c) && c.RegisterDuringContest:
		return P(ui.T(ctx, "contest_problems_register"))
	}
	return P(ui.T(ctx, "contest_no_problems"))
}

func usacoDuration(c *kilonova.Contest) string {
	return (time.Duration(c.PerUserTime) * time.Second).String()
}

// PageParams is the topbar and contest shared by the small contest pages.
type PageParams struct {
	Topbar   *problems.TopbarParams
	Contest  *kilonova.Contest
	IsEditor bool
}

// Registrations lists the contest's registered users (rendered client-side).
func Registrations(ctx context.Context, p PageParams) Node {
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"),
			Div(Class("page-content-full-wrapper"),
				Div(Class("segment-panel"),
					H2(ui.T(ctx, "contest_registrations")),
					El("kn-contest-registrations", Attr("contestid", strconv.Itoa(p.Contest.ID)),
						Attr("usacomode", strconv.FormatBool(usacoStyle(p.Contest)))),
				),
			),
		),
	}
}

// Leaderboard is the contest leaderboard (rendered client-side).
func Leaderboard(ctx context.Context, p PageParams) Node {
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"),
			Div(Class("page-content-full"),
				H2(ui.T(ctx, "leaderboard")),
				El("kn-leaderboard", Attr("contestid", strconv.Itoa(p.Contest.ID)), Attr("editor", strconv.FormatBool(p.IsEditor))),
			),
		),
	}
}
