package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) maxScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		UserID    int
		ProblemID int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.UserID == 0 {
		if util.UserBrief(r) == nil {
			errorData(w, "No user specified", 400)
			return
		}
		args.UserID = util.UserBrief(r).ID
	}

	if args.ProblemID == 0 {
		errorData(w, "No problem specified", 400)
		return
	}

	returnData(w, s.base.MaxScore(r.Context(), args.UserID, args.ProblemID))
}

func (s *API) deleteProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.base.DeleteProblem(r.Context(), util.Problem(r).ID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Deleted problem")
}

// initProblem assigns an ID for the problem
// TODO: Move most stuff to logic
func (s *API) initProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Title        string `json:"title"`
		ConsoleInput bool   `json:"consoleInput"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	id, err := s.base.CreateProblem(r.Context(), args.Title, util.UserBrief(r), args.ConsoleInput)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, id)
}

// getProblems returns all the problems from the DB matching a filter
// TODO: Pagination
func (s *API) getProblems(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ProblemFilter
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	args.LookingUser = util.UserBrief(r)

	problems, err := s.base.Problems(r.Context(), args)
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, problems)
}

func (s *API) updateProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ProblemUpdate
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if err := s.base.UpdateProblem(r.Context(), util.Problem(r).ID, args, util.UserBrief(r)); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated problem")
}
