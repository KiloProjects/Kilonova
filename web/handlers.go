package web

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"math/rand/v2"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"runtime/metrics"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/web/views"
	"github.com/KiloProjects/kilonova/web/views/modals"
	"github.com/KiloProjects/kilonova/web/views/utilviews"
	"github.com/skip2/go-qrcode"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/components/layout"
	"github.com/a-h/templ"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/schema"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	PblistCntCacheKey = util.KNContextType("pblist_cache")

	MiddlewareStartKey = util.KNContextType("middleware_start")
)

func (rt *Web) buildPblistCache(r *http.Request, listIDs []int) *http.Request {
	if util.UserBrief(r) == nil {
		return r
	}
	cache, err := rt.base.NumSolvedFromPblists(r.Context(), listIDs, util.UserBrief(r))
	if err == nil {
		return r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, cache))
	}
	if errors.Is(err, context.Canceled) {
		// Build mock cache to silence cache misses
		mockCache := make(map[int]int)
		for _, id := range listIDs {
			mockCache[id] = 0
		}
		return r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, mockCache))
	}
	slog.WarnContext(r.Context(), "Couldn't build problem list cache", slog.Any("err", err))
	return r
}

func (rt *Web) discordLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if util.UserFull(r).DiscordID != nil {
			http.Redirect(w, r, "/profile/linked", http.StatusTemporaryRedirect)
			return
		}
		st, err := rt.base.DiscordAuthURL(r.Context(), util.UserBrief(r).ID)
		if err != nil {
			rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
			return
		}

		http.Redirect(w, r, st, http.StatusTemporaryRedirect)
	}
}

func (rt *Web) index() http.HandlerFunc {
	templ := rt.parse(nil, "index.html", "modals/pblist.html", "modals/pbs.html", "modals/contest_brief.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runningContests, err := rt.base.VisibleRunningContests(r.Context(), util.UserBrief(r))
		if err != nil {
			runningContests = []*kilonova.Contest{}
		}
		futureContests, err := rt.base.VisibleFutureContests(r.Context(), util.UserBrief(r))
		if err != nil {
			futureContests = []*kilonova.Contest{}
		}

		var pblists []*kilonova.ProblemList
		if val := flags.RootProblemList.Value(); val > 0 {
			pblists, err = rt.base.PblistChildrenLists(r.Context(), val)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get index page problem lists", slog.Any("err", err))
				pblists = []*kilonova.ProblemList{}
			}
		}

		var pinnedLists []*kilonova.ProblemList
		if val := flags.PinnedProblemList.Value(); val > 0 {
			pinnedLists, err = rt.base.PblistChildrenLists(r.Context(), val)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get pinned problem lists", slog.Any("err", err))
				pblists = []*kilonova.ProblemList{}
			}
		}

		listIDs := []int{}
		for _, list := range pblists {
			listIDs = append(listIDs, list.ID)
			for _, slist := range list.SubLists {
				listIDs = append(listIDs, slist.ID)
			}
		}
		for _, list := range pinnedLists {
			listIDs = append(listIDs, list.ID)
			// sublists are not rendered for pinned lists
			// for _, slist := range list.SubLists {
			// 	listIDs = append(listIDs, slist.ID)
			// }
		}

		r = rt.buildPblistCache(r, listIDs)

		var hotProblems []*kilonova.ScoredProblem
		var moreHotProblems bool
		if flags.ShowTrending.Value() {
			hotProblems, err = rt.base.ScoredProblems(r.Context(), kilonova.ProblemFilter{
				LookingUser: util.UserBrief(r), Look: true,
				Ordering: "hot", Descending: true,
				Limit: 6,
			}, util.UserBrief(r), util.UserBrief(r))
			if err != nil {
				hotProblems = []*kilonova.ScoredProblem{}
			}
			if len(hotProblems) == 6 {
				hotProblems = hotProblems[:5]
				moreHotProblems = true
			}
		}

		latestProblems, problemCount, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
			LookingUser: util.UserBrief(r), Look: true,

			Ordering: "published_at", Descending: true,
			Limit: 20,
		}, util.UserBrief(r), util.UserBrief(r))
		if err != nil {
			latestProblems = []*sudoapi.FullProblem{}
			problemCount = 0
		}

		rt.runTempl(w, r, templ, &IndexParams{
			futureContests, runningContests,
			pblists,
			hotProblems, moreHotProblems,
			latestProblems, problemCount > 20,
			pinnedLists,
		})
	}
}

