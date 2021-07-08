package api

import (
	"io"
	"net/http"
	"strconv"

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
		if util.User(r) == nil {
			errorData(w, "No user specified", 400)
			return
		}
		args.UserID = util.User(r).ID
	}

	if args.ProblemID == 0 {
		errorData(w, "No problem specified", 400)
		return
	}

	returnData(w, s.db.MaxScore(r.Context(), args.UserID, args.ProblemID))
}

func (s *API) deleteProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.db.DeleteProblem(r.Context(), util.Problem(r).ID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Deleted problem")
}

// initProblem assigns an ID for the problem
// TODO: Move most stuff to logic
func (s *API) initProblem(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		errorData(w, "Title not provided", http.StatusBadRequest)
		return
	}

	cistr := r.FormValue("consoleInput")
	var consoleInput bool
	if cistr != "" {
		ci, err := strconv.ParseBool(cistr)
		if err != nil {
			errorData(w, "Invalid `consoleInput` form value", http.StatusBadRequest)
			return
		}
		consoleInput = ci
	}

	pb, err := s.db.Problems(r.Context(), kilonova.ProblemFilter{Name: &title})
	if len(pb) > 0 || err != nil {
		errorData(w, "Problem with specified title already exists in DB", http.StatusBadRequest)
		return
	}

	var problem kilonova.Problem
	problem.Name = title
	problem.AuthorID = util.User(r).ID
	problem.ConsoleInput = consoleInput
	if err := s.db.CreateProblem(r.Context(), &problem); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, problem.ID)
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

	var id int
	if util.User(r) != nil {
		id = util.User(r).ID
		if util.User(r).Admin {
			id = -1
		}
	}
	args.LookingUserID = &id

	problems, err := s.db.Problems(r.Context(), args)
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, problems)
}

// getTestData returns the test data from a specified test of a specified problem
// /problem/{id}/get/testData
// URL params:
//  - id - the test id
//  - noIn - if not empty, the input file won't be sent
//  - noOut - if not empty, the output file won't be sent
func (s *API) getTestData(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("id")
	if sid == "" {
		errorData(w, "You must specify a test ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(sid)
	if err != nil {
		errorData(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	if _, err := s.db.Test(r.Context(), util.Problem(r).ID, id); err != nil {
		errorData(w, "Test doesn't exist", 400)
		return
	}

	var ret struct {
		In  string `json:"in"`
		Out string `json:"out"`
	}
	if r.FormValue("noIn") == "" {
		in, err := s.manager.TestInput(id)
		if err != nil {
			errorData(w, err, 500)
			return
		}

		inText, err := io.ReadAll(in)
		if err != nil {
			errorData(w, err, 500)
			return
		}

		in.Close()
		ret.In = string(inText)
	}
	if r.FormValue("noOut") == "" {
		out, err := s.manager.TestOutput(id)
		if err != nil {
			errorData(w, err, 500)
			return
		}

		outText, err := io.ReadAll(out)
		if err != nil {
			errorData(w, err, 500)
			return
		}

		out.Close()
		ret.Out = string(outText)
	}
	returnData(w, ret)
}

func (s *API) updateProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Title        *string `json:"title"`
		Description  *string `json:"description"`
		ConsoleInput *bool   `json:"console_input"`
		TestName     *string `json:"test_name"`

		Type       kilonova.ProblemType `json:"type"`
		HelperCode *string              `json:"helper_code"`

		SourceCredits *string `json:"source_credits"`
		AuthorCredits *string `json:"author_credits"`

		TimeLimit   *float64 `json:"time_limit"`
		MemoryLimit *int     `json:"memory_limit"`
		StackLimit  *int     `json:"stack_limit"`

		DefaultPoints *int `json:"default_points"`

		Visible *bool `json:"visible"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Title != nil && *args.Title == "" {
		errorData(w, "Title can't be empty", 400)
		return
	}

	if args.Visible != nil && !util.User(r).Admin && *args.Visible != util.Problem(r).Visible {
		errorData(w, "You can't update visibility!", 403)
		return
	}

	if err := s.db.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{
		Name:         args.Title,
		Description:  args.Description,
		ConsoleInput: args.ConsoleInput,
		TestName:     args.TestName,

		Type:       args.Type,
		HelperCode: args.HelperCode,

		SourceCredits: args.SourceCredits,
		AuthorCredits: args.AuthorCredits,

		TimeLimit:   args.TimeLimit,
		MemoryLimit: args.MemoryLimit,
		StackLimit:  args.StackLimit,

		DefaultPoints: args.DefaultPoints,
		Visible:       args.Visible,
	}); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated problem")
}
