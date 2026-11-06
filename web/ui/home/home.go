// Package home renders the front page.
package home

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// Params holds everything the front page shows; the handler loads it and reads the flags.
// Fields named *HTML are trusted, rendered markdown keyed by translation key in MD.
type Params struct {
	Version string
	Authed  bool

	AnnouncementHTML string // rendered custom announcement; empty uses the default translation
	ShowAnnouncement bool
	FeedbackHTML     string // rendered feedback call to arms; empty hides the panel

	RunningContests []contests.Info
	FutureContests  []contests.Info
	ContestSidebar  bool // flag: contests go in the sidebar on large screens

	MainPageLogin bool
	SignupEnabled bool

	ShowProblems      bool
	ChildProblemLists []problems.ListGroupParams
	OtherProblems     []*kilonova.ScoredProblem // problems in no list
	ShowPbLinks       bool
	ShowRandomPb      bool

	ShowLatest         bool
	LatestProblems     []*sudoapi.FullProblem
	MoreLatestProblems bool

	Announcements []Node
	QuickSearch   bool

	PinnedLists     []*kilonova.ProblemList
	PinnedNumSolved map[int]int

	ShowTrending    bool
	HotProblems     []*kilonova.ScoredProblem
	MoreHotProblems bool

	DonationNag bool

	// MD maps the translation keys rendered as markdown to their HTML.
	MD map[string]string
}

// MarkdownKeys are the translation keys the page renders through markdown.
var MarkdownKeys = []string{
	"announcements.all_contests", "announcements.all_checklists", "announcements.all_problems",
	"random_problem_announcement", "random_unsolved_problem_announcement", "announcements.latest_problems",
}

// Index is the front page.
func Index(ctx context.Context, p Params) Node {
	md := func(key string) Node { return Raw(p.MD[key]) }
	return Group{
		Div(Class("segment-panel"),
			H1(Text("Kilonova "+p.Version)),
			If(p.ShowAnnouncement, P(Class("text-gray-600 dark:text-gray-300 mb-1"),
				If(p.AnnouncementHTML == "", ui.TRaw(ctx, "announcement")),
				Raw(p.AnnouncementHTML),
			)),
		),
		Div(Class("page-holder"),
			Div(Class("page-content-wrapper"),
				If(p.FeedbackHTML != "", Div(Class("segment-panel"),
					H2(ui.T(ctx, "feedbackHeader")),
					Div(Class("mb-1"), Raw(p.FeedbackHTML), Text(" ")),
				)),
				contestsPanel(ctx, p, md),
				If(p.MainPageLogin && !p.Authed, loginForm(ctx, p.SignupEnabled)),
				problemsPanel(ctx, p, md),
				If(p.ShowLatest && len(p.LatestProblems) > 0, Div(Class("segment-panel"),
					H2(ui.T(ctx, "latestPbs")),
					El("kn-pb-latest", Attr("enc", ui.EncodeJSON(p.LatestProblems))),
					If(p.MoreLatestProblems, Div(Class("my-2"), md("announcements.latest_problems"))),
				)),
			),
			Aside(Class("page-sidebar"),
				Group(p.Announcements),
				If(p.QuickSearch, Form(Class("segment-panel"), Action("/problems"),
					Label(
						H2(ui.T(ctx, "quick_search")),
						Input(Class("form-input w-full mb-2"), Type("text"), Name("q"), Placeholder(ui.TS(ctx, "name_or_id"))),
					),
					Button(Class("btn btn-blue"), ui.T(ctx, "search_pb")),
				)),
				If(p.ContestSidebar && len(p.RunningContests)+len(p.FutureContests) > 0, Div(Class("segment-panel hidden lg:block"),
					If(len(p.RunningContests) > 0, Group{H2(ui.T(ctx, "running_contests")), sidebarContests(ctx, p.RunningContests)}),
					If(len(p.FutureContests) > 0, Group{H2(ui.T(ctx, "future_contests")), sidebarContests(ctx, p.FutureContests)}),
					Div(Class("mt-3"), md("announcements.all_contests")),
				)),
				If(len(p.PinnedLists) > 0, pinnedLists(ctx, p)),
				If(p.ShowTrending && len(p.HotProblems) > 0, Div(Class("segment-panel"),
					H2(ui.T(ctx, "hot_problems")),
					problems.ProblemListGroup(ctx, &problems.ProblemListingParams{Problems: p.HotProblems, ShowID: true}),
					If(p.MoreHotProblems, P(Class("mt-2"), A(Href("/problems/?ordering=hot&descending=true"), ui.T(ctx, "view_more")))),
				)),
				If(p.DonationNag, Div(Class("segment-panel"),
					H2(ui.T(ctx, "support_kn"), Text(" "), I(Class("fas fa-hand-holding-dollar"))),
					P(Class("text-sm text-muted"), ui.TRaw(ctx, "donate.front_page_par")),
				)),
			),
		),
	}
}