func (rt *Web) problems() http.HandlerFunc {
	templ := rt.parse(nil, "pbs.html", "modals/pbs.html")
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")
	type filterQuery struct {
		Query   *string `json:"q"`
		Editor  *int    `json:"editor_user"`
		Visible *bool   `json:"published"`

		DeepListID *int `json:"deep_list_id"`

		Ordering   string `json:"ordering"`
		Descending bool   `json:"descending"`

		Language string `json:"lang"`

		Tags *string `json:"tags"`

		Page uint64 `json:"page"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var q filterQuery
		r.ParseForm()
		if err := decoder.Decode(&q, r.Form); err != nil {
			slog.WarnContext(r.Context(), "Couldn't decode problems query", slog.Any("err", err))
		}
		if q.Page < 1 {
			q.Page = 1
		}
		gr := []*kilonova.TagGroup{}
		tags := []*kilonova.Tag{}

		if q.Tags != nil {
			// Parse tag string
			// format example:
			// - !1,2_3,4_5_6,9 = (NOT 1) AND (2 OR 3) AND (4) AND (4 OR 5 OR 6) AND (9)

			var allTags []int

			groups := strings.Split(*q.Tags, ",")
			for _, group := range groups {
				group := group
				if len(group) == 0 {
					continue
				}

				// Check negate flag
				negate := false
				if group[0] == '!' {
					group = group[1:]
					negate = true
				}

				// Parse tags and, if some found, add group
				tags := strings.Split(group, "_")
				var tagIDs []int
				for _, tag := range tags {
					id, err := strconv.Atoi(tag)
					if err == nil {
						allTags = append(allTags, id)
						tagIDs = append(tagIDs, id)
					}
				}
				if len(tagIDs) > 0 {
					gr = append(gr, &kilonova.TagGroup{
						Negate: negate,
						TagIDs: tagIDs,
					})
				}
			}

			slices.Sort(allTags)
			allTags = slices.Compact(allTags)

			var err error
			tags, err = rt.base.TagsByID(r.Context(), allTags)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get tags", slog.Any("err", err))
			}

			var tagMap = make(map[int]*kilonova.Tag)
			for _, tag := range tags {
				tagMap[tag.ID] = tag
			}

			for i := range gr {
				gr[i].TagIDs = slices.DeleteFunc(gr[i].TagIDs, func(id int) bool {
					_, ok := tagMap[id]
					return !ok
				})
			}

			gr = slices.DeleteFunc(gr, func(tg *kilonova.TagGroup) bool {
				return len(tg.TagIDs) == 0
			})
		}

		var pblist *kilonova.ProblemList
		if q.DeepListID != nil {
			list, err := rt.base.ProblemList(r.Context(), *q.DeepListID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get problem list", slog.Any("err", err))
				list = nil
				q.DeepListID = nil
			}
			pblist = list
		}

		var lang *string
		if q.Language == "en" || q.Language == "ro" {
			lang = &q.Language
		}

		pbs, cnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
			LookingUser: util.UserBrief(r), Look: true,
			FuzzyName: q.Query, EditorUserID: q.Editor, Visible: q.Visible,
			DeepListID: q.DeepListID, Language: lang,

			Tags:  gr,
			Limit: 50, Offset: (q.Page - 1) * 50,
			Ordering: q.Ordering, Descending: q.Descending,
		}, util.UserBrief(r), util.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Could not search problems", slog.Any("err", err))
			// TODO: Maybe not fail to load and instead just load on the browser?
			rt.statusPage(w, r, 500, "N-am putut încărca problemele")
			return
		}
		rt.runTempl(w, r, templ, &ProblemSearchParams{pblist, pbs, gr, tags, cnt})
	}
}

func (rt *Web) tags() http.HandlerFunc {
	templ := rt.parse(nil, "tags/index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, struct{}{})
	}
}

type TagPageParams struct {
	Tag *kilonova.Tag

	RelevantTags []*kilonova.Tag
	Problems     []*sudoapi.FullProblem
	ProblemCount int
}

func (rt *Web) tag() http.HandlerFunc {
	templ := rt.parse(nil, "tags/tag.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pbs, pbsCnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
			LookingUser: util.UserBrief(r), Look: true, LookFullyVisible: true,
			Tags: []*kilonova.TagGroup{{TagIDs: []int{util.Tag(r).ID}}},

			Limit: 50,
		}, util.UserBrief(r), util.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't fetch tag problems", slog.Any("err", err))
			pbs = []*sudoapi.FullProblem{}
		}

		relevantTags, err := rt.base.RelevantTags(r.Context(), util.Tag(r).ID, 15, util.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't fetch relevant tags", slog.Any("err", err))
			relevantTags = nil
		}
		rt.runTempl(w, r, templ, &TagPageParams{util.Tag(r), relevantTags, pbs, pbsCnt})
	}
}

func (rt *Web) justRender(files ...string) http.HandlerFunc {
	templ := rt.parse(nil, files...)
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, struct{}{})
	}
}

func (rt *Web) pbListIndex() http.HandlerFunc {
	templ := rt.parse(nil, "lists/index.html", "modals/pblist.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pblists, err := rt.base.ProblemLists(r.Context(), kilonova.ProblemListFilter{Root: true})
		if err != nil {
			rt.statusPage(w, r, 500, "Eroare la obținerea listelor")
			return
		}

		listIDs := []int{}
		for _, list := range pblists {
			listIDs = append(listIDs, list.ID)
			for _, slist := range list.SubLists {
				listIDs = append(listIDs, slist.ID)
			}
		}

		r = rt.buildPblistCache(r, listIDs)

		rt.runTempl(w, r, templ, &ProblemListParams{
			nil, pblists, -1,
			nil,
		})
	}
}

func (rt *Web) pbListProgressIndex() http.HandlerFunc {
	templ := rt.parse(nil, "lists/pIndex.html")
	t := true
	return func(w http.ResponseWriter, r *http.Request) {
		pblists, err := rt.base.ProblemLists(r.Context(), kilonova.ProblemListFilter{FeaturedChecklist: &t})
		if err != nil {
			rt.statusPage(w, r, 500, "Eroare la obținerea listelor")
			return
		}

		listIDs := []int{}
		for _, list := range pblists {
			listIDs = append(listIDs, list.ID)
			for _, slist := range list.SubLists {
				listIDs = append(listIDs, slist.ID)
			}
		}

		r = rt.buildPblistCache(r, listIDs)

		rt.runTempl(w, r, templ, &ProblemListParams{
			nil, pblists, flags.RootProblemList.Value(),
			nil,
		})
	}
}

func computeChecklistSpan(list *sudoapi.FullProblemList) int {
	var cnt, countedSubLists int
	for _, sublist := range list.SubLists {
		cnt1 := computeChecklistSpan(sublist)
		cnt += cnt1
		if cnt1 > 0 {
			countedSubLists++
		}
	}
	if len(list.Problems) > 0 || countedSubLists > 0 {
		cnt++
	}

	return cnt
}

func (rt *Web) pbListProgressView() http.HandlerFunc {
	templ := rt.parse(nil, "lists/progress.html")
	return func(w http.ResponseWriter, r *http.Request) {
		// listIDs := []int{util.ProblemList(r).ID}
		// for _, slist := range util.ProblemList(r).SubLists {
		// 	listIDs = append(listIDs, slist.ID)
		// }

		// r = rt.buildPblistCache(r, listIDs)

		var checkedUser = util.UserBrief(r)
		if uname := r.FormValue("username"); len(uname) > 0 {
			user, err := rt.base.UserBriefByName(r.Context(), uname)
			if err == nil {
				checkedUser = user
			}
		}

		list, err := rt.base.FullProblemList(r.Context(), util.ProblemList(r).ID, checkedUser, util.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get problem list", slog.Any("err", err))
			rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
			return
		}
		rt.runTempl(w, r.WithContext(context.WithValue(r.Context(), util.ContentUserKey, checkedUser)), templ, &ProblemListProgressParams{list, checkedUser})
	}
}

func (rt *Web) pbListView() http.HandlerFunc {
	templ := rt.parse(nil, "lists/view.html", "modals/pblist.html", "modals/pbs.html", "proposer/createpblist.html")
	fragmentTempl := rt.parse(nil, "modals/pblist.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		listIDs := []int{util.ProblemList(r).ID}
		for _, slist := range util.ProblemList(r).SubLists {
			listIDs = append(listIDs, slist.ID)
		}

		r = rt.buildPblistCache(r, listIDs)

		if isHTMXRequest(r) {
			rt.runModal(w, r, fragmentTempl, "problemlist_show", &PblistParams{util.ProblemList(r), true})
			return
		}

		problems, err := rt.base.ProblemListProblems(r.Context(), util.ProblemList(r).List, util.UserBrief(r))
		if err != nil {
			slog.ErrorContext(r.Context(), "Could not get problems", slog.Any("err", err))
		}

		rt.runTempl(w, r, templ, &ProblemListParams{
			util.ProblemList(r), nil, -1,
			modals.ProblemSources(util.ProblemList(r), problems),
		})
	}
}

func (rt *Web) auditLog() http.HandlerFunc {
	templ := rt.parse(nil, "admin/audit_log.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.FormValue("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			page = 0
		}

		logs, err := rt.base.GetAuditLogs(r.Context(), 50, (page-1)*50)
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch logs")
			return
		}

		numLogs, err := rt.base.GetLogCount(r.Context())
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch log count")
			return
		}

		numPages := numLogs / 50
		if numLogs%50 > 0 {
			numPages++
		}

		rt.runTempl(w, r, templ, &AuditLogParams{logs, page, numPages})
	}
}

func (rt *Web) debugPage() http.HandlerFunc {
	templ := rt.parse(nil, "admin/debug.html")
	type Metric struct {
		Name        string
		Description string

		Int   *uint64
		Float *float64
	}
	return func(w http.ResponseWriter, r *http.Request) {
		all := metrics.All()
		var descs = make(map[string]string)
		samples := make([]metrics.Sample, 0, len(all))
		for _, m := range all {
			if m.Kind == metrics.KindFloat64 || m.Kind == metrics.KindUint64 {
				descs[m.Name] = m.Description
				samples = append(samples, metrics.Sample{Name: m.Name})
			}
		}
		metrics.Read(samples)
		finalMetrics := make([]*Metric, 0, len(samples))
		for _, sample := range samples {
			desc, ok := descs[sample.Name]
			if !ok {
				slog.WarnContext(r.Context(), "Could not find description for metric", slog.String("name", sample.Name))
				desc = "No description provided"
			}
			m := &Metric{
				Name:        sample.Name,
				Description: desc,
			}
			switch sample.Value.Kind() {
			case metrics.KindFloat64:
				v := sample.Value.Float64()
				m.Float = &v
			case metrics.KindUint64:
				v := sample.Value.Uint64()
				m.Int = &v
			default:
				slog.WarnContext(r.Context(), "Unknown metric type", slog.String("name", sample.Name), slog.Any("kind", sample.Value.Kind()))
			}
			finalMetrics = append(finalMetrics, m)
		}

		var stats = make([]*datastore.BucketStats, 0, 16)
		for _, bucket := range rt.base.DataStore().GetAll() {
			stats = append(stats, bucket.Statistics(false))
		}
		slices.SortFunc(stats, func(a, b *datastore.BucketStats) int { return cmp.Compare(a.Name, b.Name) })

		rt.runTempl(w, r, templ, &struct {
			Metrics []*Metric

			BucketStats []*datastore.BucketStats
		}{finalMetrics, stats})
	}
}

func (rt *Web) submission(w http.ResponseWriter, r *http.Request) {
	sub := util.Submission(r)

	var olderSubs *modals.OlderSubmissionsParams = nil
	var err error
	if util.UserBrief(r) != nil {
		olderSubs, err = rt.getOlderSubmissions(r.Context(), util.UserBrief(r), util.UserBrief(r).ID, sub.Problem, nil, 5)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get submissions", slog.Any("err", err))
		}
	}
	if olderSubs != nil {
		olderSubs.AutoReload = false
	}

	rt.runLayout(w, r, &LayoutParams{
		Title: fmt.Sprintf("%s %d", kilonova.GetText(util.Language(r), "sub"), sub.ID),
		Head:  utilviews.NoRobotsHead(),
		Content: views.Submission(views.SubmissionPageParams{
			Submission:    sub,
			ForceShowCode: r.URL.Query().Get("forceCode") == "1",
			LanguageFormatter: func(lang string) string {
				return rt.base.Language(r.Context(), lang).PrintableName
			},
			OlderSubmissions: olderSubs,
		}),
	})
}

func (rt *Web) downloadSubmission(w http.ResponseWriter, r *http.Request) {
	if len(util.Submission(r).Code) == 0 {
		rt.statusPage(w, r, 400, "Code is either unavailable or doesn't exist.")
		return
	}
	extension := rt.base.Language(r.Context(), util.Submission(r).Language).Extensions()[0]
	if extension == ".outputOnly" {
		extension = ".txt"
	}
	filename := fmt.Sprintf("%d-%s%s", util.Submission(r).ID, kilonova.MakeSlug(util.Submission(r).Problem.Name), extension)
	w.Header().Add("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	http.ServeContent(w, r, filename, util.Submission(r).CreatedAt, bytes.NewReader(util.Submission(r).Code))
}

func (rt *Web) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	// Check submission permissions
	if !util.Submission(r).CanDelete(util.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't delete this submission!")
		return
	}

	if err := rt.base.DeleteSubmission(r.Context(), util.Submission(r).ID); err != nil {
		rt.statusPage(w, r, 500, "Could not delete submission")
		return
	}

	w.Header().Set("HX-Redirect", "/submissions")
	http.Redirect(w, r, "/submissions", http.StatusNoContent)
}

func (rt *Web) reevaluateSubmission(w http.ResponseWriter, r *http.Request) {
	// Check submission permissions
	if !util.Submission(r).CanDelete(util.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't reevaluate this submission!")
		return
	}

	if err := rt.base.ResetSubmission(r.Context(), util.Submission(r).ID); err != nil {
		rt.statusPage(w, r, 500, "Could not reevaluate submission")
		return
	}

	htmxSuccessToast(w, r, "Reset submission")
	w.Header().Set("HX-Refresh", "true")
	http.Redirect(w, r, "/submissions/"+strconv.Itoa(util.Submission(r).ID), http.StatusFound)
}

func (rt *Web) createPaste(w http.ResponseWriter, r *http.Request) {
	if !util.Submission(r).IsEditor(util.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't create a paste for this submission!")
		return
	}

	id, err := rt.base.CreatePaste(r.Context(), &util.Submission(r).Submission, util.UserBrief(r))
	if err != nil {
		rt.statusPage(w, r, 500, "Could not create paste")
		return
	}
	http.Redirect(w, r, "/pastes/"+id, http.StatusFound)
}

func (rt *Web) submissions() http.HandlerFunc {
	templ := rt.parse(nil, "submissions.html")
	return func(w http.ResponseWriter, r *http.Request) {
		if !rt.canViewAllSubs(util.UserBrief(r)) {
			rt.statusPage(w, r, 401, "Cannot view all submissions")
			return
		}
		rt.runTempl(w, r, templ, struct{}{})
	}
}

// canViewAllSubs is just for the text in the navbar and the submissions page
// TODO: Restrict on the backend, as well.
func (rt *Web) canViewAllSubs(user *kilonova.UserBrief) bool {
	if flags.AllSubsPage.Value() {
		return true
	}
	return user.IsProposer()
}

func (rt *Web) paste(w http.ResponseWriter, r *http.Request) {
	sub, err := rt.base.FullSubmission(r.Context(), util.Paste(r).Submission.ID)
	if err != nil {
		rt.statusPage(w, r, 500, "N-am putut obține submisia aferentă")
		return
	}

	var olderSubs *modals.OlderSubmissionsParams
	if util.UserBrief(r) != nil {
		olderSubs, err = rt.getOlderSubmissions(r.Context(), util.UserBrief(r), util.UserBrief(r).ID, sub.Problem, nil, 5)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get submissions", slog.Any("err", err))
		}
	}
	if olderSubs != nil {
		olderSubs.AutoReload = false
	}

	rt.runLayout(w, r, &LayoutParams{
		Title: fmt.Sprintf("%s #%d", kilonova.GetText(util.Language(r), "sub"), sub.ID),
		Content: views.Paste(views.PastePageParams{
			Paste: util.Paste(r),
			SubmissionPageParams: views.SubmissionPageParams{
				Submission:    sub,
				ForceShowCode: true,
				LanguageFormatter: func(lang string) string {
					return rt.base.Language(r.Context(), lang).PrintableName
				},
				OlderSubmissions: olderSubs,
			},
		}),
	})
}

func (rt *Web) deletePaste(w http.ResponseWriter, r *http.Request) {
	// Check paste permissions
	if !util.Paste(r).IsEditor(util.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't delete this paste!")
		return
	}

	if err := rt.base.DeletePaste(r.Context(), util.Paste(r).ID); err != nil {
		rt.statusPage(w, r, 500, "Could not delete submission")
		return
	}

	w.Header().Set("HX-Redirect", "/submissions")
	http.Redirect(w, r, "/submissions", http.StatusNoContent)
}

func (rt *Web) downloadPaste() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fullSub, err := rt.base.FullSubmission(r.Context(), util.Paste(r).Submission.ID)
		if err != nil {
			rt.statusPage(w, r, 500, "N-am putut obține submisia aferentă")
			return
		}
		code, err := rt.base.SubmissionCode(r.Context(), &fullSub.Submission, fullSub.Problem, nil, false)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get submission code", slog.Any("err", err))
			code = nil
		}

		extension := rt.base.Language(r.Context(), fullSub.Language).Extensions()[0]
		if extension == ".outputOnly" {
			extension = ".txt"
		}
		filename := fmt.Sprintf("%s-%s%s", util.Paste(r).ID, kilonova.MakeSlug(fullSub.Problem.Name), extension)
		w.Header().Add("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		http.ServeContent(w, r, filename, fullSub.CreatedAt, bytes.NewReader(code))
	}
}

func (rt *Web) randomProblem() http.HandlerFunc {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")

	type problemArgs struct {
		ListID int   `json:"list_id"`
		TagIDs []int `json:"tag_id"`

		// If nil, it's disregarded
		// If true, searches for unsolved problems
		// If false, searches for solved problems
		Unsolved *bool `json:"unsolved"`
		// User ID to base searches on.
		UnsolvedBy *int `json:"unsolved_by"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args problemArgs
		if err := decoder.Decode(&args, r.Form); err != nil {
			slog.WarnContext(r.Context(), "Could not decode HTTP form", slog.Any("err", err))
		}
		filter := kilonova.ProblemFilter{
			Look: true, LookingUser: util.UserBrief(r),
		}
		if args.ListID > 0 {
			filter.DeepListID = &args.ListID
		}
		if len(args.TagIDs) > 0 {
			filter.Tags = []*kilonova.TagGroup{
				{TagIDs: args.TagIDs},
			}
		}
		if args.Unsolved != nil {
			userID := -1
			if util.UserBrief(r) != nil {
				userID = util.UserBrief(r).ID
			}
			if args.UnsolvedBy != nil && *args.UnsolvedBy > 0 {
				userID = *args.UnsolvedBy
			}
			if userID > 0 {
				if *args.Unsolved {
					filter.UnsolvedBy = &userID
				} else {
					filter.SolvedBy = &userID
				}
			}
		}

		pbs, err := rt.base.Problems(r.Context(), filter)
		if err != nil {
			w.Header().Add("X-Problem-ID", "-1")
			rt.statusPage(w, r, 500, "Could not get random problem: "+err.Error())
			return
		}
		if len(pbs) == 0 {
			w.Header().Add("X-Problem-ID", "-1")
			rt.statusPage(w, r, 400, "Could not find a random problem matching the given criteria")
			return
		}

		pbid := strconv.Itoa(pbs[rand.N(len(pbs))].ID)
		w.Header().Add("X-Problem-ID", pbid)
		http.Redirect(w, r, "/problems/"+pbid, http.StatusFound)
	}
}

