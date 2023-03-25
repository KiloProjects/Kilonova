package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

func (s *API) getProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	list, err := s.base.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, list)
}

func (s *API) getComplexProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	list, err := s.base.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	pbs, err := s.base.ProblemListProblems(r.Context(), list.List, util.UserBrief(r))
	if err != nil {
		errorData(w, err, 500)
		return
	}

	numSolved := -1
	numSubSolved := map[int]int{}
	if s.base.IsAuthed(util.UserBrief(r)) {
		numSolved, err = s.base.NumSolvedFromPblist(r.Context(), list.ID, util.UserBrief(r).ID)
		if err != nil {
			zap.S().Warn(err)
		}
		for _, sublist := range list.SubLists {
			numSolved, err := s.base.NumSolvedFromPblist(r.Context(), sublist.ID, util.UserBrief(r).ID)
			if err != nil {
				zap.S().Warn(err)
			}
			numSubSolved[sublist.ID] = numSolved
		}
	}

	desc, err1 := s.base.RenderMarkdown([]byte(list.Description))
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

	lists, err := s.base.ProblemLists(r.Context(), args.Root)
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
	if err := parseJsonBody(r, &listData); err != nil {
		err.WriteError(w)
		return
	}

	if listData.Title == "" {
		errorData(w, "Invalid problem list", 400)
		return
	}

	actualIDs, err := s.filterProblems(r.Context(), listData.ProblemIDs, util.UserBrief(r), false)
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
			zap.S().Warn(err)
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
		ID          int     `json:"id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		List        []int   `json:"list"`
		Sublists    []int   `json:"sublists"`

		SidebarHidable *bool `json:"sidebar_hidable"`
	}
	if err := parseJsonBody(r, &args); err != nil {
		err.WriteError(w)
		return
	}
	list, err := s.base.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, "Couldn't find problem list", 400)
		return
	}

	if !(util.UserBrief(r).Admin || util.UserBrief(r).ID == list.AuthorID) {
		errorData(w, "You can't update this problem list!", 403)
		return
	}

	if err := s.base.UpdateProblemList(r.Context(), args.ID, kilonova.ProblemListUpdate{
		Title:       args.Title,
		Description: args.Description,

		SidebarHidable: args.SidebarHidable,
	}); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.List != nil {
		list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r), false)
		if err != nil {
			err.WriteError(w)
			return
		}

		if err := s.base.UpdateProblemListProblems(r.Context(), args.ID, list); err != nil {
			err.WriteError(w)
			return
		}
	}

	if args.Sublists != nil {
		if err := s.base.UpdateProblemListSublists(r.Context(), args.ID, args.Sublists); err != nil {
			err.WriteError(w)
			return
		}
	}

	returnData(w, "Updated problem list")
}

func (s *API) deleteProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	list, err := s.base.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, "Couldn't find problem list", 400)
		return
	}

	if !(util.UserBrief(r).Admin || util.UserBrief(r).ID == list.AuthorID) {
		errorData(w, "You can't delete this problem list!", 403)
		return
	}

	if err := s.base.DeleteProblemList(r.Context(), args.ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Removed problem list")
}

func (s *API) filterProblems(ctx context.Context, problemIDs []int, user *kilonova.UserBrief, filterEditor bool) ([]int, *kilonova.StatusError) {
	if len(problemIDs) == 0 {
		return []int{}, nil
	}
	pbs, err := s.base.Problems(ctx, kilonova.ProblemFilter{IDs: problemIDs, LookingUser: user, Look: true, LookEditor: filterEditor})
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