func contestsPanel(ctx context.Context, p Params, md func(string) Node) Node {
	if len(p.RunningContests)+len(p.FutureContests) == 0 {
		return nil
	}
	grid := func(cs []contests.Info) Node {
		return Div(Class("contest-grid"), Map(cs, func(c contests.Info) Node { return contests.Brief(ctx, c) }))
	}
	return Div(Classes{"segment-panel": true, "lg:hidden": p.ContestSidebar},
		If(len(p.RunningContests) > 0, Group{H2(ui.T(ctx, "running_contests")), grid(p.RunningContests)}),
		If(len(p.RunningContests) > 0 && len(p.FutureContests) > 0, Div(Class("my-2"))),
		If(len(p.FutureContests) > 0, Group{H2(ui.T(ctx, "future_contests")), grid(p.FutureContests)}),
		Div(Class("mt-3"), md("announcements.all_contests")),
	)
}

func loginForm(ctx context.Context, signup bool) Node {
	return Group{
		Form(Class("segment-panel"), ID("login_form"),
			H1(Class("mb-4"), ui.T(ctx, "authenticate")),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "username_email")),
				Input(Class("form-input w-full"), Type("text"), ID("login_uname"), Name("username")),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "password")),
				Input(Class("form-input w-full"), Type("password"), ID("login_upwd"), Name("password")),
			),
			Button(Class("block btn btn-blue mb-3"), ui.T(ctx, "auth.login")),
			If(signup, P(Class("text-gray-600 dark:text-gray-300"), ui.TRaw(ctx, "signupReminder"))),
			P(Class("text-gray-600 dark:text-gray-300"), ui.TRaw(ctx, "forgot_pwd_announcement")),
		),
		Script(Raw(loginScript)),
	}
}

const loginScript = `
document.getElementById("login_form").addEventListener("submit", login)
async function login(e) {
	e.preventDefault()
	let username = document.getElementById("login_uname").value;
	let password = document.getElementById("login_upwd").value;

	let res = await bundled.postCall("/auth/login", {username, password})
	if(res.status == "error") {
		bundled.apiToast(res)
		return
	}
	bundled.setSession(res.data)
	console.log(window.location.pathname)
	if(window.location.pathname.startsWith("/login")) {
		bundled.navigateBack()
	} else {
		window.location.reload()
	}
}
`

func problemsPanel(ctx context.Context, p Params, md func(string) Node) Node {
	if !p.ShowProblems || len(p.OtherProblems)+len(p.ChildProblemLists) == 0 {
		return nil
	}
	onlyOthers := len(p.ChildProblemLists) == 0
	restName := "the_rest"
	if onlyOthers {
		restName = "problems"
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "problems")),
		Div(Class("list-group mb-4"),
			Map(p.ChildProblemLists, func(l problems.ListGroupParams) Node { return problems.ListGroup(ctx, l) }),
			If(len(p.OtherProblems) > 0, Details(Class("list-group-head"), If(onlyOthers, Open()),
				Summary(Class("pb-1 mt-1"),
					Span(ui.T(ctx, restName)), Text(" "),
					Span(Class("float-right badge"), problemCount(ctx, p.Authed, p.OtherProblems)),
				),
				Div(Class("mt-2"), problems.ProblemListGroup(ctx, &problems.ProblemListingParams{Problems: p.OtherProblems, ShowID: true, ShowPublished: true})),
			)),
		),
		If(p.ShowPbLinks, Div(Class("mt-2"), md("announcements.all_checklists"), md("announcements.all_problems"))),
		If(p.ShowRandomPb, Div(Class("mt-2"), md("random_problem_announcement"), md("random_unsolved_problem_announcement"))),
	)
}

func problemCount(ctx context.Context, authed bool, pbs []*kilonova.ScoredProblem) Node {
	switch {
	case authed:
		return ui.T(ctx, "num_solved", NumSolved(pbs), len(pbs))
	case len(pbs) == 1:
		return ui.T(ctx, "single_problem")
	default:
		return ui.T(ctx, "num_problems", len(pbs))
	}
}

// NumSolved counts the problems with a full score.
func NumSolved(pbs []*kilonova.ScoredProblem) int {
	var cnt int
	hundred := decimal.NewFromInt(100)
	for _, pb := range pbs {
		if pb.MaxScore != nil && pb.MaxScore.Equal(hundred) {
			cnt++
		}
	}
	return cnt
}

