package web

import (
	"fmt"
	"net/http"

	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/problems"
)

type ProblemTopbar = problems.TopbarParams

func (rt *Web) problemTopbar(r *http.Request, page string, pageID int) *ProblemTopbar {
	prefix := ""
	contest, problem, u := util.Contest(r), util.Problem(r), user.UserBrief(r)
	if contest != nil {
		prefix = fmt.Sprintf("/contests/%d", contest.ID)
	}
	t := &ProblemTopbar{
		IsProblemEditor: rt.base.IsProblemEditor(u, problem),
		IsContestEditor: contest.IsEditor(u),
		CanViewTests:    rt.base.CanViewTests(u, problem),
		Authed:          u != nil,
		FullyVisible:    rt.base.IsProblemFullyVisible(u, problem),

		Contest: contest,
		Problem: problem,

		URLPrefix: prefix,

		Page:   page,
		PageID: pageID,
	}
	if contest != nil {
		t.CanSubmitInContest = rt.base.CanSubmitInContest(u, contest)
		t.LeaderboardVisible = rt.base.CanViewContestLeaderboard(u, contest)
	}
	return t
}
