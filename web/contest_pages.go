package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	g "maragu.dev/gomponents"
)

func (rt *Web) contestTabs(r *http.Request, page string) contests.TabsParams {
	u := user.UserBrief(r)
	return contests.TabsParams{
		Page:      page,
		Authed:    u != nil,
		CanCreate: flags.NormalUserVirtualContests.Value() || u.IsProposer(),
	}
}

func (rt *Web) contests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := kilonova.ContestFilter{Look: true, LookingUser: user.UserBrief(r)}

		filter.Ordering = r.FormValue("ord")
		asc, err := strconv.ParseBool(r.FormValue("asc"))
		if err != nil {
			asc = false
		}
		filter.Ascending = asc

		if contestant := r.FormValue("contestant"); len(contestant) > 0 {
			userBrief, err := rt.base.UserBriefByName(r.Context(), contestant)
			if err == nil {
				filter.ContestantID = &userBrief.ID
			}
		}

		var pageNum int
		switch v := r.FormValue("page"); v {
		case "virtual", "official", "personal":
			// Compatibility with old URL format (`/contests?page=official`)
			http.Redirect(w, r, fmt.Sprintf("/contests/%s", v), http.StatusPermanentRedirect)
			return
		default:
			// Interpret as page number
			pageNum, err = strconv.Atoi(v)
			if err != nil {
				pageNum = 1
			}
		}

		page := strings.TrimSuffix(r.URL.Path, "/")
		switch page {
		case "/contests":
			// no additional filter
		case "/contests/official":
			filter.Type = kilonova.ContestTypeOfficial
		case "/contests/virtual":
			filter.Type = kilonova.ContestTypeVirtual
		case "/contests/personal":
			if !user.UserBrief(r).IsAuthed() {
				// Important to redirect and return, since we will dereference for ID later
				http.Redirect(w, r, "/contests/official", http.StatusTemporaryRedirect)
				return
			}

			filter.ImportantContests = true
			filter.ImportantContestsUser = user.UserBrief(r)
		default:
			slog.WarnContext(r.Context(), "Unknown page type", slog.String("type", page))
		}

		cnt, err := rt.base.ContestCount(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contest count", slog.Any("err", err))
			cnt = -1
		}

		filter.Limit = 60
		filter.Offset = filter.Limit * (pageNum - 1)

		cs, err := rt.base.Contests(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contests", slog.Any("err", err))
			rt.statusPage(w, r, 400, "Nu am putut obține concursurile")
			return
		}

		numPages := cnt / filter.Limit
		if cnt%filter.Limit > 0 {
			numPages++
		}

		var pagination g.Node
		if numPages > 1 {
			pagination = ui.Paginator(ui.PaginatorConfig{
				Page:       pageNum,
				NumPages:   numPages,
				ShowArrows: true,
			})
		}

		infos := make([]contests.Info, 0, len(cs))
		for _, c := range cs {
			infos = append(infos, rt.contestInfo(r, c))
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "contests"),
			Head:  ui.CanonicalURL("/contests"),
			Content: contests.Index(r.Context(), contests.IndexParams{
				TabsParams: rt.contestTabs(r, page),
				Contests:   infos,
				Pagination: pagination,
			}),
		})
	}
}

func (rt *Web) createContest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(user.UserBrief(r).IsProposer() || flags.NormalUserVirtualContests.Value()) {
			rt.statusPage(w, r, 403, "Nu poți crea concursuri!")
			return
		}
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "createContest"),
			Head:    ui.CanonicalURL("/contests/create"),
			Content: contests.Create(r.Context(), rt.contestTabs(r, "create")),
		})
	}
}

func (rt *Web) contest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := util.Contest(r)
		var desc string
		if c.Description != "" {
			val, err := rt.base.RenderMarkdown([]byte(c.Description), nil)
			if err != nil {
				slog.WarnContext(r.Context(), "Error rendering markdown", slog.Any("err", err))
				val = []byte("[Error rendering markdown]")
			}
			desc = string(val)
		}
		rt.runLayout(w, r, &LayoutParams{
			Title: c.Name,
			Head:  ui.CanonicalURL(fmt.Sprintf("/contests/%d", c.ID)),
			Content: contests.View(r.Context(), contests.ViewParams{
				Topbar:          rt.problemTopbar(r, "contest_general", -1),
				DescriptionHTML: desc,
				Sidebar:         rt.contestSidebar(r, c),
			}),
		})
	}
}