func (rt *Web) appropriateDescriptionVariant(r *http.Request, variants []*kilonova.StatementVariant) *kilonova.StatementVariant {
	prefLang, prefFormat, prefType := util.Language(r), "md", ""
	variant := strings.SplitN(r.FormValue("var"), "-", 3)
	if lang := r.FormValue("pref_lang"); len(lang) > 0 { // Backwards compatibility
		prefLang = lang
	}
	if len(variant) > 0 && len(variant[0]) > 0 {
		prefLang = variant[0]
	}
	if format := r.FormValue("pref_format"); len(format) > 0 { // Backwards compatibility
		prefFormat = format
	}
	if len(variant) > 1 && len(variant[1]) > 0 {
		prefFormat = variant[1]
	}
	if len(variant) > 2 && len(variant[2]) > 0 {
		prefType = variant[2]
	}

	if len(variants) == 0 {
		return &kilonova.StatementVariant{}
	}
	// Search for the ideal scenario
	for _, v := range variants {
		if v.Language == prefLang && v.Format == prefFormat && v.Type == prefType {
			return v
		}
	}
	// Then search if anything matches the format
	for _, v := range variants {
		if v.Format == prefFormat {
			return v
		}
	}
	// Then search if anything matches the language
	for _, v := range variants {
		if v.Language == prefLang {
			return v
		}
	}
	// If nothing was found, then just return the first available variant
	return variants[0]
}

