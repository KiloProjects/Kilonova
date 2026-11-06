package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	knlanguage "github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	"github.com/KiloProjects/kilonova/web/ui/problempages"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/submissions"
	"github.com/shopspring/decimal"
	g "maragu.dev/gomponents"
)

func (rt *Web) problem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, problem, u := r.Context(), util.Problem(r), user.UserBrief(r)
		isEditor := rt.base.IsProblemEditor(u, problem)

		var statement = []byte("This problem doesn't have a statement.")

		variants, err := rt.base.ProblemDescVariants(ctx, problem.ID, isEditor)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get problem desc variants", slog.Any("err", err))
		}

		descVariant := rt.appropriateDescriptionVariant(r, variants)

		assetLink := fmt.Sprintf("/assets/problem/%d/attachment/%s?t=%d", problem.ID, rt.base.FormatDescName(descVariant), descVariant.LastUpdatedAt.UnixMilli())
		switch descVariant.Format {
		case "md":
			statement, err = rt.base.RenderedProblemDesc(ctx, problem, descVariant)
			if err != nil {
				slog.WarnContext(ctx, "Error getting problem markdown", slog.Any("err", err), slog.Any("variant", descVariant), slog.Any("problem", problem))
				statement = []byte("Error loading markdown.")
			}
		case "pdf":
			statement = fmt.Appendf(nil,
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh; background: white; object-fit: contain;"></embed>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link"), assetLink,
			)
		case "":
		default:
			statement = fmt.Appendf(nil,
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link"),
			)
		}

		atts, err := rt.base.ProblemAttachments(ctx, problem.ID)
		if err != nil {
			atts = nil
		}
		atts = slices.DeleteFunc(atts, func(att *kilonova.Attachment) bool { return !att.Visible && !isEditor })

		langs, err := rt.base.ProblemLanguages(ctx, problem)
		if err != nil {
			slog.WarnContext(ctx, "Error getting problem languages", slog.Any("err", err), slog.Any("problem", problem))
			rt.statusPage(w, r, 500, "Couldn't get supported problem languages.")
			return
		}

		var tags []*kilonova.Tag
		var showExternalResources bool
		var externalResources []*kilonova.ExternalResource
		if rt.base.IsProblemFullyVisible(u, problem) {
			tags, err = rt.base.ProblemTags(ctx, problem.ID)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get tags", slog.Any("err", err))
				tags = []*kilonova.Tag{}
			}
			if flags.ExternalResourcesEnabled.Value() {
				externalResources, _ = rt.base.ExternalResources(ctx, kilonova.ExternalResourceFilter{
					ProblemID: &problem.ID,
					Accepted:  new(true),
					// Technically not needed but just for safety
					Look:        true,
					LookingUser: u,
				})
				showExternalResources = len(externalResources) > 0 || isEditor
			}
		}

		var olderSubmissions *modals.OlderSubmissionsParams
		if u != nil {
			olderSubmissions, err = rt.getOlderSubmissions(ctx, u, u.ID, problem, util.Contest(r))
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get submissions", slog.Any("err", err))
				olderSubmissions = nil
			}
		}

		problemEditors, err := rt.base.ProblemEditors(ctx, problem.ID)
		if err != nil {
			problemEditors = nil
		}
		problemViewers, err := rt.base.ProblemViewers(ctx, problem.ID)
		if err != nil {
			problemViewers = nil
		}

		var maxScore *decimal.Decimal
		if u.IsAuthed() {
			maxScore = new(rt.base.MaxScore(ctx, u.ID, problem.ID))
		}

		topbar := rt.problemTopbar(r, "pb_statement", -1)
		params := problempages.StatementParams{
			Topbar: topbar,
			Info: &problems.InfoParams{
				Problem:      problem,
				IsEditor:     isEditor,
				Editors:      problemEditors,
				Viewers:      problemViewers,
				Tags:         tags,
				MaxScore:     maxScore,
				CanViewTests: rt.base.CanViewTests(u, problem),
				Contest:      util.Contest(r),
			},
			FromContest: slices.ContainsFunc(tags, func(tag *kilonova.Tag) bool {
				return tag.Type == kilonova.TagTypeContest
			}),
			OlderSubmissions:      olderSubmissions,
			Attachments:           atts,
			ShowExternalResources: showExternalResources,
			ExternalResources:     externalResources,
			Variants:              variants,
			SelectedVariant:       descVariant,
			StatementHTML:         string(statement),
			ReqPath:               reqPath(r),
		}
		if topbar.Contest != nil {
			params.ContestSidebar = new(rt.contestSidebar(r, topbar.Contest))
		} else {
			params.ParentLists = rt.problemParentLists(r, problem)
		}
		if u != nil {
			params.SubmitForm = new(rt.submitForm(r, topbar, langs))
		}

		rt.runLayout(w, r, &LayoutParams{
			Title:       kilonova.GetText(util.Language(r), "problem", problem.ID, problem.Name),
			Description: problempages.StatementDescription(problem, tags),
			Head:        ui.CanonicalURL(fmt.Sprintf("/problems/%d", problem.ID)),
			Content:     problempages.Statement(ctx, params),
		})
	}
}

