package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
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

	scores := map[int]int{}
	numSolved := -1
	if s.base.IsAuthed(util.UserBrief(r)) {
		numSolved = s.base.NumSolved(r.Context(), util.UserBrief(r).ID, list.List)
		scores = s.base.MaxScores(r.Context(), util.UserBrief(r).ID, list.List)
	}

	desc, err1 := s.base.RenderMarkdown([]byte(list.Description))
	if err1 != nil {
		errorData(w, err1, 500)
		return
	}

	returnData(w, struct {
		List          *kilonova.ProblemList `json:"list"`
		NumSolved     int                   `json:"numSolved"`
		Problems      []*kilonova.Problem   `json:"problems"`
		ProblemScores map[int]int           `json:"problemScores"`
		RenderedDesc  string                `json:"description"`
	}{
		List:          list,
		NumSolved:     numSolved,
		Problems:      pbs,
		ProblemScores: scores,
		RenderedDesc:  string(desc),
	})
	// returnData(w, list)
}

func (s *API) problemLists(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ProblemListFilter
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	lists, err := s.base.ProblemLists(r.Context(), args)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, lists)
}

func (s *API) initProblemList(w http.ResponseWriter, r *http.Request) {
	var listData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ProblemIDs  []int  `json:"ids"`
		SublistIDs  []int  `json:"sublists"`
	}
	if err := parseJsonBody(r, &listData); err != nil {
		err.WriteError(w)
		return
	}

	if listData.Title == "" || len(listData.ProblemIDs) == 0 {
		errorData(w, "Invalid problem list", 400)
		return
	}

	actualIDs, err := s.filterProblems(r.Context(), listData.ProblemIDs, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}
	if len(actualIDs) == 0 {
		errorData(w, "Invalid problems", 400)
		return
	}

	var list kilonova.ProblemList
	list.Title = listData.Title
	list.Description = listData.Description
	list.AuthorID = util.UserBrief(r).ID
	list.List = actualIDs
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

	returnData(w, list.ID)
}

func (s *API) updateProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID          int     `json:"id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		List        []int   `json:"list"`
		Sublists    []int   `json:"sublists"`
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
	}); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.List != nil {
		list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r))
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

func (s *API) filterProblems(ctx context.Context, problemIDs []int, user *kilonova.UserBrief) ([]int, *kilonova.StatusError) {
	if len(problemIDs) == 0 {
		return []int{}, nil
	}
	pbs, err := s.base.Problems(ctx, kilonova.ProblemFilter{IDs: problemIDs, LookingUser: user, Look: true})
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