func (rt *Web) problem() http.HandlerFunc {
	pageTempl := rt.parse(nil, "problem/summary.html", "problem/topbar.html", "modals/contest_sidebar.html", "modals/pb_submit_form.html")
	return func(w http.ResponseWriter, r *http.Request) {
		problem := util.Problem(r)

		var statement = []byte("This problem doesn't have a statement.")

		variants, err := rt.base.ProblemDescVariants(r.Context(), problem.ID, rt.base.IsProblemEditor(util.UserBrief(r), problem))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get problem desc variants", slog.Any("err", err))
		}

		descVariant := rt.appropriateDescriptionVariant(r, variants)

		assetLink := fmt.Sprintf("/assets/problem/%d/attachment/%s?t=%d", problem.ID, rt.base.FormatDescName(descVariant), descVariant.LastUpdatedAt.UnixMilli())
		switch descVariant.Format {
		case "md":
			statement, err = rt.base.RenderedProblemDesc(r.Context(), problem, descVariant)
			if err != nil {
				slog.WarnContext(r.Context(), "Error getting problem markdown", slog.Any("err", err), slog.Any("variant", descVariant), slog.Any("problem", problem))
				statement = []byte("Error loading markdown.")
			}
		case "pdf":
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh; background: white; object-fit: contain;"></embed>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link"), assetLink,
			))
		case "":
		default:
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link"),
			))
		}

		atts, err := rt.base.ProblemAttachments(r.Context(), problem.ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}

		if atts != nil {
			newAtts := make([]*kilonova.Attachment, 0, len(atts))
			for _, att := range atts {
				if att.Visible || rt.base.IsProblemEditor(util.UserBrief(r), problem) {
					newAtts = append(newAtts, att)
				}
			}

			atts = newAtts
		}

		langs, err := rt.base.ProblemLanguages(r.Context(), util.Problem(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Error getting problem languages", slog.Any("err", err), slog.Any("problem", util.Problem(r)))
			rt.statusPage(w, r, 500, "Couldn't get supported problem languages.")
			return
		}

		var tags []*kilonova.Tag
		var showExternalResources bool
		var externalResources []*kilonova.ExternalResource
		if rt.base.IsProblemFullyVisible(util.UserBrief(r), util.Problem(r)) {
			tags, err = rt.base.ProblemTags(r.Context(), util.Problem(r).ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get tags", slog.Any("err", err))
				tags = []*kilonova.Tag{}
			}
			if flags.ExternalResourcesEnabled.Value() {
				accepted := true
				externalResources, err = rt.base.ExternalResources(r.Context(), kilonova.ExternalResourceFilter{
					ProblemID: &util.Problem(r).ID,
					Accepted:  &accepted,
					// Technically not needed but just for safety
					Look:        true,
					LookingUser: util.UserBrief(r),
				})
				showExternalResources = len(externalResources) > 0 || rt.base.IsProblemEditor(util.UserBrief(r), util.Problem(r))
			}
		}

		var olderSubmissions templ.Component = nil
		if util.UserBrief(r) != nil {
			olderSubs, err := rt.getOlderSubmissions(r.Context(), util.UserBrief(r), util.UserBrief(r).ID, util.Problem(r), util.Contest(r), 5)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get submissions", slog.Any("err", err))
			} else {
				olderSubmissions = modals.OlderSubmissions(olderSubs)
			}
		}

		rt.runTempl(w, r, pageTempl, &ProblemParams{
			Topbar: rt.problemTopbar(r, "pb_statement", -1),

			Problem:     util.Problem(r),
			Attachments: atts,
			Tags:        tags,

			Statement: template.HTML(statement),
			Languages: langs,
			Variants:  variants,

			OlderSubmissions: olderSubmissions,

			SelectedVariant: descVariant,

			ShowExternalResources: showExternalResources,
			ExternalResources:     externalResources,
		})
	}
}

func (rt *Web) problemSubmissions() http.HandlerFunc {
	normalTempl := rt.parse(nil, "problem/pb_submissions.html", "problem/topbar.html")

	return func(w http.ResponseWriter, r *http.Request) {
		// OlderSubs HTMX fragment
		userID, err := strconv.Atoi(r.FormValue("user_id"))
		if isHTMXRequest(r) && err == nil {
			olderSubs, err := rt.getOlderSubmissions(r.Context(), util.UserBrief(r), userID, util.Problem(r), util.Contest(r), 5)
			if err != nil {
				rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
				return
			}
			rt.componentModal(w, r, modals.OlderSubmissions(olderSubs))
			return
		}

		rt.runTempl(w, r, normalTempl, &ProblemTopbarParams{
			Topbar: rt.problemTopbar(r, "pb_submissions", -1),

			Problem: util.Problem(r),
		})
	}
}

func (rt *Web) problemStatistics() http.HandlerFunc {
	templ := rt.parse(nil, "problem/statistics.html", "problem/topbar.html", "modals/htmx/helpers.html")
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := rt.base.ProblemStatistics(r.Context(), util.Problem(r), util.UserBrief(r))
		if err != nil {
			rt.authedStatusPage(w, r, kilonova.ErrorCode(err), err.Error())
			return
		}

		rt.runTemplModal(w, r, templ, &ProblemStatisticsParams{
			Topbar: rt.problemTopbar(r, "pb_statistics", -1),

			Problem:           util.Problem(r),
			ProblemStatistics: stats,
		})
	}
}

func (rt *Web) problemSubmit() http.HandlerFunc {
	templ := rt.parse(nil, "problem/pb_submit.html", "problem/topbar.html", "modals/contest_sidebar.html", "modals/pb_submit_form.html")
	return func(w http.ResponseWriter, r *http.Request) {
		langs, err := rt.base.ProblemLanguages(r.Context(), util.Problem(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Error getting problem languages", slog.Any("err", err), slog.Any("problem", util.Problem(r)))
			rt.statusPage(w, r, 500, "Couldn't get supported problem languages.")
			return
		}

		rt.runTempl(w, r, templ, &ProblemTopbarParams{
			Topbar: rt.problemTopbar(r, "pb_submit", -1),

			Languages: langs,
			Problem:   util.Problem(r),
		})
	}
}

func (rt *Web) problemArchive() http.HandlerFunc {
	templ := rt.parse(nil, "problem/pb_archive.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		topbar := rt.problemTopbar(r, "pb_archive", -1)
		var tests []*kilonova.Test
		if topbar.CanViewTests {
			tests2, err := rt.base.Tests(r.Context(), util.Problem(r).ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get tests", slog.Any("err", err))
			} else {
				tests = tests2
			}
		}
		settings, err := rt.base.ProblemSettings(r.Context(), util.Problem(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get problem settings", slog.Any("err", err))
			settings = nil
		}

		rt.runTempl(w, r, templ, &ProblemArchiveParams{
			Topbar: topbar,

			Tests:    tests,
			Problem:  util.Problem(r),
			Settings: settings,
		})
	}
}

var maxPostsPerPage = 10

func (rt *Web) blogPosts() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.Atoi(r.FormValue("page"))
		if err != nil {
			page = 1
		}

		filter := kilonova.BlogPostFilter{
			Limit:  maxPostsPerPage,
			Offset: (page - 1) * maxPostsPerPage,

			Look:        true,
			LookingUser: util.UserBrief(r),
		}

		posts, err := rt.base.BlogPosts(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get blog posts", slog.Any("err", err))
			rt.statusPage(w, r, 500, "N-am putut încărca postările")
			return
		}

		numPosts, err := rt.base.CountBlogPosts(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get number of posts", slog.Any("err", err))
			numPosts = 0
		}

		numPages := numPosts / maxPostsPerPage
		if numPosts%maxPostsPerPage > 0 {
			numPages++
		}

		authorIDs := []int{}
		for _, post := range posts {
			authorIDs = append(authorIDs, post.AuthorID)
		}

		authors, err := rt.base.UsersBrief(r.Context(), kilonova.UserFilter{IDs: authorIDs})
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get authors", slog.Any("err", err))
		}

		authorMap := make(map[int]*kilonova.UserBrief)
		for _, author := range authors {
			authorMap[author.ID] = author
		}

		for _, post := range posts {
			if _, ok := authorMap[post.AuthorID]; !ok {
				authorMap[post.AuthorID] = nil
			}
		}

		rt.runTempl(w, r, templ, &BlogPostIndexParams{
			Posts:   posts,
			Authors: authorMap,

			Page:     page,
			NumPages: numPages,
		})
	}
}

func (rt *Web) blogPost() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/view.html", "blogpost/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		post := util.BlogPost(r)
		var statement = []byte("Post is empty.")

		variants, err := rt.base.BlogPostDescVariants(r.Context(), post.ID, rt.base.IsBlogPostEditor(util.UserBrief(r), post))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get problem desc variants", slog.Any("err", err))
		}

		descVariant := rt.appropriateDescriptionVariant(r, variants)

		assetLink := fmt.Sprintf("/assets/blogPost/%s/attachment/%s?t=%d", post.Slug, rt.base.FormatDescName(descVariant), descVariant.LastUpdatedAt.UnixMilli())
		switch descVariant.Format {
		case "md":
			statement, err = rt.base.RenderedBlogPostDesc(r.Context(), post, descVariant)
			if err != nil {
				slog.WarnContext(
					r.Context(),
					"Error getting problem markdown",
					slog.Any("err", err),
					slog.Any("variant", descVariant),
					slog.Any("problem", post),
				)
				statement = []byte("Error loading markdown.")
			}
		case "pdf":
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh; background: white; object-fit: contain;"></embed>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link_post"), assetLink,
			))
		case "":
		default:
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link_post"),
			))
		}

		atts, err := rt.base.BlogPostAttachments(r.Context(), post.ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}

		if atts != nil {
			newAtts := make([]*kilonova.Attachment, 0, len(atts))
			for _, att := range atts {
				if att.Visible || rt.base.IsBlogPostEditor(util.UserBrief(r), post) {
					newAtts = append(newAtts, att)
				}
			}

			atts = newAtts
		}

		att, err := rt.base.BlogPostAttByName(r.Context(), post.ID, rt.base.FormatDescName(descVariant))
		if err != nil {
			att = nil
		}

		rt.runTempl(w, r, templ, &BlogPostParams{
			Topbar: rt.postTopbar(r, "view"),

			Attachments:  atts,
			Statement:    template.HTML(statement),
			StatementAtt: att,
			Variants:     variants,

			SelectedVariant: descVariant,
		})
	}
}

