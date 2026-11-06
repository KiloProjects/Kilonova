// Package contests renders contest fragments shared across pages.
package contests

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Info is a contest plus what the viewer may do with it, loaded by the handler.
type Info struct {
	Contest      *kilonova.Contest
	Registration *kilonova.ContestRegistration // nil when not registered or anonymous
	Authed       bool
	IsEditor     bool
	CanJoin      bool
	ReqPath      string // where the login link sends the user back to
}

// SidebarParams adds the problem list shown in the contest sidebar.
type SidebarParams struct {
	Info
	Problems []*kilonova.ScoredProblem
	// CurrentProblem is set on problem pages: the editor list is hidden and the problem highlighted.
	CurrentProblem *kilonova.Problem
}

// Brief is the contest card shown in contest lists.
func Brief(ctx context.Context, p Info) Node {
	c := p.Contest
	return Div(Class("segment-panel my-1"),
		H2(A(Href(fmt.Sprintf("/contests/%d", c.ID)), Text(c.Name))),
		If(c.Type == "virtual" || p.IsEditor, P(ui.T(ctx, "contest_type."+string(c.Type)))),
		commonInfo(ctx, p),
		Iff(p.CanJoin, func() Node {
			switch {
			case !p.Authed:
				return loginToRegister(ctx, p.ReqPath)
			case p.Registration == nil:
				return registerButton(ctx, c.ID)
			case !c.Running():
				return Span(Class("my-2"), ui.T(ctx, "registered"))
			case usacoStyle(c) && p.Registration.IndividualStartTime == nil:
				return startButton(ctx, c.ID, "btn btn-blue my-2")
			}
			return nil
		}),
	)
}

// Sidebar is the contest panel next to contest and contest-problem pages, followed by
// the contest's problems when there are any.
func Sidebar(ctx context.Context, p SidebarParams) Node {
	c := p.Contest
	return Group{
		Div(Class("segment-panel"),
			H2(Class("font-normal"), ui.T(ctx, "contestInfo")),
			Div(Class("px-2"),
				P(ui.T(ctx, "contest_type."+string(c.Type))),
				commonInfo(ctx, p.Info, If(p.CurrentProblem == nil && len(c.Editors) > 0,
					Details(Class("reset-list"), Open(),
						Summary(ui.T(ctx, "organisedBy"), Text(":")),
						Ul(Map(c.Editors, func(u *kilonova.UserBrief) Node {
							return Li(A(Href("/profile/"+u.Name), Text(u.Name)))
						})),
					),
				)),
				Iff(p.CanJoin, func() Node { return sidebarRegistration(ctx, p.Info) }),
			),
		),
		If(len(p.Problems) > 0, Div(Class("segment-panel"),
			H2(ui.T(ctx, "problems")),
			Div(Class("reset-list text-lg"),
				Ul(Map(p.Problems, func(pb *kilonova.ScoredProblem) Node {
					return sidebarProblem(c, pb, p.CurrentProblem)
				})),
			),
		)),
	}
}

// commonInfo is the start time, duration, status and editor-only badges; extra goes
// between the duration and the status, as in the sidebar's organiser list.
func commonInfo(ctx context.Context, p Info, extra ...Node) Node {
	c := p.Contest
	return Group{
		P(ui.T(ctx, "startTime"), Text(": "), ui.ServerTimestamp(c.StartTime)),
		P(ui.T(ctx, "duration"), Text(": "+c.EndTime.Sub(c.StartTime).Round(time.Minute).String())),
		Group(extra),
		P(ui.T(ctx, "status"), Text(": "), status(ctx, c, p.Registration)),
		If(usacoStyle(c), P(ui.T(ctx, "contest_user_time"), Text(": "+(time.Duration(c.PerUserTime)*time.Second).String()))),
		If(p.IsEditor, Group{
			P(Class("my-1"), ui.T(ctx, "visibility"), Text(": "),
				badge(ctx, c.Visible, "visible", "invisible")),
			P(Class("my-1"), ui.T(ctx, "registration_status"), Text(": "),
				badge(ctx, c.PublicJoin, "registration_open", "registration_closed")),
		}),
	}
}

