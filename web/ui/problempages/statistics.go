package problempages

import (
	"context"
	"fmt"
	"math"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/dustin/go-humanize"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type StatisticsParams struct {
	Topbar *problems.TopbarParams
	Stats  *sudoapi.ProblemStatistics
}

// StatisticsTitle is the page title of the statistics page.
func StatisticsTitle(ctx context.Context, pb *kilonova.Problem) string {
	return ui.TS(ctx, "header.problem_statistics") + " " + pb.Name
}

// Statistics is the full statistics page.
func Statistics(ctx context.Context, p StatisticsParams) Node {
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"), Div(Class("page-content-full"), statisticsContent(ctx, p.Stats, false))),
	}
}

// StatisticsModal is the statistics page as an htmx out-of-band modal, appended to #modals.
func StatisticsModal(ctx context.Context, p StatisticsParams) Node {
	return Div(Attr("hx-swap-oob", "beforeend:#modals"),
		Dialog(Class("modal-container modal-container-large"),
			Div(Class("modal-header"),
				H1(ui.T(ctx, "header.problem_statistics"), Text(" "), Code(Text(p.Topbar.Problem.Name))),
				Form(Method("dialog"), Button(Type("submit"), I(Class("modal-close fas fa-xmark")))),
			),
			Div(Class("modal-content"), statisticsContent(ctx, p.Stats, true)),
			Div(Class("modal-footer")),
		),
	)
}

func statisticsContent(ctx context.Context, s *sudoapi.ProblemStatistics, inModal bool) Node {
	if s == nil {
		return nil
	}
	heading := H1(ui.T(ctx, "generalStats"))
	if inModal {
		heading = H2(ui.T(ctx, "generalStats"))
	}
	return Group{
		heading,
		P(ui.T(ctx, "numUsersSolved", s.NumSolved)),
		P(ui.T(ctx, "numUsersAttempted", s.NumAttempted)),
		leaderboard(ctx, "timeLeaderboard", "time", s.TimeLeaderboard, func(sub *kilonova.Submission) string {
			if sub.MaxTime > -1 {
				return fmt.Sprintf("%dms", int(math.Floor(sub.MaxTime*1000)))
			}
			return ""
		}),
		leaderboard(ctx, "memoryLeaderboard", "memory", s.MemoryLeaderboard, func(sub *kilonova.Submission) string {
			if sub.MaxMemory > -1 {
				return humanize.IBytes(uint64(sub.MaxMemory) * 1024)
			}
			return ""
		}),
		// The size leaderboard is hidden when even the best entry has no recorded size.
		Iff(s.SizeLeaderboard != nil && len(s.SizeLeaderboard.Submissions) > 0 && s.SizeLeaderboard.Submissions[0].CodeSize > 0, func() Node {
			return leaderboard(ctx, "sizeLeaderboard", "codeSize", s.SizeLeaderboard, func(sub *kilonova.Submission) string {
				if sub.CodeSize > 0 {
					return humanize.IBytes(uint64(sub.CodeSize))
				}
				return ""
			})
		}),
	}
}

// leaderboard lists subs; value formats a submission's measurement, "" meaning none.
func leaderboard(ctx context.Context, titleKey, valueKey string, subs *sudoapi.Submissions, value func(*kilonova.Submission) string) Node {
	if subs == nil || len(subs.Submissions) == 0 {
		return nil
	}
	i := 0
	return Group{
		H2(ui.T(ctx, titleKey)),
		Table(Class("kn-table kn-table-slim"),
			THead(Tr(
				Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "position")),
				Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "name")),
				Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, valueKey)),
			)),
			TBody(Map(subs.Submissions, func(sub *kilonova.Submission) Node {
				i++
				name := Text("????")
				if u, ok := subs.Users[sub.UserID]; ok && u != nil {
					name = A(Href("/profile/"+u.Name), Text(u.Name))
				}
				val := Text("-")
				if v := value(sub); v != "" {
					val = A(Href(fmt.Sprintf("/submissions/%d", sub.ID)), Text(v))
				}
				return Tr(Class("kn-table-row"),
					Td(Class("kn-table-cell"), Textf("%d.", i)),
					Td(Class("kn-table-cell"), name),
					Td(Class("kn-table-cell"), val),
				)
			})),
		),
	}
}
