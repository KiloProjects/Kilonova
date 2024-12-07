package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) getProblemList(ctx context.Context, _ struct{}) (*kilonova.ProblemList, *kilonova.StatusError) {
	return util.ProblemListContext(ctx), nil
}

// if there are multiple, will return the one with the smallest ID
func (s *API) problemListByName(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Name string }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	list, err := s.base.ProblemListByName(r.Context(), args.Name)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, list)
}

func (s *API) getComplexProblemList(w http.ResponseWriter, r *http.Request) {
	list := util.ProblemList(r)

	pbs, err := s.base.ProblemListProblems(r.Context(), list.List, util.UserBrief(r))
	if err != nil {
		errorData(w, err, 500)
		return
	}

	numSolved := -1
	numSubSolved := map[int]int{}
	if util.UserBrief(r).IsAuthed() {
		listIDs := []int{list.ID}
		for _, sublists := range list.SubLists {
			listIDs = append(listIDs, sublists.ID)
		}
		numSubSolved, err = s.base.NumSolvedFromPblists(r.Context(), listIDs, util.UserBrief(r))
		if err != nil {
			slog.WarnContext(r.Context(), "NumSolvedFromPblists fail", slog.Any("err", err))
			numSubSolved = map[int]int{}
		}
		if val, ok := numSubSolved[list.ID]; ok {
			numSolved = val
		}
	}

	desc, err1 := s.base.RenderMarkdown([]byte(list.Description), nil)
	if err1 != nil {
		errorData(w, err1, 500)
		return
	}

	returnData(w, struct {
		List         *kilonova.ProblemList     `json:"list"`
		NumSolved    int                       `json:"numSolved"`
		Problems     []*kilonova.ScoredProblem `json:"problems"`
		RenderedDesc string                    `json:"description"`
		NumSubSolved map[int]int               `json:"numSubSolved"`
	}{
		List:         list,
		NumSolved:    numSolved,
		Problems:     pbs,
		RenderedDesc: string(desc),
		NumSubSolved: numSubSolved,
	})
	// returnData(w, list)
}

func (s *API) problemLists(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Root bool `json:"root"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	lists, err := s.base.ProblemLists(r.Context(), kilonova.ProblemListFilter{Root: args.Root})
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, lists)
}

type pblistInitReturn struct {
	ID     int  `json:"id"`
	Nested bool `json:"nested"`
}

func (s *API) initProblemList(w http.ResponseWriter, r *http.Request) {
	var listData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ProblemIDs  []int  `json:"ids"`
		SublistIDs  []int  `json:"sublists"`
		ParentID    *int   `json:"parent_id"`

		SidebarHidable *bool `json:"sidebar_hidable"`
	}
	if err := parseJSONBody(r, &listData); err != nil {
		err.WriteError(w)
		return
	}

	if listData.Title == "" {
		errorData(w, "Invalid problem list", 400)
		return
	}

	actualIDs, err := s.filterProblems(r.Context(), listData.ProblemIDs, util.UserBrief(r), false, false)
	if err != nil {
		err.WriteError(w)
		return
	}

	var list kilonova.ProblemList
	list.Title = listData.Title
	list.Description = listData.Description
	list.AuthorID = util.UserBrief(r).ID
	list.List = actualIDs
	if listData.SidebarHidable != nil {
		list.SidebarHidable = *listData.SidebarHidable
	}
	if err := s.base.CreateProblemList(r.Context(), &list); err != nil {
		err.WriteError(w)
		return
	}

	if len(listData.SublistIDs) > 0 {
		if err := s.base.UpdateProblemListSublists(r.Context(), list.ID, listData.SublistIDs); err != nil {
			err.WriteError(w)
			return
		}
	}

	// This is a totally optional step that, if it fails, will just not nest the list automatically
	if listData.ParentID != nil {
		parent, err := s.base.ProblemList(r.Context(), *listData.ParentID)
		if err != nil {
			returnData(w, pblistInitReturn{
				ID:     list.ID,
				Nested: false,
			})
			return
		}
		var ids []int
		for _, sublist := range parent.SubLists {
			ids = append(ids, sublist.ID)
		}
		ids = append(ids, list.ID)
		if err := s.base.UpdateProblemListSublists(r.Context(), *listData.ParentID, ids); err != nil {
			slog.WarnContext(r.Context(), "Couldn't update sublists", slog.Any("err", err))
			returnData(w, pblistInitReturn{
				ID:     list.ID,
				Nested: false,
			})
			return
		}

		returnData(w, pblistInitReturn{
			ID:     list.ID,
			Nested: true,
		})
		return
	}

	returnData(w, pblistInitReturn{
		ID:     list.ID,
		Nested: false,
	})
}

func (s *API) updateProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		List        []int   `json:"list"`
		Sublists    []int   `json:"sublists"`

		SidebarHidable    *bool `json:"sidebar_hidable"`
		FeaturedChecklist *bool `json:"featured_checklist"`
	}
	if err := parseJSONBody(r, &args); err != nil {
		err.WriteError(w)
		return
	}
	orgList := util.ProblemList(r)

	if !(util.UserBrief(r).Admin || util.UserBrief(r).ID == orgList.AuthorID) {
		errorData(w, "You can't update this problem list!", 403)
		return
	}

	if err := s.base.UpdateProblemList(r.Context(), orgList.ID, kilonova.ProblemListUpdate{
		Title:       args.Title,
		Description: args.Description,

		SidebarHidable:    args.SidebarHidable,
		FeaturedChecklist: args.FeaturedChecklist,
	}); err != nil && !errors.Is(err, kilonova.ErrNoUpdates) {
		errorData(w, err, 500)
		return
	}

	if args.List != nil {
		list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r), false, false)
		if err != nil {
			err.WriteError(w)
			return
		}

		if err := s.base.UpdateProblemListProblems(r.Context(), orgList.ID, list); err != nil {
			err.WriteError(w)
			return
		}
	}

	if args.Sublists != nil {
		if err := s.base.UpdateProblemListSublists(r.Context(), orgList.ID, args.Sublists); err != nil {
			err.WriteError(w)
			return
		}
	}

	returnData(w, "Updated problem list")
}

func (s *API) deleteProblemList(w http.ResponseWriter, r *http.Request) {
	list := util.ProblemList(r)

	if !(util.UserBrief(r).Admin || util.UserBrief(r).ID == list.AuthorID) {
		errorData(w, "You can't delete this problem list!", 403)
		return
	}

	if err := s.base.DeleteProblemList(r.Context(), list.ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Removed problem list")
}

func (s *API) filterProblems(ctx context.Context, problemIDs []int, user *kilonova.UserBrief, filterEditor bool, filterFullyVisible bool) ([]int, *kilonova.StatusError) {
	if len(problemIDs) == 0 {
		return []int{}, nil
	}
	pbs, err := s.base.Problems(ctx, kilonova.ProblemFilter{IDs: problemIDs, LookingUser: user, Look: true, LookEditor: filterEditor, LookFullyVisible: filterFullyVisible})
	if err != nil {
		return nil, err
	}

	// Do this in order to maintain problemIDs order.
	// Necessary for problem list ordering
	available := make(map[int]bool)
	for _, pb := range pbs {
		available[pb.ID] = true
	}

	actualIDs := make([]int, 0, len(problemIDs))
	for _, pb := range problemIDs {
		if _, ok := available[pb]; ok {
			actualIDs = append(actualIDs, pb)
		}
	}
	return actualIDs, nil
}