func status(ctx context.Context, c *kilonova.Contest, reg *kilonova.ContestRegistration) Node {
	switch {
	case c.Ended():
		return ui.T(ctx, "contest_ended")
	case c.Running():
		return Group{countdown(remainingTime(c, reg), "running"), Text(" "), ui.T(ctx, "contest_remaining")}
	default:
		return Group{ui.T(ctx, "contest_starting_in"), Text(" "), countdown(c.StartTime, "before_start")}
	}
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

func usacoStyle(c *kilonova.Contest) bool { return c.PerUserTime > 0 }

func countdown(t time.Time, typ string) Node {
	return El("kn-contest-countdown", Attr("target_time", strconv.FormatInt(t.UnixMilli(), 10)), Attr("type", typ))
}

func badge(ctx context.Context, ok bool, yes, no string) Node {
	if ok {
		return Span(Class("badge-lite bg-green-700 text-sm font-semibold"), ui.T(ctx, yes))
	}
	return Span(Class("badge-lite bg-red-700 text-sm font-semibold"), ui.T(ctx, no))
}

// BUG (kept from the template): if registration during contest is disabled, the USACO start button will not show up.
func sidebarRegistration(ctx context.Context, p Info) Node {
	c, reg := p.Contest, p.Registration
	switch {
	case !p.Authed:
		return loginToRegister(ctx, p.ReqPath)
	case reg == nil:
		return registerButton(ctx, c.ID)
	}
	var state Node
	switch {
	case !c.Running() || !usacoStyle(c):
		state = Span(Class("badge-lite"), ui.T(ctx, "registered"))
	case reg.IndividualStartTime == nil:
		state = startButton(ctx, c.ID, "btn btn-blue")
	case reg.IndividualEndTime.Before(time.Now()):
		state = Span(Class("badge-lite"), ui.T(ctx, "time_elapsed"))
	default:
		state = Span(Class("badge-lite"), ui.T(ctx, "started"))
	}
	return Div(Class("my-2"), state)
}

func registerButton(ctx context.Context, id int) Node {
	return Button(Class("btn btn-blue my-2"), Attr("onclick", fmt.Sprintf("bundled.registerForContest(%d)", id)), ui.T(ctx, "register_btn"))
}

func startButton(ctx context.Context, id int, class string) Node {
	return Button(Class(class), Attr("onclick", fmt.Sprintf("bundled.startContestRegistration(%d)", id)), ui.T(ctx, "start_contest"))
}

func loginToRegister(ctx context.Context, reqPath string) Node {
	return Span(Class("my-2"),
		A(Href("/login?back="+url.QueryEscape(reqPath)), ui.T(ctx, "register_login_anchor")),
		Text(" "), ui.T(ctx, "register_login_text"),
	)
}

func sidebarProblem(c *kilonova.Contest, pb *kilonova.ScoredProblem, current *kilonova.Problem) Node {
	var score Node
	if pb.ScoreUserID != nil {
		score = ui.Score(&pb.Problem, pb.MaxScore, true)
		if pb.ScoringStrategy == kilonova.ScoringTypeSumSubtasks {
			score = A(Class("max_score_breakdown"), Href("#"),
				Data("problemid", strconv.Itoa(pb.ID)), Data("contestid", strconv.Itoa(c.ID)), score)
		}
	}
	return Li(
		A(If(current != nil && current.ID == pb.ID, Class("topbar-selected p-0")),
			Href(fmt.Sprintf("/contests/%d/problems/%d", c.ID, pb.ID)), Text(pb.Name)),
		If(score != nil, Group{Text(" ("), score, Text(")")}),
	)
}
