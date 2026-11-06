package problems

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
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

	return Div(Class("segment-panel"),
		H1(topbarHeading(ctx, t)),
		Nav(ID("topbar_wrapper"), Class("mt-1"), spaced(nav)),
		If(problemPage && t.Contest != nil && t.Contest.Ended(), Group{
			Div(ui.TRaw(ctx, "problem_contest_page_warning")),
			Script(Raw(`
        document.addEventListener("DOMContentLoaded", () => {
            document.getElementById("pb_link_rewrite_a").setAttribute("href", window.location.href.replace(/contests\/[0-9]+\//, ''))
        })
    `)),
		}),
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
	c := t.Contest
	tabs := []Node{
		tab(t, "contest_general", "/", ui.T(ctx, "contest_info")),
		separator(),
		tab(t, "contest_communication", "/communication", Span(
			ui.T(ctx, "communication"),
			If(!c.Ended() && t.Page != "contest_communication", Group{
				Text(" "),
				El("kn-comm-announcer", Class("inline-block"),
					Attr("contestID", strconv.Itoa(c.ID)), Attr("contestEditor", strconv.FormatBool(t.IsContestEditor))),
			}),
		)),
	}
	if t.IsContestEditor {
		tabs = append(tabs,
			separator(), tab(t, "contest_edit", "/manage/edit", ui.T(ctx, "contest_edit")),
			separator(), tab(t, "contest_registrations", "/manage/registrations", ui.T(ctx, "contest_registrations")),
		)
	}
	if t.LeaderboardVisible {
		tabs = append(tabs, separator(), tab(t, "contest_leaderboard", "/leaderboard", ui.T(ctx, "leaderboard")))
	}
	if problemPage {
		tabs = append(tabs, Div(Class("topbar-separator topbar-separator-lg")))
	}
	if !c.Ended() {
		pollTime := "10000; // 10 seconds"
		if t.IsContestEditor {
			pollTime = "5000; // 5 seconds for editors"
		}
		tabs = append(tabs, Script(Raw(`
                (()=>{
                    let pollTime = `+pollTime+`
                    bundled.startReloadingQnA(pollTime); // Trigger reload at specified interval
                })()
            `)))
	}
	return tabs
}

func problemTabs(ctx context.Context, t *TopbarParams, editProblemPage bool) []Node {
	pb := fmt.Sprintf("/problems/%d", t.Problem.ID)
	tabs := []Node{tab(t, "pb_statement", pb+"/", ui.T(ctx, "statement"))}
	if t.Authed {
		tabs = append(tabs, separator(), tab(t, "pb_submit", pb+"/submit", ui.T(ctx, "submit")))
	}
	if t.Contest == nil || t.Contest.Ended() || t.CanSubmitInContest {
		tabs = append(tabs, separator(), tab(t, "pb_submissions", pb+"/submissions", ui.T(ctx, "submissions")))
	}
	if t.FullyVisible && t.Contest == nil && flags.ProblemStatistics.Value() {
		tabs = append(tabs, separator(), A(
			Classes{"p-1": true, "topbar-selected": t.Page == "pb_statistics"},
			Attr("hx-swap", "none"), Attr("hx-get", pb+"/statistics?hx_modal=true"),
			Href(t.URLPrefix+pb+"/statistics"),
			ui.T(ctx, "problemStats"),
		))
	}

	if !t.IsProblemEditor {
		if t.Page == "pb_archive" {
			tabs = append(tabs, separator(), A(Class("p-1 topbar-selected"), Href(t.URLPrefix+pb+"/archive"), ui.T(ctx, "archive")))
		}
		return tabs
	}

	editTab := func(selected bool, path, key string) Node {
		return A(Classes{"p-1": true, "topbar-selected": selected}, Href(t.URLPrefix+pb+path), ui.T(ctx, key))
	}
	return append(tabs,
		Div(Class("topbar-separator topbar-separator-lg")),
		Span(ID("problem_topbar_wrapper"),
			A(ID("topbar_edit_toggle"), Classes{"p-1": true, "hidden": editProblemPage}, Href("#"), ui.T(ctx, "topbar.edit")),
			Text(" "),
			Span(ID("pb_edit_context"), Classes{"hidden": !editProblemPage}, spaced([]Node{
				editTab(t.Page == "general", "/edit", "general"),
				separator(),
				editTab(t.Page == "desc", "/edit/desc", "stmt"),
				separator(),
				editTab(t.Page == "access", "/edit/access", "access_control_short"),
				separator(),
				editTab(t.Page == "attachments", "/edit/attachments", "attachments"),
				separator(),
				editTab(t.Page == "tests" && t.PageID == -2, "/edit/test", "tests"),
				separator(),
				editTab(t.Page == "pb_archive", "/archive", "archive"),
				separator(),
				editTab(t.Page == "subtasks" && t.PageID == -2, "/edit/subtasks", "subTasks"),
			})),
		),
		Script(Raw(`
            (() => {
                let force_open = `+strconv.FormatBool(editProblemPage)+`;

                if(force_open === false) {
                    document.getElementById("topbar_edit_toggle")?.addEventListener("click", (e) => {
                        force_open = true;
                        e.preventDefault();
                        document.getElementById("topbar_edit_toggle").classList.toggle("hidden", force_open)
                        document.getElementById("pb_edit_context").classList.toggle("hidden", !force_open)
                    });
                    document.getElementById("problem_topbar_wrapper").addEventListener("mouseenter", (e) => {
                        document.getElementById("topbar_edit_toggle").classList.toggle("hidden", true)
                        document.getElementById("pb_edit_context").classList.toggle("hidden", false)
                    })
                    document.getElementById("topbar_wrapper").addEventListener("mouseleave", (e) => {
                        document.getElementById("topbar_edit_toggle").classList.toggle("hidden", force_open)
                        document.getElementById("pb_edit_context").classList.toggle("hidden", !force_open)
                    })
                }
            })()
        `)),
	)
}

func tab(t *TopbarParams, page, path string, label Node) Node {
	return A(Classes{"p-1": true, "topbar-selected": t.Page == page}, Href(t.URLPrefix+path), label)
}

func separator() Node { return Div(Class("topbar-separator")) }

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