// TODO: Properly figure out priorities
func (rt *Web) getFinalVariant(prefLang string, prefType string, variants []*kilonova.StatementVariant) *kilonova.StatementVariant {
	var finalVariant *kilonova.StatementVariant

	for _, vr := range variants {
		if vr.Format == "md" && vr.Language == prefLang {
			if len(prefType) > 0 && vr.Type == prefType {
				return vr
			}
			if finalVariant == nil || finalVariant.Type > vr.Type {
				finalVariant = vr
			}
		}
	}

	if finalVariant != nil {
		return finalVariant
	}

	for _, vr := range variants {
		if vr.Format == "md" {
			if len(prefType) > 0 && vr.Type == prefType {
				return vr
			}
			if finalVariant == nil || finalVariant.Type > vr.Type {
				finalVariant = vr
			}
		}
	}

	if finalVariant == nil {
		return &kilonova.StatementVariant{
			Language: prefLang,
			Format:   "md",
			Type:     "",
		}
	}
	return finalVariant
}

func (rt *Web) editBlogPostIndex() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/editIndex.html", "modals/md_att_editor.html", "blogpost/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {

		variants, err := rt.base.BlogPostDescVariants(r.Context(), util.BlogPost(r).ID, true)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get blog post desc variants", slog.Any("err", err))
			http.Error(w, "Couldn't get statement variants", 500)
			return
		}

		finalVariant := rt.getFinalVariant(r.FormValue("pref_lang"), r.FormValue("pref_type"), variants)

		var statementData string
		var att *kilonova.Attachment
		att, err = rt.base.BlogPostAttByName(r.Context(), util.BlogPost(r).ID, rt.base.FormatDescName(finalVariant))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Couldn't get blog post content attachment", slog.Any("err", err))
			http.Error(w, "Couldn't get post content attachment", 500)
			return
		}
		if att != nil {
			val, err := rt.base.AttachmentData(r.Context(), att.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get blog post content attachment data", slog.Any("err", err))
				http.Error(w, "Couldn't get post content", 500)
				return
			}
			statementData = string(val)
		}

		rt.runTempl(w, r, templ, &BlogPostParams{
			Topbar: rt.postTopbar(r, "editIndex"),

			StatementEditor: &StatementEditorParams{
				Variants: variants,
				Variant:  finalVariant,
				Data:     statementData,
				Att:      att,

				APIPrefix: fmt.Sprintf("/blogPosts/%d", util.BlogPost(r).ID),
			},
		})
	}
}

func (rt *Web) editBlogPostAtts() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/editAttachments.html", "modals/att_manager.html", "blogpost/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := rt.base.BlogPostAttachments(r.Context(), util.BlogPost(r).ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		rt.runTempl(w, r, templ, &BlogPostParams{
			Topbar: rt.postTopbar(r, "editAttachments"),

			AttachmentEditor: &AttachmentEditorParams{
				Attachments: atts,
				BlogPost:    util.BlogPost(r),
				APIPrefix:   fmt.Sprintf("/blogPosts/%d", util.BlogPost(r).ID),
			},
		})
	}
}

func (rt *Web) contests() http.HandlerFunc {
	templ := rt.parse(nil, "contest/index.html", "modals/contest_brief.html", "contest/index_topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		filter := kilonova.ContestFilter{Look: true, LookingUser: util.UserBrief(r)}

		filter.Ordering = r.FormValue("ord")
		asc, err := strconv.ParseBool(r.FormValue("asc"))
		if err != nil {
			asc = false
		}
		filter.Ascending = asc

		if contestant := r.FormValue("contestant"); len(contestant) > 0 {
			user, err := rt.base.UserBriefByName(r.Context(), contestant)
			if err == nil {
				filter.ContestantID = &user.ID
			}
		}

		page := "all"
		switch v := r.FormValue("page"); v {
		case "virtual", "official":
			page = v
		case "personal":
			if !util.UserBrief(r).IsAuthed() {
				// Important to redirect and return, since we will dereference for ID later
				http.Redirect(w, r, "/contests?page=official", http.StatusTemporaryRedirect)
				return
			}
			page = v
		}
		switch page {
		case "all":
			// no additional filter
		case "official":
			filter.Type = kilonova.ContestTypeOfficial
		case "virtual":
			filter.Type = kilonova.ContestTypeVirtual
		case "personal":
			filter.ImportantContestsUID = &util.UserBrief(r).ID
		default:
			slog.WarnContext(r.Context(), "Unknown page type", slog.String("type", page))
		}

		cnt, err := rt.base.ContestCount(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contest count", slog.Any("err", err))
			cnt = -1
		}

		pageNum, err := strconv.Atoi(r.FormValue("p"))
		if err != nil {
			pageNum = 1
		}

		filter.Limit = 60
		filter.Offset = filter.Limit * (pageNum - 1)

		contests, err := rt.base.Contests(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contests", slog.Any("err", err))
			rt.statusPage(w, r, 400, "Nu am putut obține concursurile")
			return
		}

		rt.runTempl(w, r, templ, &ContestsIndexParams{
			Contests: contests,
			Page:     page,

			ContestCount: cnt,
			PageNum:      pageNum,
		})
	}
}

func (rt *Web) createContest() http.HandlerFunc {
	templ := rt.parse(nil, "contest/create.html", "contest/index_topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		if !(util.UserBrief(r).IsProposer() || flags.NormalUserVirtualContests.Value()) {
			rt.statusPage(w, r, 403, "Nu poți crea concursuri!")
			return
		}
		rt.runTempl(w, r, templ, &ContestsIndexParams{
			Contests: nil,
			Page:     "create",
		})
	}
}

func (rt *Web) contest() http.HandlerFunc {
	templ := rt.parse(nil, "contest/view.html", "problem/topbar.html", "modals/pbs.html", "modals/contest_sidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Topbar: rt.problemTopbar(r, "contest_general", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestEdit() http.HandlerFunc {
	templ := rt.parse(nil, "contest/edit.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		invitations, err := rt.base.ContestInvitations(r.Context(), util.Contest(r).ID)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get contest invitations", slog.Any("err", err))
			invitations = []*kilonova.ContestInvitation{}
		}

		mossSubs, err := rt.base.MOSSSubmissions(r.Context(), util.Contest(r).ID)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get MOSS submissions", slog.Any("err", err))
			mossSubs = []*kilonova.MOSSSubmission{}
		}

		rt.runTempl(w, r, templ, &ContestParams{
			Topbar: rt.problemTopbar(r, "contest_edit", -1),

			Contest: util.Contest(r),

			ContestInvitations: invitations,
			MOSSResults:        mossSubs,
		})
	}
}

func (rt *Web) contestInvite() http.HandlerFunc {
	templ := rt.parse(nil, "contest/invite.html")
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
			user, err := rt.base.UserBrief(r.Context(), *inv.CreatorID)
			if err == nil && user != nil {
				invCreator = user
			}
		}

		var alreadyRegistered bool
		reg, err := rt.base.ContestRegistration(r.Context(), contest.ID, util.UserBrief(r).ID)
		if err == nil && reg != nil {
			alreadyRegistered = true
		}

		rt.runTempl(w, r, templ, &ContestInviteParams{
			Contest: contest,
			Invite:  inv,
			Inviter: invCreator,

			AlreadyRegistered: alreadyRegistered,
		})
	}
}

func (rt *Web) contestInviteQRCode(w http.ResponseWriter, r *http.Request) {
	inv, err := rt.base.ContestInvitation(r.Context(), r.PathValue("inviteID"))
	if err != nil || inv == nil {
		slog.WarnContext(r.Context(), "Couldn't get contest invitation", slog.Any("err", err))
		rt.statusPage(w, r, 404, "Invite not found")
		return
	}

	inviteURL := config.Common.HostURL.JoinPath("contests/invite", inv.ID).String()
	data, err := qrcode.Encode(inviteURL, qrcode.High, 256)
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't encode QR code", slog.Any("err", err))
		http.Error(w, "Couldn't generate QR code", 500)
		return
	}

	http.ServeContent(w, r, "qr.png", inv.CreatedAt, bytes.NewReader(data))
}

