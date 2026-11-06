package web

import (
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/lists"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/tags"
	"github.com/gorilla/schema"
)

func (rt *Web) problems() http.HandlerFunc {
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

		ReviewRequested *bool `json:"review_requested"`

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

			for group := range strings.SplitSeq(*q.Tags, ",") {
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
			LookingUser: user.UserBrief(r), Look: true,
			FuzzyName: q.Query, EditorUserID: q.Editor, Visible: q.Visible,
			DeepListID: q.DeepListID, Language: lang,
			ReviewRequested: q.ReviewRequested,

			Tags:  gr,
			Limit: 50, Offset: (q.Page - 1) * 50,
			Ordering: q.Ordering, Descending: q.Descending,
		}, user.UserBrief(r), user.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Could not search problems", slog.Any("err", err))
			// TODO: Maybe not fail to load and instead just load on the browser?
			rt.statusPage(w, r, 500, "N-am putut încărca problemele")
			return
		}
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "problems"),
			Head:  ui.CanonicalURL("/problems"),
			Content: lists.ProblemSearch(lists.SearchParams{
				ProblemList: pblist, Results: pbs, Groups: gr, GroupTags: tags, Count: cnt,
			}),
		})
	}
}

func (rt *Web) tags() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		byType := make(map[kilonova.TagType][]*kilonova.Tag, len(tags.Types))
		for _, t := range tags.Types {
			list, err := rt.base.TagsByType(r.Context(), t)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get tags", slog.String("tag_type", string(t)))
			}
			byType[t] = list
		}
		u := user.UserBrief(r)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.tags"),
			Content: tags.Index(r.Context(), tags.IndexParams{Tags: byType, CanCreate: u.IsAuthed() && u.Proposer}),
		})
	}
}

func (rt *Web) tag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pbs, pbsCnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
			LookingUser: user.UserBrief(r), Look: true, LookFullyVisible: true,
			Tags: []*kilonova.TagGroup{{TagIDs: []int{util.Tag(r).ID}}},

			Limit: 50,
		}, user.UserBrief(r), user.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't fetch tag problems", slog.Any("err", err))
			pbs = []*sudoapi.FullProblem{}
		}

		relevantTags, err := rt.base.RelevantTags(r.Context(), util.Tag(r).ID, 15, user.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't fetch relevant tags", slog.Any("err", err))
			relevantTags = nil
		}
		u := user.UserBrief(r)
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "tag_header") + " " + util.Tag(r).Name,
			Content: tags.Tag(r.Context(), tags.TagParams{
				Tag: util.Tag(r), RelevantTags: relevantTags, Problems: pbs, ProblemCount: pbsCnt,
				Authed:    u.IsAuthed(),
				CanEdit:   u.IsAuthed() && u.Proposer,
				CanDelete: u.IsAuthed() && u.Proposer && u.Admin,
			}),
		})
	}
}

// withSubListIDs is the IDs of lists and their direct sublists, for buildPblistCache.
func withSubListIDs(pblists []*kilonova.ProblemList) []int {
	listIDs := []int{}
	for _, list := range pblists {
		listIDs = append(listIDs, list.ID)
		for _, slist := range list.SubLists {
			listIDs = append(listIDs, slist.ID)
		}
	}
	return listIDs
}

func (rt *Web) pbListIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pblists, err := rt.base.ProblemLists(r.Context(), kilonova.ProblemListFilter{Root: true})
		if err != nil {
			rt.statusPage(w, r, 500, "Eroare la obținerea listelor")
			return
		}

		r = rt.buildPblistCache(r, withSubListIDs(pblists))

		u := user.UserBrief(r)
		p := lists.IndexParams{Authed: u != nil, Others: rt.unassociatedProblems(r.Context(), u)}
		for _, list := range pblists {
			p.Lists = append(p.Lists, rt.pblistGroup(r, list, false))
		}
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "lists"),
			Head:    ui.CanonicalURL("/problem_lists"),
			Content: lists.Index(r.Context(), p),
		})
	}
}

func (rt *Web) pbListProgressIndex() http.HandlerFunc {
	t := true
	return func(w http.ResponseWriter, r *http.Request) {
		pblists, err := rt.base.ProblemLists(r.Context(), kilonova.ProblemListFilter{FeaturedChecklist: &t})
		if err != nil {
			rt.statusPage(w, r, 500, "Eroare la obținerea listelor")
			return
		}

		r = rt.buildPblistCache(r, withSubListIDs(pblists))

		ids := make([]int, 0, len(pblists))
		for _, list := range pblists {
			ids = append(ids, list.ID)
		}
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "progress_overview"),
			Content: lists.ProgressIndex(r.Context(), lists.ProgressIndexParams{
				RootList: flags.RootProblemList.Value(), Lists: pblists,
				Authed: user.UserBrief(r) != nil, NumSolved: rt.numSolvedPblists(r, ids),
			}),
		})
	}
}

func (rt *Web) pbListProgressView() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var checkedUser = user.UserBrief(r)
		if uname := r.FormValue("username"); len(uname) > 0 {
			userBrief, err := rt.base.UserBriefByName(r.Context(), uname)
			if err == nil {
				checkedUser = userBrief
			}
		}

		list, err := rt.base.FullProblemList(r.Context(), util.ProblemList(r).ID, checkedUser, user.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get problem list", slog.Any("err", err))
			rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
			return
		}
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.progressChecklist") + " | " + list.Title,
			Content: lists.Checklist(r.Context(), lists.ProgressParams{List: list, CheckedUser: checkedUser}),
		})
	}
}

func (rt *Web) pbListView() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list := util.ProblemList(r)
		listIDs := []int{list.ID}
		for _, slist := range list.SubLists {
			listIDs = append(listIDs, slist.ID)
		}

		r = rt.buildPblistCache(r, listIDs)

		if isHTMXRequest(r) {
			rt.componentModal(w, r, problems.ListGroup(r.Context(), rt.pblistGroup(r, list, true)))
			return
		}

		ctx, u := r.Context(), user.UserBrief(r)
		p := lists.ViewParams{
			List:    list,
			Group:   rt.pblistGroup(r, list, true),
			Authed:  u != nil,
			CanEdit: u != nil && (u.ID == list.AuthorID || u.IsAdmin()),
			IsAdmin: u.IsAdmin(),
		}
		if parents, err := rt.base.PblistParentLists(ctx, list.ID); err == nil {
			p.Parents = parents[:min(len(parents), 5)]
		}
		if list.AuthorID != 0 {
			if author, err := rt.base.UserBrief(ctx, list.AuthorID); err == nil {
				p.Author = author
			}
		}
		if p.IsAdmin {
			listProblems, err := rt.base.ProblemListProblems(ctx, list.List, u)
			if err != nil {
				slog.ErrorContext(ctx, "Could not get problems", slog.Any("err", err))
			}
			p.ProblemSources = modals.ProblemSources(ctx, list, listProblems)
		}

		rt.runLayout(w, r, &LayoutParams{
			Title:   list.Title,
			Head:    ui.CanonicalURL("/problem_lists/" + strconv.Itoa(list.ID)),
			Content: lists.View(ctx, p),
		})
	}
}