func (rt *Web) contestEdit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, c, u := r.Context(), util.Contest(r), user.UserBrief(r)
		invitations, err := rt.base.ContestInvitations(ctx, c.ID)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get contest invitations", slog.Any("err", err))
			invitations = nil
		}

		mossSubs, err := rt.base.MOSSSubmissions(ctx, c.ID)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get MOSS submissions", slog.Any("err", err))
			mossSubs = nil
		}

		pbs, err := rt.base.ContestProblems(ctx, c, u)
		if err != nil {
			pbs = nil
		}

		inviters := make(map[int]*kilonova.UserBrief)
		for _, inv := range invitations {
			if inv.CreatorID == nil {
				continue
			}
			if _, ok := inviters[*inv.CreatorID]; ok {
				continue
			}
			ub, err := rt.base.UserBrief(ctx, *inv.CreatorID)
			if err != nil {
				ub = nil
			}
			inviters[*inv.CreatorID] = ub
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "contest_edit"),
			Content: contests.Edit(ctx, contests.EditParams{
				Topbar:      rt.problemTopbar(r, "contest_edit", -1),
				Contest:     c,
				Problems:    pbs,
				Invitations: invitations,
				Inviters:    inviters,
				MOSS:        mossSubs,
				IsAdmin:     u.IsAdmin(),
				IsProposer:  u.IsProposer(),
			}),
		})
	}
}

func (rt *Web) contestInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inv, err := rt.base.ContestInvitation(r.Context(), r.PathValue("inviteID"))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contest invitation", slog.Any("err", err))
			rt.statusPage(w, r, 404, "Invite not found")
			return
		}
		contest, err := rt.base.Contest(r.Context(), inv.ContestID)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contest", slog.Any("err", err))
			rt.statusPage(w, r, 500, "Couldn't get invite's contest")
			return
		}

		var invCreator *kilonova.UserBrief
		if inv.CreatorID != nil {
			userBrief, err := rt.base.UserBrief(r.Context(), *inv.CreatorID)
			if err == nil && userBrief != nil {
				invCreator = userBrief
			}
		}

		reg, err := rt.base.ContestRegistration(r.Context(), contest.ID, user.UserBrief(r).ID)
		if err != nil {
			reg = nil
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.invitation", contest.Name),
			Head:  ui.NoRobots(),
			Content: contests.Invite(r.Context(), contests.InviteParams{
				Contest:           contest,
				Invite:            inv,
				Inviter:           invCreator,
				Registration:      reg,
				AlreadyRegistered: reg != nil,
			}),
		})
	}
}

func (rt *Web) contestPage(r *http.Request, topbarPage string) contests.PageParams {
	c := util.Contest(r)
	return contests.PageParams{
		Topbar:   rt.problemTopbar(r, topbarPage, -1),
		Contest:  c,
		IsEditor: c.IsEditor(user.UserBrief(r)),
	}
}

func (rt *Web) contestCommunication() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, c, u := r.Context(), util.Contest(r), user.UserBrief(r)
		p := contests.CommunicationParams{
			PageParams: rt.contestPage(r, "contest_communication"),
			CanSubmit:  rt.base.CanSubmitInContest(u, c),
		}
		var err error
		if p.Announcements, err = rt.base.ContestAnnouncements(ctx, c.ID); err != nil || p.Announcements == nil {
			p.Announcements = []*kilonova.ContestAnnouncement{}
		}
		if p.IsEditor {
			if p.AllQuestions, err = rt.base.ContestQuestions(ctx, c.ID); err != nil || p.AllQuestions == nil {
				p.AllQuestions = []*kilonova.ContestQuestion{}
			}
		}
		if p.CanSubmit && u != nil {
			if p.UserQuestions, err = rt.base.ContestUserQuestions(ctx, c.ID, u.ID); err != nil || p.UserQuestions == nil {
				p.UserQuestions = []*kilonova.ContestQuestion{}
			}
		}
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "communication"),
			Head:    ui.NoRobots(),
			Content: contests.Communication(ctx, p),
		})
	}
}

func (rt *Web) contestRegistrations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "contest_registrations"),
			Content: contests.Registrations(r.Context(), rt.contestPage(r, "contest_registrations")),
		})
	}
}

func (rt *Web) contestLeaderboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This is assumed to be called from a context in which
		// IsContestVisible is already true
		if !(util.Contest(r).PublicLeaderboard || util.Contest(r).IsEditor(user.UserBrief(r))) {
			rt.statusPage(w, r, 400, "You are not allowed to view the leaderboard")
			return
		}
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "leaderboard"),
			Content: contests.Leaderboard(r.Context(), rt.contestPage(r, "contest_leaderboard")),
		})
	}
}