// problemParentLists returns up to 5 lists containing problem, with the list_id one first.
func (rt *Web) problemParentLists(r *http.Request, problem *kilonova.Problem) []problempages.ParentList {
	ctx := r.Context()
	var topList *kilonova.ProblemList
	if id, err := strconv.Atoi(r.FormValue("list_id")); err == nil {
		list, err := rt.base.ProblemList(ctx, id)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get problem list", slog.Any("err", err))
		} else if slices.Contains(list.ProblemIDs(), problem.ID) {
			topList = list
		}
	}
	lists, err := rt.base.ProblemParentLists(ctx, problem.ID, false)
	if err != nil {
		return nil
	}
	if topList != nil {
		lists = slices.DeleteFunc(lists, func(l *kilonova.ProblemList) bool { return l.ID == topList.ID })
		lists = append([]*kilonova.ProblemList{topList}, lists...)
	}
	if len(lists) > 5 {
		lists = lists[:5]
	}
	out := make([]problempages.ParentList, 0, len(lists))
	for _, list := range lists {
		pbs, err := rt.base.ProblemListProblems(ctx, list.List, user.UserBrief(r))
		if err != nil {
			pbs = nil
		}
		out = append(out, problempages.ParentList{List: list, Problems: pbs})
	}
	return out
}

func (rt *Web) submitForm(r *http.Request, topbar *problems.TopbarParams, langs []knlanguage.Lang) problempages.SubmitFormParams {
	ctx := r.Context()
	p := problempages.SubmitFormParams{Topbar: topbar, Languages: langs}
	if topbar.Contest == nil {
		running, err := rt.base.ProblemRunningContests(ctx, topbar.Problem.ID)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get running contests", slog.Any("err", err))
		}
		for _, c := range running {
			if rt.base.CanSubmitInContest(user.UserBrief(r), c) {
				p.RunningContests = append(p.RunningContests, c)
			}
		}
	}
	explanation, err := rt.base.RenderMarkdown([]byte(kilonova.GetText(util.Language(r), "filename_explanation")), nil)
	if err != nil {
		slog.WarnContext(ctx, "Error rendering markdown", slog.Any("err", err))
		explanation = []byte("[Error rendering markdown]")
	}
	p.FilenameExplanationHTML = string(explanation)
	return p
}

func (rt *Web) problemSubmissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// OlderSubs HTMX fragment
		userID, err := strconv.Atoi(r.FormValue("user_id"))
		if isHTMXRequest(r) && err == nil {
			olderSubs, err := rt.getOlderSubmissions(r.Context(), user.UserBrief(r), userID, util.Problem(r), util.Contest(r))
			if err != nil {
				rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
				return
			}
			rt.componentModal(w, r, modals.OlderSubmissions(r.Context(), olderSubs))
			return
		}

		problem := util.Problem(r)
		query := parseSubsViewerQuery(r)
		query.ProblemID = &problem.ID
		overwrites := submissions.SubsViewerOverwrites{ProblemID: true}
		if contest := util.Contest(r); contest != nil {
			query.ContestID = &contest.ID
			overwrites.ContestID = true
		}
		filter := buildSubsFilter(query)
		subs, err := rt.base.Submissions(r.Context(), filter, true, user.UserBrief(r))
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch submissions")
			return
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.problem_past_submissions", problem.ID, problem.Name),
			Head:  ui.NoRobots(),
			Content: g.Group{
				problems.Topbar(r.Context(), rt.problemTopbar(r, "pb_submissions", -1)),
				submissions.SubsViewer(r.Context(), submissions.SubsViewerParams{
					Submissions: subs,
					Query:       query,
					Overwrites:  overwrites,
					NumPages:    (subs.Count + 49) / 50,
					Languages:   rt.base.EnabledLanguages(),
					IsAdmin:     user.UserBrief(r) != nil && user.UserBrief(r).IsAdmin(),
				}),
			},
		})
	}
}

