package problems

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TopbarParams drives the tab bar shared by problem, problem-edit and contest pages.
// Permission bits are computed by the handler.
type TopbarParams struct {
	IsProblemEditor bool
	IsContestEditor bool
	CanViewTests    bool

	Authed             bool // user is logged in
	CanSubmitInContest bool // only read when Contest != nil
	LeaderboardVisible bool // only read when Contest != nil
	FullyVisible       bool // problem is fully visible to the user

	Contest *kilonova.Contest
	Problem *kilonova.Problem

	URLPrefix string

	Page   string
	PageID int
}

// Topbar renders the header and tab bar of a problem or contest page.
func Topbar(ctx context.Context, t *TopbarParams) Node {
	problemPage, editProblemPage := true, true
	switch t.Page {
	case "contest_general", "contest_edit", "contest_registrations", "contest_communication", "contest_leaderboard":
		problemPage = false
	case "pb_statement", "pb_submissions", "pb_submit":
		editProblemPage = false
	}

	var nav []Node
	if t.Contest != nil {
		nav = append(nav, contestTabs(ctx, t, problemPage)...)
	}
	if problemPage {
		nav = append(nav, problemTabs(ctx, t, editProblemPage)...)
	}

	// Running contests poll Q&A; editors faster. Handled in web/assets/topbar.ts.
	pollMs := ""
	if t.Contest != nil && !t.Contest.Ended() {
		pollMs = "10000"
		if t.IsContestEditor {
			pollMs = "5000"
		}
	}
	return Div(Class("segment-panel"), If(pollMs != "", Data("qna-poll", pollMs)),
		H1(topbarHeading(ctx, t)),
		Nav(Class("mt-1"), spaced(nav)),
		If(problemPage && t.Contest != nil && t.Contest.Ended(), Div(ui.TRaw(ctx, "problem_contest_page_warning"))),
	)
}

func topbarHeading(ctx context.Context, t *TopbarParams) Node {
	contestName := func() string { return t.Contest.Name }
	switch t.Page {
	case "contest_general", "contest_edit":
		return B(Text(contestName()))
	case "contest_registrations":
		return B(Text(contestName()+" | "), ui.T(ctx, "contest_registrations"))
	case "contest_communication":
		return B(Text(contestName()+" | "), ui.T(ctx, "communication"))
	case "contest_leaderboard":
		return B(Text(contestName()+" | "), ui.T(ctx, "leaderboard"))
	case "pb_statement", "pb_submissions", "pb_submit":
		name := t.Problem.Name
		if t.Contest != nil {
			name = t.Contest.Name + " | " + name
		}
		return B(Text(name))
	case "pb_archive":
		return Group{ui.T(ctx, "header.problem_archive"), Text(" "), B(Text(t.Problem.Name))}
	case "pb_statistics":
		return Group{ui.T(ctx, "header.problem_statistics"), Text(" "), B(Text(t.Problem.Name))}
	default:
		return Group{ui.T(ctx, "title.edit.problem"), Text(" "), B(Text(t.Problem.Name))}
	}
}

func contestTabs(ctx context.Context, t *TopbarParams, problemPage bool) []Node {
	c, pre := t.Contest, t.URLPrefix
	tabs := []Node{ui.Tabs(t.Page,
		ui.Tab{Key: "contest_general", Href: pre + "/", Label: ui.T(ctx, "contest_info")},
		ui.Tab{Key: "contest_communication", Href: pre + "/communication", Label: Span(
			ui.T(ctx, "communication"),
			If(!c.Ended() && t.Page != "contest_communication", Group{
				Text(" "),
				El("kn-comm-announcer", Class("inline-block"),
					Attr("contestID", strconv.Itoa(c.ID)), Attr("contestEditor", strconv.FormatBool(t.IsContestEditor))),
			}),
		)},
		ui.Tab{Key: "contest_edit", Href: pre + "/manage/edit", Label: ui.T(ctx, "contest_edit"), Hide: !t.IsContestEditor},
		ui.Tab{Key: "contest_registrations", Href: pre + "/manage/registrations", Label: ui.T(ctx, "contest_registrations"), Hide: !t.IsContestEditor},
		ui.Tab{Key: "contest_leaderboard", Href: pre + "/leaderboard", Label: ui.T(ctx, "leaderboard"), Hide: !t.LeaderboardVisible},
	)}
	if problemPage {
		tabs = append(tabs, ui.TabGroupSeparator())
	}
	return tabs
}

func problemTabs(ctx context.Context, t *TopbarParams, editProblemPage bool) []Node {
	pb := fmt.Sprintf("/problems/%d", t.Problem.ID)
	pre := t.URLPrefix + pb
	tabs := []Node{ui.Tabs(t.Page,
		ui.Tab{Key: "pb_statement", Href: pre + "/", Label: ui.T(ctx, "statement")},
		ui.Tab{Key: "pb_submit", Href: pre + "/submit", Label: ui.T(ctx, "submit"), Hide: !t.Authed},
		ui.Tab{Key: "pb_submissions", Href: pre + "/submissions", Label: ui.T(ctx, "submissions"),
			Hide: t.Contest != nil && !t.Contest.Ended() && !t.CanSubmitInContest},
		ui.Tab{Key: "pb_statistics", Href: pre + "/statistics", Label: ui.T(ctx, "problemStats"),
			Hide:  !t.FullyVisible || t.Contest != nil || !flags.ProblemStatistics.Get(ctx),
			Attrs: Group{Attr("hx-swap", "none"), Attr("hx-get", pb+"/statistics?hx_modal=true")}},
		// Non-editors only reach the archive via a direct link; show the tab while they're on it.
		ui.Tab{Key: "pb_archive", Href: pre + "/archive", Label: ui.T(ctx, "archive"), Hide: t.IsProblemEditor || t.Page != "pb_archive"},
	)}
	if !t.IsProblemEditor {
		return tabs
	}

	// Test and subtask tabs are only selected on their index pages, not on a single test/subtask.
	cur := t.Page
	if (cur == "tests" || cur == "subtasks") && t.PageID != -2 {
		cur = ""
	}
	return append(tabs,
		ui.TabGroupSeparator(),
		ui.CollapsedTabs(ui.T(ctx, "topbar.edit"), editProblemPage, ui.Tabs(cur,
			ui.Tab{Key: "general", Href: pre + "/edit", Label: ui.T(ctx, "general")},
			ui.Tab{Key: "desc", Href: pre + "/edit/desc", Label: ui.T(ctx, "stmt")},
			ui.Tab{Key: "access", Href: pre + "/edit/access", Label: ui.T(ctx, "access_control_short")},
			ui.Tab{Key: "attachments", Href: pre + "/edit/attachments", Label: ui.T(ctx, "attachments")},
			ui.Tab{Key: "tests", Href: pre + "/edit/test", Label: ui.T(ctx, "tests")},
			ui.Tab{Key: "pb_archive", Href: pre + "/archive", Label: ui.T(ctx, "archive")},
			ui.Tab{Key: "subtasks", Href: pre + "/edit/subtasks", Label: ui.T(ctx, "subTasks")},
		)),
	)
}

// spaced puts a space between siblings, as the template's source newlines did.
func spaced(nodes []Node) Group {
	out := make(Group, 0, 2*len(nodes))
	for i, n := range nodes {
		if i > 0 {
			out = append(out, Text(" "))
		}
		out = append(out, n)
	}
	return out
}