func pinnedLists(ctx context.Context, p Params) Node {
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "pinned_lists")),
		Div(Class("list-group"), Map(p.PinnedLists, func(l *kilonova.ProblemList) Node {
			var badge Node
			switch {
			case l.NumProblems <= 0:
			case p.Authed:
				badge = Textf("%d / %d", p.PinnedNumSolved[l.ID], l.NumProblems)
			case l.NumProblems == 1:
				badge = ui.T(ctx, "single_problem")
			default:
				badge = ui.T(ctx, "num_problems", l.NumProblems)
			}
			return A(Href(fmt.Sprintf("/problem_lists/%d", l.ID)), Class("list-group-item flex justify-between"),
				Span(Class("pb-1 mt-1"), Textf("%s (#%d)", l.Title, l.ID)),
				If(badge != nil, Span(Class("badge mt-1 ml-1 whitespace-nowrap"), badge)),
			)
		})),
	)
}

// sidebarContests is the compact contest list in the front page sidebar.
func sidebarContests(ctx context.Context, cs []contests.Info) Node {
	return Map(cs, func(p contests.Info) Node {
		c := p.Contest
		var status Node
		switch {
		case c.Ended():
			status = ui.T(ctx, "contest_ended")
		case c.Running():
			status = Group{countdown(remainingTime(c, p.Registration), "running"), Text(" "), ui.T(ctx, "contest_remaining")}
		default:
			status = Group{ui.T(ctx, "contest_starting_in"), Text(" "), countdown(c.StartTime, "before_start")}
		}
		return Div(Class("segment-panel my-1"),
			H2(A(Href(fmt.Sprintf("/contests/%d", c.ID)), Text(c.Name))),
			If(c.Type == "virtual" || p.IsEditor, P(ui.T(ctx, "contest_type."+string(c.Type)))),
			P(status, Text(" "), registerURL(ctx, p)),
			If(c.PerUserTime > 0, P(ui.T(ctx, "contest_user_time"), Text(": "+(time.Duration(c.PerUserTime)*time.Second).String()))),
			If(p.IsEditor, Group{
				P(ui.T(ctx, "visibility"), Text(": "), badge(ctx, c.Visible, "visible", "invisible")),
				P(ui.T(ctx, "registration_status"), Text(": "), badge(ctx, c.PublicJoin, "registration_open", "registration_closed")),
			}),
		)
	})
}

func registerURL(ctx context.Context, p contests.Info) Node {
	c, reg := p.Contest, p.Registration
	jsLink := func(fn, key string) Node {
		return A(Href("#"), Attr("onclick", fmt.Sprintf("event.preventDefault(), bundled.%s(%d)", fn, c.ID)), ui.T(ctx, key))
	}
	var n Node
	switch {
	case !p.Authed && p.CanJoin:
		n = Group{A(Href("/login?back="+url.QueryEscape(p.ReqPath)), ui.T(ctx, "register_login_anchor")), Text(" "), ui.T(ctx, "register_login_text")}
	case !p.Authed:
	case reg == nil && p.CanJoin:
		n = jsLink("registerForContest", "register_btn")
	case reg == nil:
	case !c.Running() || c.PerUserTime == 0:
		n = Span(ui.T(ctx, "registered"))
	case reg.IndividualStartTime == nil:
		n = jsLink("startContestRegistration", "start_contest")
	case reg.IndividualEndTime.Before(time.Now()):
		n = Span(ui.T(ctx, "time_elapsed"))
	default:
		n = Span(ui.T(ctx, "started"))
	}
	if n == nil {
		return nil
	}
	return Group{Text("- "), n}
}

// remainingTime is when the contest ends for this user: their individual end time in a
// started USACO-style contest, otherwise the contest's end.
func remainingTime(c *kilonova.Contest, reg *kilonova.ContestRegistration) time.Time {
	if c.PerUserTime == 0 || reg == nil || reg.IndividualStartTime == nil {
		return c.EndTime
	}
	if time.Now().Before(*reg.IndividualEndTime) {
		return *reg.IndividualEndTime
	}
	return c.EndTime
}

func countdown(t time.Time, typ string) Node {
	return El("kn-contest-countdown", Attr("target_time", strconv.FormatInt(t.UnixMilli(), 10)), Attr("type", typ))
}

func badge(ctx context.Context, ok bool, yes, no string) Node {
	if ok {
		return Span(Class("badge-lite bg-green-700 text-sm font-semibold"), ui.T(ctx, yes))
	}
	return Span(Class("badge-lite bg-red-700 text-sm font-semibold"), ui.T(ctx, no))
}