func (rt *Web) problemStatistics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := rt.base.ProblemStatistics(r.Context(), util.Problem(r), user.UserBrief(r))
		if err != nil {
			rt.authedStatusPage(w, r, kilonova.ErrorCode(err), err.Error())
			return
		}

		params := problempages.StatisticsParams{Topbar: rt.problemTopbar(r, "pb_statistics", -1), Stats: stats}
		if r.FormValue("hx_modal") == "true" {
			rt.componentModal(w, r, problempages.StatisticsModal(r.Context(), params))
			return
		}
		rt.runLayout(w, r, &LayoutParams{
			Title:   problempages.StatisticsTitle(r.Context(), util.Problem(r)),
			Content: problempages.Statistics(r.Context(), params),
		})
	}
}

func (rt *Web) problemSubmit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		problem := util.Problem(r)
		langs, err := rt.base.ProblemLanguages(r.Context(), problem)
		if err != nil {
			slog.WarnContext(r.Context(), "Error getting problem languages", slog.Any("err", err), slog.Any("problem", problem))
			rt.statusPage(w, r, 500, "Couldn't get supported problem languages.")
			return
		}

		userBrief := user.UserBrief(r)
		query := parseSubsViewerQuery(r)
		query.ProblemID = &problem.ID
		query.UserID = &userBrief.ID
		overwrites := submissions.SubsViewerOverwrites{ProblemID: true, UserID: true}
		if contest := util.Contest(r); contest != nil {
			query.ContestID = &contest.ID
			overwrites.ContestID = true
		}
		filter := buildSubsFilter(query)
		subs, subsErr := rt.base.Submissions(r.Context(), filter, true, userBrief)
		if subsErr != nil {
			slog.WarnContext(r.Context(), "Couldn't fetch past submissions", slog.Any("err", subsErr))
		}
		var subViewer g.Node
		if subs != nil {
			subViewer = submissions.SubsViewer(r.Context(), submissions.SubsViewerParams{
				Submissions: subs,
				Query:       query,
				Overwrites:  overwrites,
				NumPages:    (subs.Count + 49) / 50,
				Title:       kilonova.GetText(util.Language(r), "header.past_submissions"),
				Languages:   rt.base.EnabledLanguages(),
				IsAdmin:     userBrief.IsAdmin(),
				InPlace:     true, // keep the code in the editor while browsing past submissions
			})
		}

		topbar := rt.problemTopbar(r, "pb_submit", -1)
		var sidebar *contests.SidebarParams
		if topbar.Contest != nil {
			sidebar = new(rt.contestSidebar(r, topbar.Contest))
		}
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.problem_submit", problem.ID, problem.Name),
			Content: problempages.SubmitPage(r.Context(), problempages.SubmitPageParams{
				Form:           rt.submitForm(r, topbar, langs),
				ContestSidebar: sidebar,
				SubViewer:      subViewer,
			}),
		})
	}
}

func (rt *Web) problemArchive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		problem := util.Problem(r)
		topbar := rt.problemTopbar(r, "pb_archive", -1)
		var tests []*kilonova.Test
		if topbar.CanViewTests {
			tests2, err := rt.base.Tests(r.Context(), problem.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get tests", slog.Any("err", err))
			} else {
				tests = tests2
			}
		}
		settings, err := rt.base.ProblemSettings(r.Context(), problem)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get problem settings", slog.Any("err", err))
			settings = nil
		}
		var captchaID string
		if flags.CaptchaEnabled.Value() && !user.UserBrief(r).IsAdmin() {
			captchaID = rt.base.NewCaptchaID()
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.problem_archive", problem.ID, problem.Name),
			Head:  ui.NoRobots(),
			Content: problempages.Archive(r.Context(), problempages.ArchiveParams{
				Topbar:    topbar,
				Tests:     tests,
				Settings:  settings,
				CaptchaID: captchaID,
			}),
		})
	}
}
