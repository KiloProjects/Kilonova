package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/problems"
)

// Data loaders for fragments shared by several pages.

func (rt *Web) contestInfo(r *http.Request, c *kilonova.Contest) contests.Info {
	u := user.UserBrief(r)
	info := contests.Info{
		Contest:  c,
		Authed:   u != nil,
		IsEditor: c.IsEditor(u),
		CanJoin:  rt.base.CanJoinContest(c),
		ReqPath:  reqPath(r),
	}
	if u != nil {
		reg, err := rt.base.ContestRegistration(r.Context(), c.ID, u.ID)
		if err != nil {
			if !errors.Is(err, kilonova.ErrNotFound) {
				slog.WarnContext(r.Context(), "Couldn't get contest registration", slog.Any("err", err))
			}
			reg = nil
		}
		info.Registration = reg
	}
	return info
}

func (rt *Web) contestSidebar(r *http.Request, c *kilonova.Contest) contests.SidebarParams {
	pbs, err := rt.base.ContestProblems(r.Context(), c, user.UserBrief(r))
	if err != nil {
		pbs = nil
	}
	return contests.SidebarParams{Info: rt.contestInfo(r, c), Problems: pbs, CurrentProblem: util.Problem(r)}
}

func (rt *Web) pblistGroup(r *http.Request, list *kilonova.ProblemList, open bool) problems.ListGroupParams {
	ctx, u := r.Context(), user.UserBrief(r)
	p := problems.ListGroupParams{List: list, Open: open, Authed: u != nil}
	if u != nil {
		cache, _ := ctx.Value(PblistCntCacheKey).(map[int]int)
		p.NumSolved = make(map[int]int, len(list.SubLists)+1)
		ids := []int{list.ID}
		for _, sl := range list.SubLists {
			ids = append(ids, sl.ID)
		}
		for _, id := range ids {
			if cnt, ok := cache[id]; ok {
				p.NumSolved[id] = cnt
				continue
			}
			slog.WarnContext(ctx, "Cache miss", slog.Int("listID", id), slog.String("page", r.URL.Path))
			cnt, err := rt.base.NumSolvedFromPblist(ctx, id, u.ID)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get problem list number solved", slog.Any("err", err))
				cnt = -1
			}
			p.NumSolved[id] = cnt
		}
	}
	if list.Description != "" {
		desc, err := rt.base.RenderMarkdown([]byte(list.Description), nil)
		if err != nil {
			slog.WarnContext(ctx, "Error rendering markdown", slog.Any("err", err))
			desc = []byte("[Error rendering markdown]")
		}
		p.DescriptionHTML = string(desc)
	}
	pbs, err := rt.base.ProblemListProblems(ctx, list.List, u)
	if err == nil {
		p.Problems = pbs
	}
	return p
}