func (rt *Web) contestCommunication() http.HandlerFunc {
	templ := rt.parse(nil, "contest/communication.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Topbar: rt.problemTopbar(r, "contest_communication", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestRegistrations() http.HandlerFunc {
	templ := rt.parse(nil, "contest/registrations.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Topbar: rt.problemTopbar(r, "contest_registrations", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestLeaderboard() http.HandlerFunc {
	templ := rt.parse(nil, "contest/leaderboard.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		// This is assumed to be called from a context in which
		// IsContestVisible is already true
		if !(util.Contest(r).PublicLeaderboard || util.Contest(r).IsEditor(util.UserBrief(r))) {
			rt.statusPage(w, r, 400, "You are not allowed to view the leaderboard")
			return
		}
		rt.runTempl(w, r, templ, &ContestParams{
			Topbar: rt.problemTopbar(r, "contest_leaderboard", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) graderInfo() http.HandlerFunc {
	templ := rt.parse(nil, "admin/grader.html")
	return func(w http.ResponseWriter, r *http.Request) {
		langs := rt.base.GraderLanguages(r.Context())
		rt.runTempl(w, r, templ, &GraderInfoParams{langs})
	}
}

func (rt *Web) donationPage() http.HandlerFunc {
	templ := rt.parse(nil, "donate.html")
	return func(w http.ResponseWriter, r *http.Request) {
		donations, err := rt.base.Donations(r.Context())
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get donations", slog.Any("err", err))
			donations = []*kilonova.Donation{}
		}

		rt.runTempl(w, r, templ, &DonateParams{
			Donations: donations,

			Status: r.FormValue("status"),
		})
	}
}

func (rt *Web) externalResources() http.HandlerFunc {
	templ := rt.parse(nil, "externalResources/index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		problemResources, err := rt.base.ExternalResources(r.Context(), kilonova.ExternalResourceFilter{
			ProblemID: &util.Problem(r).ID,
		})
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get problem resources", slog.Any("err", err))
			problemResources = []*kilonova.ExternalResource{}
		}
		rt.runTempl(w, r, templ, &ResourcesIndexParams{problemResources})
	}
}

func (rt *Web) externalResource() http.HandlerFunc {
	templ := rt.parse(nil, "externalResources/view.html")
	return func(w http.ResponseWriter, r *http.Request) {
		var author *kilonova.UserBrief
		if res := util.ExternalResource(r); res.ProposedBy != nil {
			if proposer, err := rt.base.UserBrief(r.Context(), *res.ProposedBy); err != nil {
				slog.WarnContext(r.Context(), "Couldn't get external resource author", slog.Any("err", err))
			} else {
				author = proposer
			}
		}
		rt.runTempl(w, r, templ, &ResourcesPageParams{
			util.ExternalResource(r),
			util.Problem(r),
			author,
		})
	}
}

func (rt *Web) createExternalResourceView() http.HandlerFunc {
	templ := rt.parse(nil, "externalResources/new.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ResourcesNewParams{util.Problem(r)})
	}
}

func (rt *Web) createExternalResource() http.HandlerFunc {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")
	return func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Name        string                `json:"name"`
			Description string                `json:"description"`
			URL         string                `json:"url"`
			Type        kilonova.ResourceType `json:"type"`
			PreApproved bool                  `json:"preApproved"`
		}
		r.ParseForm()
		if err := decoder.Decode(&params, r.Form); err != nil {
			htmxErrorToast(w, r, "Invalid request body")
			rt.statusPage(w, r, 400, "Invalid request body")
			return
		}
		if params.Name == "" || params.URL == "" || params.Type == "" {
			htmxErrorToast(w, r, "Invalid request body")
			rt.statusPage(w, r, 400, "Invalid request body")
			return
		}
		if params.PreApproved {
			params.PreApproved = util.UserBrief(r).IsAdmin()
		}

		res, err := rt.base.CreateExternalResource(
			r.Context(),
			params.Name, params.Description,
			params.URL,
			kilonova.ResourceType(params.Type),
			util.UserBrief(r),
			util.Problem(r),
			params.PreApproved,
		)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't create external resource", slog.Any("err", err))
			htmxErrorToast(w, r, "Couldn't create external resource")
			rt.statusPage(w, r, 500, "Couldn't create external resource")
			return
		}
		w.Header().Add("X-Resource-ID", fmt.Sprintf("%d", res.ID))
		htmxSuccessToast(w, r, "External resource created")
		w.Header().Add("HX-Redirect", fmt.Sprintf("/problems/%d", util.Problem(r).ID))
		http.Redirect(w, r, fmt.Sprintf("/problems/%d", util.Problem(r).ID), http.StatusOK)
	}
}

//func (rt *Web) updateExternalResource() http.HandlerFunc {
//	templ := rt.parse(nil, "externalResources/view.html")
//	return func(w http.ResponseWriter, r *http.Request) {
//		var author *kilonova.UserBrief
//		if res := util.ExternalResource(r); res.ProposedBy != nil {
//			if proposer, err := rt.base.UserBrief(r.Context(), *res.ProposedBy); err != nil {
//				slog.WarnContext(r.Context(), "Couldn't get external resource author", slog.Any("err", err))
//			} else {
//				author = proposer
//			}
//		}
//		rt.runTempl(w, r, templ, &ResourcesPageParams{
//			util.ExternalResource(r),
//			util.Problem(r),
//			author,
//		})
//	}
//}
//func (rt *Web) deleteExternalResource() http.HandlerFunc {
//	templ := rt.parse(nil, "externalResources/view.html")
//	return func(w http.ResponseWriter, r *http.Request) {
//		var author *kilonova.UserBrief
//		if res := util.ExternalResource(r); res.ProposedBy != nil {
//			if proposer, err := rt.base.UserBrief(r.Context(), *res.ProposedBy); err != nil {
//				slog.WarnContext(r.Context(), "Couldn't get external resource author", slog.Any("err", err))
//			} else {
//				author = proposer
//			}
//		}
//		rt.runTempl(w, r, templ, &ResourcesPageParams{
//			util.ExternalResource(r),
//			util.Problem(r),
//			author,
//		})
//	}
//}

func (rt *Web) profilePage(w http.ResponseWriter, r *http.Request, templ *template.Template, user *kilonova.UserFull) {
	if !(flags.ViewOtherProfiles.Value() || util.UserBrief(r).IsAdmin() || (util.UserBrief(r) != nil && util.UserBrief(r).ID == user.ID)) {
		rt.statusPage(w, r, 400, "You are not allowed to view other profiles!")
		return
	}

	solvedPbs, solvedCnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
		LookingUser: util.UserBrief(r), Look: true, LookFullyVisible: true,
		SolvedBy: &user.ID,

		Limit: 50,
	}, user.Brief(), util.UserBrief(r))
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't get solved problems", slog.Any("err", err))
		solvedPbs = []*sudoapi.FullProblem{}
	}

	attemptedPbs, attemptedCnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
		LookingUser: util.UserBrief(r), Look: true, LookFullyVisible: true,
		AttemptedBy: &user.ID,

		Limit: 50,
	}, user.Brief(), util.UserBrief(r))
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't get attempted problems", slog.Any("err", err))
		attemptedPbs = []*sudoapi.FullProblem{}
	}

	changeHistory, err := rt.base.UsernameChangeHistory(r.Context(), user.ID)
	if err != nil {
		changeHistory = []*kilonova.UsernameChange{}
	}

	rt.runTempl(w, r, templ, &ProfileParams{
		user, solvedPbs, solvedCnt, attemptedPbs, attemptedCnt, changeHistory,
	})
}

func (rt *Web) selfProfile() http.HandlerFunc {
	templ := rt.parse(nil, "profile.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.profilePage(w, r, templ, util.UserFull(r))
	}
}

func (rt *Web) profile() http.HandlerFunc {
	templ := rt.parse(nil, "profile.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimSpace(r.PathValue("user"))
		user, err := rt.base.UserFullByName(r.Context(), username)
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
			rt.statusPage(w, r, 500, "")
			return
		}
		if user == nil {
			user2, err := rt.base.HistoricalUsernameHolder(r.Context(), username)
			if err != nil || user2 == nil {
				rt.statusPage(w, r, 404, "")
			} else {
				http.Redirect(w, r, "/profile/"+user2.Name, http.StatusMovedPermanently)
			}
			return
		}

		rt.profilePage(w, r, templ, user)
	}
}

func (rt *Web) linkStatusPage(w http.ResponseWriter, r *http.Request, templ *template.Template, user *kilonova.UserFull) {
	dUser, err := rt.base.GetDiscordIdentity(r.Context(), user.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not get Discord identity", slog.Any("user", user), slog.Any("err", err))
		dUser = nil
	}
	rt.runTempl(w, r, templ, &DiscordLinkParams{
		ContentUser: user,
		DiscordUser: dUser,
	})
}

func (rt *Web) selfLinkStatus() http.HandlerFunc {
	templ := rt.parse(nil, "discordLink.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.linkStatusPage(w, r, templ, util.UserFull(r))
	}
}

func (rt *Web) linkStatus() http.HandlerFunc {
	templ := rt.parse(nil, "discordLink.html")
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(r.PathValue("user")))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
			rt.statusPage(w, r, 500, "")
			return
		}
		if user == nil {
			rt.statusPage(w, r, 404, "")
			return
		}
		// Only admins and that specific user can view their sessions
		if !(util.UserBrief(r).IsAdmin() || util.UserBrief(r).ID == user.ID) {
			rt.statusPage(w, r, 403, "")
		}

		rt.linkStatusPage(w, r, templ, user)
	}
}

func (rt *Web) userSessionsPage(w http.ResponseWriter, r *http.Request, templ *template.Template, user *kilonova.UserFull) {
	sessions, err := rt.base.UserSessions(r.Context(), user.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't get sessions", slog.Any("err", err))
		rt.statusPage(w, r, 500, err.Error())
		return
	}

	rt.runTempl(w, r, templ, &SessionsParams{
		ContentUser: user,
		Sessions:    sessions,
	})
}

func (rt *Web) selfSessions() http.HandlerFunc {
	templ := rt.parse(nil, "auth/sessions.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.userSessionsPage(w, r, templ, util.UserFull(r))
	}
}

func (rt *Web) userSessions() http.HandlerFunc {
	templ := rt.parse(nil, "auth/sessions.html")
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(r.PathValue("user")))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
			rt.statusPage(w, r, 500, "")
			return
		}
		if user == nil {
			rt.statusPage(w, r, 404, "")
			return
		}
		// Only admins and that specific user can view their sessions
		if !(util.UserBrief(r).IsAdmin() || util.UserBrief(r).ID == user.ID) {
			rt.statusPage(w, r, 403, "")
		}

		rt.userSessionsPage(w, r, templ, user)
	}
}

