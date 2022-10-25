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

	var list kilonova.ProblemList
	list.Title = listData.Title
	list.Description = listData.Description
	list.AuthorID = util.UserBrief(r).ID
	list.List = actualIDs
	if err := s.base.CreateProblemList(r.Context(), &list); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, list.ID)
}

func (s *API) updateProblemList(w http.ResponseWriter, r *http.Request) {
	// TODO: Nu merge, trebuie schimbat la json body thing
	r.ParseForm()
	var args struct {
		ID          int     `json:"id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		List        []int   `json:"list"`
	}
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

	if len(args.List) > 0 {
		list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r))
		if err != nil {
			err.WriteError(w)
			return
		}

		if err := s.base.UpdateProblemListProblems(r.Context(), args.ID, list); err != nil {
			errorData(w, err, 500)
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
		errorData(w, err, 500)
		return
	}
	returnData(w, "Removed problem list")
}

func (s *API) filterProblems(ctx context.Context, problemIDs []int, user *kilonova.UserBrief) ([]int, *kilonova.StatusError) {
	if len(problemIDs) == 0 {
		return []int{}, nil
	}
	pbs, err := s.base.Problems(ctx, kilonova.ProblemFilter{IDs: problemIDs, LookingUser: user})
	if err != nil {
		return nil, err
	}
	actualIDs := make([]int, 0, len(pbs))
	for _, pb := range pbs {
		actualIDs = append(actualIDs, pb.ID)
	}
	return actualIDs, nil
}
