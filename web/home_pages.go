package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui/home"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	g "maragu.dev/gomponents"
)

func (rt *Web) index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, u := r.Context(), user.UserBrief(r)
		runningContests, err := rt.base.VisibleRunningContests(ctx, u)
		if err != nil {
			runningContests = []*kilonova.Contest{}
		}
		futureContests, err := rt.base.VisibleFutureContests(ctx, u)
		if err != nil {
			futureContests = []*kilonova.Contest{}
		}

		var pblists []*kilonova.ProblemList
		if val := flags.RootProblemList.Value(); val > 0 {
			pblists, err = rt.base.PblistChildrenLists(ctx, val)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get index page problem lists", slog.Any("err", err))
				pblists = []*kilonova.ProblemList{}
			}
		}

		var pinnedLists []*kilonova.ProblemList
		if val := flags.PinnedProblemList.Value(); val > 0 {
			pinnedLists, err = rt.base.PblistChildrenLists(ctx, val)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get pinned problem lists", slog.Any("err", err))
				pinnedLists = []*kilonova.ProblemList{}
			}
		}

		listIDs := []int{}
		for _, list := range pblists {
			listIDs = append(listIDs, list.ID)
			for _, slist := range list.SubLists {
				listIDs = append(listIDs, slist.ID)
			}
		}
		pinnedIDs := []int{}
		for _, list := range pinnedLists {
			// sublists are not rendered for pinned lists
			pinnedIDs = append(pinnedIDs, list.ID)
		}
		listIDs = append(listIDs, pinnedIDs...)

		r = rt.buildPblistCache(r, listIDs)

		var hotProblems []*kilonova.ScoredProblem
		var moreHotProblems bool
		if flags.ShowTrending.Value() {
			hotProblems, err = rt.base.ScoredProblems(ctx, kilonova.ProblemFilter{
				LookingUser: u, Look: true,
				Ordering: "hot", Descending: true,
				Limit: 6,
			}, u, u)
			if err != nil {
				hotProblems = []*kilonova.ScoredProblem{}
			}
			if len(hotProblems) == 6 {
				hotProblems = hotProblems[:5]
				moreHotProblems = true
			}
		}

		var latestProblems []*sudoapi.FullProblem
		var problemCount int
		if flags.FrontPageLatestProblems.Value() {
			latestProblems, problemCount, err = rt.base.SearchProblems(ctx, kilonova.ProblemFilter{
				LookingUser: u, Look: true,

				Ordering: "published_at", Descending: true,
				Limit: 20,
			}, u, u)
			if err != nil {
				latestProblems = []*sudoapi.FullProblem{}
				problemCount = 0
			}
		}

		var others []*kilonova.ScoredProblem
		if flags.FrontPageProblems.Value() {
			others = rt.unassociatedProblems(ctx, u)
		}

		md := func(src string) string {
			out, err := rt.base.RenderMarkdown([]byte(src), nil)
			if err != nil {
				slog.WarnContext(ctx, "Error rendering markdown", slog.Any("err", err))
				return "[Error rendering markdown]"
			}
			return string(out)
		}
		lang := util.Language(r)
		mdKeys := make(map[string]string, len(home.MarkdownKeys))
		for _, key := range home.MarkdownKeys {
			mdKeys[key] = md(kilonova.GetText(lang, key))
		}

		p := home.Params{
			Version: kilonova.Version,
			Authed:  u != nil,

			ContestSidebar: flags.SidebarContests.Value(),
			MainPageLogin:  flags.MainPageLogin.Value(),
			SignupEnabled:  flags.SignupEnabled.Value(),

			ShowProblems:  flags.FrontPageProblems.Value(),
			OtherProblems: others,
			ShowPbLinks:   flags.FrontPagePbDetails.Value(),
			ShowRandomPb:  flags.FrontPageRandomProblem.Value(),

			ShowLatest:         flags.FrontPageLatestProblems.Value(),
			LatestProblems:     latestProblems,
			MoreLatestProblems: problemCount > 20,

			Announcements: []g.Node{modals.GPCAnnouncement()},
			QuickSearch:   flags.QuickSearchBox.Value(),

			PinnedLists:     pinnedLists,
			PinnedNumSolved: rt.numSolvedPblists(r, pinnedIDs),

			ShowTrending:    flags.ShowTrending.Value(),
			HotProblems:     hotProblems,
			MoreHotProblems: moreHotProblems,

			DonationNag: flags.DonationsEnabled.Value() && flags.DonationsNag.Value(),
			MD:          mdKeys,
		}
		if ann := flags.FrontPageAnnouncement.Value(); ann != "" {
			p.ShowAnnouncement = true
			if ann != "default" {
				p.AnnouncementHTML = md(ann)
			}
		}
		if url := flags.FeedbackURL.Value(); url != "" {
			p.FeedbackHTML = md(kilonova.GetText(lang, "feedbackCallToArms", url))
		}
		for _, c := range runningContests {
			p.RunningContests = append(p.RunningContests, rt.contestInfo(r, c))
		}
		for _, c := range futureContests {
			p.FutureContests = append(p.FutureContests, rt.contestInfo(r, c))
		}
		if p.ShowProblems {
			for _, list := range pblists {
				p.ChildProblemLists = append(p.ChildProblemLists, rt.pblistGroup(r, list, false))
			}
		}

		rt.runLayout(w, r, &LayoutParams{Title: "Kilonova", Content: home.Index(r.Context(), p)})
	}
}

// unassociatedProblems are the problems visible to u that are in no problem list.
func (rt *Web) unassociatedProblems(ctx context.Context, u *kilonova.UserBrief) []*kilonova.ScoredProblem {
	pbs, err := rt.base.ScoredProblems(ctx, kilonova.ProblemFilter{
		LookingUser: u, Look: true, Unassociated: true,
	}, u, u)
	if err != nil {
		return nil
	}
	return pbs
}

// numSolvedPblists is the number of problems the user solved in each list, from the
// cache built by buildPblistCache when possible. It is nil for visitors.
func (rt *Web) numSolvedPblists(r *http.Request, ids []int) map[int]int {
	ctx, u := r.Context(), user.UserBrief(r)
	if u == nil {
		return nil
	}
	cache, _ := ctx.Value(PblistCntCacheKey).(map[int]int)
	out := make(map[int]int, len(ids))
	for _, id := range ids {
		if cnt, ok := cache[id]; ok {
			out[id] = cnt
			continue
		}
		slog.WarnContext(ctx, "Cache miss", slog.Int("listID", id), slog.String("page", r.URL.Path))
		cnt, err := rt.base.NumSolvedFromPblist(ctx, id, u.ID)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get problem list number solved", slog.Any("err", err))
			cnt = -1
		}
		out[id] = cnt
	}
	return out
}