func (rt *Web) problemQueue() http.HandlerFunc {
	templ := rt.parse(nil, "admin/problemQueue.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, nil)
	}
}

func (rt *Web) sessionsFilter() http.HandlerFunc {
	templ := rt.parse(nil, "auth/sessions.html")
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")
	type filterQuery struct {
		ID         *string `json:"id"`
		UserID     *int    `json:"user_id"`
		UserPrefix *string `json:"user_prefix"`

		IPAddr   *string `json:"ip_addr"`
		IPPrefix *string `json:"ip_prefix"`

		Page int `json:"page"`

		Ordering  string `json:"ord"`
		Ascending bool   `json:"asc"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var q filterQuery
		r.ParseForm()
		if err := decoder.Decode(&q, r.Form); err != nil {
			slog.WarnContext(r.Context(), "Couldn't decode filter query", slog.Any("err", err))
		}
		if q.Page < 1 {
			q.Page = 1
		}

		f := &sudoapi.SessionFilter{
			ID: q.ID, UserID: q.UserID, UserPrefix: q.UserPrefix,

			Limit: 50, Offset: (q.Page - 1) * 50,
			Ordering: q.Ordering, Ascending: q.Ascending,
		}

		if q.IPAddr != nil {
			addr, err := netip.ParseAddr(*q.IPAddr)
			if err != nil {
				rt.statusPage(w, r, 400, "Invalid IP address: "+err.Error())
				return
			}
			f.IPAddr = &addr
		}
		if q.IPPrefix != nil {
			prefix, err := netip.ParsePrefix(*q.IPPrefix)
			if err != nil {
				rt.statusPage(w, r, 400, "Invalid IP prefix: "+err.Error())
				return
			}
			f.IPPrefix = &prefix
		}

		sessions, err := rt.base.Sessions(r.Context(), f)
		if err != nil {
			rt.statusPage(w, r, 500, err.Error())
			return
		}

		numSessions, err := rt.base.CountSessions(r.Context(), f)
		if err != nil {
			rt.statusPage(w, r, 500, err.Error())
			return
		}

		numPages := numSessions / 50
		if numSessions%50 > 0 {
			numPages++
		}

		rt.runTempl(w, r, templ, &SessionsParams{
			ContentUser: nil,
			Sessions:    sessions,
			Page:        q.Page,
			NumPages:    numPages,
		})
	}
}

func (rt *Web) resendEmail() http.HandlerFunc {
	templ := rt.parse(nil, "util/sent.html")
	return func(w http.ResponseWriter, r *http.Request) {
		u := util.UserFull(r)
		if u.VerifiedEmail {
			rt.statusPage(w, r, 403, "Deja ai verificat emailul!")
			return
		}
		t := time.Since(u.EmailVerifResent)
		if t < 5*time.Minute {
			text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", (5*time.Minute - t).Truncate(time.Millisecond))
			rt.statusPage(w, r, 403, text)
			return
		}
		if err := rt.base.SendVerificationEmail(context.WithoutCancel(r.Context()), u.ID, u.Name, u.Email, u.PreferredLanguage); err != nil {
			slog.WarnContext(r.Context(), "Could not resend verification email", slog.Any("err", err))
			rt.statusPage(w, r, 500, "N-am putut retrimite emailul de verificare")
			return
		}

		rt.runTempl(w, r, templ, struct{}{})
	}
}

func (rt *Web) verifyEmail() http.HandlerFunc {
	templ := rt.parse(nil, "verified-email.html")
	return func(w http.ResponseWriter, r *http.Request) {
		vid := r.PathValue("vid")
		if !rt.base.CheckVerificationEmail(r.Context(), vid) {
			rt.statusPage(w, r, 404, "")
			return
		}

		uid, err := rt.base.GetVerificationUser(r.Context(), vid)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get email verification", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		user, err := rt.base.UserBrief(r.Context(), uid)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get user", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		if err := rt.base.ConfirmVerificationEmail(context.WithoutCancel(r.Context()), vid, user); err != nil {
			slog.WarnContext(r.Context(), "Could not confirm email", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		// rebuild session for user to disable popup
		rt.initSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rt.runTempl(w, r, templ, &VerifiedEmailParams{user})
		})).ServeHTTP(w, r)
	}
}

func (rt *Web) resetPassword() http.HandlerFunc {
	templ := rt.parse(nil, "auth/forgot_pwd_reset.html")
	return func(w http.ResponseWriter, r *http.Request) {
		reqid := r.PathValue("reqid")
		if !rt.base.CheckPasswordResetRequest(r.Context(), reqid) {
			rt.statusPage(w, r, 404, "")
			return
		}

		uid, err := rt.base.GetPwdResetRequestUser(r.Context(), reqid)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get user from password reset request", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		user, err := rt.base.UserFull(r.Context(), uid)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get full user", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		rt.runTempl(w, r, templ, &PasswordResetParams{user, reqid})
	}
}

func (rt *Web) redirectDiscord(w http.ResponseWriter, r *http.Request) {
	id := flags.DiscordInviteID.Value()
	if id == "" {
		rt.statusPage(w, r, 500, "Discord was not configured on this instance")
		return
	}

	http.Redirect(w, r, "https://discord.gg/"+id, http.StatusFound)
}

func (rt *Web) checkLockout() func(next http.Handler) http.Handler {
	templ := rt.parse(nil, "util/lockout.html")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if flags.ForceLogin.Value() && !util.UserBrief(r).IsAuthed() {
				http.Redirect(w, r, "/login?back="+url.PathEscape(r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			if util.UserFull(r) != nil && util.UserFull(r).NameChangeForced {
				rt.runTempl(w, r, templ, struct{}{})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rt *Web) logout(w http.ResponseWriter, r *http.Request) {
	emptyCookie := &http.Cookie{
		Name:    "kn-sessionid",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, emptyCookie)

	c, err := r.Cookie("kn-sessionid")
	if err != nil {
		return
	}
	rt.base.RemoveSession(r.Context(), c.Value)

	redirect := "/"
	back := r.FormValue("back")
	backURL, err := url.Parse(back)
	if err == nil && backURL.Path != "" {
		redirect = backURL.Path
	}

	http.Redirect(w, r, redirect+"?logout=1", http.StatusTemporaryRedirect)
}

func (rt *Web) runTemplate(w io.Writer, r *http.Request, hTempl *template.Template, name string, data any) {
	hTempl, err := hTempl.Clone()
	if err != nil {
		fmt.Fprintf(w, "Error cloning template, report to admin: %s", err)
		return
	}

	// "cache" most util.* calls
	lang := util.Language(r)
	fullAuthedUser := util.UserFull(r)
	authedUser := util.UserBrief(r)
	var pblistCache map[int]int
	switch v := r.Context().Value(PblistCntCacheKey).(type) {
	case map[int]int:
		pblistCache = v
	}

	// Add request-specific functions
	hTempl.Funcs(template.FuncMap{
		"getText": func(line string, args ...any) string {
			return kilonova.GetText(lang, line, args...)
		},
		"pLanguages": func() map[string]string {
			return rt.base.EnabledLanguages()
		},
		"formatStmtVariant": func(fmt *kilonova.StatementVariant) string {
			var b strings.Builder
			b.Grow(32)
			switch fmt.Language {
			case "en":
				b.WriteString("🇬🇧 English")
			case "ro":
				b.WriteString("🇷🇴 Română")
			case "hu":
				b.WriteString("🇭🇺 Magyar")
			default:
				b.WriteString(fmt.Language)
			}

			if len(fmt.Type) > 0 {
				b.WriteString(" - ")
				switch fmt.Type {
				case "llm", "short", "long", "editorial":
					b.WriteString(kilonova.GetText(lang, "stmt_type."+fmt.Type))
				default:
					b.WriteString(cases.Title(language.English).String(fmt.Type))
				}
			}

			b.WriteString(" - ")
			switch fmt.Format {
			case "pdf":
				b.WriteString("PDF")
			case "md":
				b.WriteString("Markdown")
			case "tex":
				b.WriteString("LaTeX")
			default:
				b.WriteString(fmt.Format)
			}
			return b.String()
		},
		"reqPath": func() string {
			return reqPath(r)
		},
		"htmxRequest": func() bool {
			return isHTMXRequest(r)
		},
		"language": func() string {
			return lang
		},
		"authed": func() bool {
			return authedUser != nil
		},
		"contentUser": func() *kilonova.UserBrief {
			return util.ContentUserBrief(r)
		},
		"fullAuthedUser": func() *kilonova.UserFull {
			return fullAuthedUser
		},
		"authedUser": func() *kilonova.UserBrief {
			return authedUser
		},
		"isAdmin": authedUser.IsAdmin,
		"discordIdentity": func(user *kilonova.UserFull) *discordgo.User {
			dUser, err := rt.base.GetDiscordIdentity(r.Context(), user.ID)
			if err != nil {
				dUser = nil
			}
			return dUser
		},
		"currentProblem": func() *kilonova.Problem {
			return util.Problem(r)
		},
		"isContestEditor": func(c *kilonova.Contest) bool {
			return c.IsEditor(authedUser)
		},
		"genContestProblemsParams": func(pbs []*kilonova.ScoredProblem, contest *kilonova.Contest) *ProblemListingParams {
			return &ProblemListingParams{pbs, contest.IsEditor(authedUser) || contest.Ended(), true, contest.ID, -1}
		},
		"contestLeaderboardVisible": func(c *kilonova.Contest) bool {
			return rt.base.CanViewContestLeaderboard(authedUser, c)
		},
		"contestQuestions": func(c *kilonova.Contest) []*kilonova.ContestQuestion {
			questions, err := rt.base.ContestUserQuestions(r.Context(), c.ID, authedUser.ID)
			if err != nil {
				return []*kilonova.ContestQuestion{}
			}
			return questions
		},
		"contestRegistration": func(c *kilonova.Contest) *kilonova.ContestRegistration {
			if authedUser == nil || c == nil {
				return nil
			}
			reg, err := rt.base.ContestRegistration(r.Context(), c.ID, authedUser.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get contest registration", slog.Any("err", err))
				return nil
			}
			return reg
		},
		"problemFullyVisible": func() bool {
			return rt.base.IsProblemFullyVisible(util.UserBrief(r), util.Problem(r))
		},
		"numSolvedPblist": func(listID int) int {
			if pblistCache != nil {
				if val, ok := pblistCache[listID]; ok {
					return val
				}
			}
			slog.WarnContext(r.Context(), "Cache miss", slog.Int("listID", listID), slog.String("page", r.URL.Path), slog.Any("cache", pblistCache))
			cnt, err := rt.base.NumSolvedFromPblist(r.Context(), listID, authedUser.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get problem list number solved", slog.Any("err", err))
				return -1
			}
			return cnt
		},
		"pbParentPblists": func(problem *kilonova.Problem) []*kilonova.ProblemList {
			var topList *kilonova.ProblemList
			if r.FormValue("list_id") != "" {
				id, err := strconv.Atoi(r.FormValue("list_id"))
				if err == nil {
					list, err := rt.base.ProblemList(r.Context(), id)
					if err != nil {
						slog.WarnContext(r.Context(), "Couldn't get problem list", slog.Any("err", err))
					} else {
						var ok bool
						for _, pbid := range list.ProblemIDs() {
							if pbid == problem.ID {
								ok = true
								break
							}
						}
						if ok {
							topList = list
						}
					}
				}
			}
			lists, err := rt.base.ProblemParentLists(r.Context(), problem.ID, false)
			if err != nil {
				return nil
			}
			if topList != nil {
				for i := range lists {
					if lists[i].ID == topList.ID {
						lists = slices.Delete(lists, i, i+1)
						break // This assumes that there are no duplicates in the lists array
					}
				}
				lists = append([]*kilonova.ProblemList{topList}, lists...)
			}
			if len(lists) > 5 {
				lists = lists[:5]
			}
			return lists
		},
		"subCode": func(sub *kilonova.FullSubmission) []byte {
			code, err := rt.base.SubmissionCode(r.Context(), &sub.Submission, sub.Problem, util.UserBrief(r), true)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get submission code", slog.Any("err", err))
				code = nil
			}
			return code
		},
		"mustSolveCaptcha": func() bool {
			ip, _ := rt.base.GetRequestInfo(r)
			return rt.base.MustSolveCaptcha(r.Context(), ip)
		},
		"inModal": func() bool {
			return name == "modal"
		},
		"renderComponent": func(c templ.Component) (template.HTML, error) {
			return templ.ToGoHTML(r.Context(), c)
		},
	})

	if name != "" {
		if err := hTempl.ExecuteTemplate(w, name, data); err != nil {
			fmt.Fprintf(w, "Error executing template, report to admin: %s", err)
			slog.WarnContext(r.Context(), "Error executing template", slog.Any("err", err), slog.String("path", r.URL.Path), slog.Any("user", util.UserBrief(r)))
		}
		return
	}

	var title string
	if hTempl.Lookup("title") != nil {
		var titleBuf strings.Builder
		if err := hTempl.ExecuteTemplate(&titleBuf, "title", data); err != nil {
			slog.WarnContext(r.Context(), "Error executing title template", slog.Any("err", err), slog.String("path", r.URL.Path), slog.Any("user", util.UserBrief(r)))
		}
		title = titleBuf.String()
	}

	var description string
	if hTempl.Lookup("description") != nil {
		var descriptionBuf strings.Builder
		if err := hTempl.ExecuteTemplate(&descriptionBuf, "description", data); err != nil {
			slog.WarnContext(r.Context(), "Error executing description template", slog.Any("err", err), slog.String("path", r.URL.Path), slog.Any("user", util.UserBrief(r)))
		}
		description = descriptionBuf.String()
	}

	var head templ.Component = templ.NopComponent
	if hTempl.Lookup("head") != nil {
		head = templ.FromGoHTML(hTempl.Lookup("head"), data)
	}

	rt.runLayout(w, r, &LayoutParams{
		Title:       title,
		Description: description,
		Head:        head,
		Content:     templ.FromGoHTML(hTempl.Lookup("content"), data),
	})
}

type LayoutParams struct {
	Title       string
	Description string
	Head        templ.Component
	Content     templ.Component
}

func (rt *Web) runLayout(w io.Writer, r *http.Request, params *LayoutParams) {
	layoutParams := layout.LayoutParams{
		EnabledLanguages: rt.base.EnabledLanguages(),
		Title:            params.Title,
		Description:      params.Description,
		Navbar:           layout.Navbar(rt.canViewAllSubs(util.UserBrief(r)), reqPath(r)),
		Head:             params.Head,
		Content:          params.Content,
		HashNamer:        fsys,
		HideFooter:       false,
	}
	if layoutParams.Head == nil {
		layoutParams.Head = templ.NopComponent
	}

	if err := layout.Layout(layoutParams).Render(r.Context(), w); err != nil {
		slog.WarnContext(r.Context(), "Error rendering layout", slog.Any("err", err), slog.String("path", r.URL.Path), slog.Any("user", util.UserBrief(r)))
		fmt.Fprintf(w, "Error rendering layout, report to admin: %s", err)
	}
}

func (rt *Web) runEmptyPage(w io.Writer, r *http.Request, params *LayoutParams) {
	layoutParams := layout.LayoutParams{
		EnabledLanguages: rt.base.EnabledLanguages(),
		Title:            params.Title,
		Description:      params.Description,
		Navbar:           templ.NopComponent,
		Head:             params.Head,
		Content:          params.Content,
		HashNamer:        fsys,
		HideFooter:       true,
	}

	if err := layout.Layout(layoutParams).Render(r.Context(), w); err != nil {
		slog.WarnContext(r.Context(), "Error rendering layout", slog.Any("err", err), slog.String("path", r.URL.Path), slog.Any("user", util.UserBrief(r)))
		fmt.Fprintf(w, "Error rendering layout, report to admin: %s", err)
	}
}

func reqPath(r *http.Request) string {
	if r.URL.Path == "/login" || r.URL.Path == "/signup" {
		// when navigating between /login and /signup, retain back or just leave empty
		val := "/"
		if back := r.FormValue("back"); back != "" {
			link, err := url.Parse(back)
			if err == nil {
				val = link.Path
			}
		}
		return val
	}
	return r.URL.Path
}

// Runs template, but if hx_modal form parameter is "true" and template has "modal_content" template defined, can also return a modal
func (rt *Web) runTemplModal(w http.ResponseWriter, r *http.Request, templ *template.Template, data any) {
	if r.FormValue("hx_modal") == "true" && templ.Lookup("modal_content") != nil {
		rt.runModal(w, r, templ, "modal", &ModalParams{false, data})
		return
	}
	rt.runTempl(w, r, templ, data)
}

func (rt *Web) runTempl(w http.ResponseWriter, r *http.Request, templ *template.Template, data any) {
	rt.runTemplate(w, r, templ, "", data)
}

func (rt *Web) runModal(w http.ResponseWriter, r *http.Request, templ *template.Template, name string, data any) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	rt.runTemplate(w, r, templ, name, data)
}

func (rt *Web) componentModal(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		slog.WarnContext(r.Context(), "Error rendering modal", slog.Any("err", err))
	}
}

func (rt *Web) getOlderSubmissions(ctx context.Context, lookingUser *kilonova.UserBrief, userID int, problem *kilonova.Problem, contest *kilonova.Contest, limit int) (*modals.OlderSubmissionsParams, error) {
	var filter = kilonova.SubmissionFilter{
		UserID:    &userID,
		ProblemID: &problem.ID,
	}
	if contest != nil {
		filter.ContestID = &contest.ID
	}
	if limit > 0 {
		filter.Limit = limit
	}
	subs, err := rt.base.Submissions(ctx, filter, true, lookingUser)
	if err != nil {
		return nil, err
	}
	allFinished := true
	for _, sub := range subs.Submissions {
		if sub.Status != kilonova.StatusFinished {
			allFinished = false
		}
	}

	return &modals.OlderSubmissionsParams{
		UserID:  userID,
		Problem: problem,
		Contest: contest,
		Limit:   limit,

		Submissions: subs,
		NumHidden:   subs.Count - len(subs.Submissions),
		AllFinished: allFinished,
		AutoReload:  true,
	}, nil
}
