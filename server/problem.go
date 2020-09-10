package server

import (
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/gosimple/slug"
)

func (s *API) setProblemVisible(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Visible bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}
	if err := s.db.UpdateProblemField(getContextValue(r, "pbID").(uint), "visible", args.Visible); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated visibility status")
}

func (s *API) maxScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := util.IDFromContext(r, util.PbID)
	var args struct{ UserID uint }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	max, err := s.db.MaxScoreFor(args.UserID, id)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, max)
}

func (s *API) maxScoreSelf(w http.ResponseWriter, r *http.Request) {
	id := util.IDFromContext(r, util.PbID)
	uid := util.IDFromContext(r, util.UserID)
	max, err := s.db.MaxScoreFor(uint(uid), uint(id))
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, max)
}

func (s *API) updateTitle(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("title")
	if err := s.db.UpdateProblemField(getContextValue(r, "pbID").(uint), "name", val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated title")
}

func (s *API) updateDescription(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("text")
	if err := s.db.UpdateProblemField(getContextValue(r, "pbID").(uint), "text", val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated description")
}

func (s *API) saveTestData(w http.ResponseWriter, r *http.Request) {
	in := r.FormValue("input")
	out := r.FormValue("output")
	id, ok := getFormInt(w, r, "id")
	if !ok {
		return
	}
	if err := s.manager.SaveTest(util.IDFromContext(r, util.PbID), uint(id), []byte(in), []byte(out)); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test data")
}

func (s *API) updateTestID(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		NewID uint
		ID    uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateProblemTestVisibleID(util.IDFromContext(r, util.PbID), args.ID, args.NewID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test id")
}

func (s *API) updateTestScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Score int
		ID    uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateProblemTestScore(util.IDFromContext(r, util.PbID), args.ID, args.Score); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test score")
}

func (s *API) getTests(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.ProblemFromContext(r).Tests)
}

func (s *API) getTest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID uint }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	for _, t := range util.ProblemFromContext(r).Tests {
		if t.VisibleID == uint(args.ID) {
			returnData(w, t)
			return
		}
	}
	errorData(w, "Test not found (did put the visible ID?)", http.StatusBadRequest)
}

func (s *API) setInputType(w http.ResponseWriter, r *http.Request) {
	problem := util.ProblemFromContext(r)
	var args struct{ IsSet bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	s.db.UpdateProblemField(problem.ID, "console_input", args.IsSet)
}

func (s *API) setTestName(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("testName")
	if val == "" {
		errorData(w, "You must set the `testName` form value", http.StatusBadRequest)
		return
	}
	problem := util.ProblemFromContext(r)
	if err := s.db.UpdateProblemField(problem.ID, "test_name", val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test name")
}

func (s *API) purgeTests(w http.ResponseWriter, r *http.Request) {
	problem := util.ProblemFromContext(r)

	var tests = make([]models.Test, len(problem.Tests))
	copy(tests, problem.Tests)
	s.db.UpdateProblemField(problem.ID, "tests", []models.Test{})
	for _, t := range tests {
		s.logger.Println("Trying to delete test with ID", t.ID)
		if err := s.db.DB.Delete(&t).Error; err != nil {
			errorData(w, err, 500)
			return
		}
	}
	returnData(w, "Purged all tests")
}

func (s *API) setLimits(w http.ResponseWriter, r *http.Request) {
	pb := util.ProblemFromContext(r)

	// in case limits is empty, set up the problem ID to save it to the DB

	r.ParseForm()
	var args struct {
		MemoryLimit uint64  `schema:"memoryLimit"`
		StackLimit  uint64  `schema:"stackLimit"`
		TimeLimit   float64 `schema:"timeLimit"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	pb.MemoryLimit = args.MemoryLimit
	pb.StackLimit = args.StackLimit
	pb.TimeLimit = args.TimeLimit

	if err := s.db.Save(&pb); err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, "Limits saved")
}

// createTest inserts a new test to the problem
func (s *API) createTest(w http.ResponseWriter, r *http.Request) {
	// TODO: Change to files, or make both options interchangeable
	score, err := strconv.Atoi(r.FormValue("score"))
	if err != nil {
		errorData(w, "Score not integer", http.StatusBadRequest)
		return
	}
	var visibleID uint64
	if vID := r.FormValue("visibleID"); vID != "" {
		visibleID, err = strconv.ParseUint(vID, 10, 32)
		if err != nil {
			errorData(w, "Visible ID not int", http.StatusBadRequest)
			return
		}
	} else {
		// set it to be the largest visible id of a test + 1
		tests := util.ProblemFromContext(r).Tests
		var max uint = 0
		for _, t := range tests {
			if max < t.VisibleID {
				max = t.VisibleID
			}
		}
		if max == 0 {
			visibleID = 1
		} else {
			visibleID = uint64(max)
		}
	}

	var test models.Test
	test.ProblemID = s.pbIDFromReq(r)
	test.Score = score
	test.VisibleID = uint(visibleID)
	s.db.Save(&test)
	if err := s.manager.SaveTest(
		s.pbIDFromReq(r),
		uint(test.VisibleID),
		[]byte(r.FormValue("input")),
		[]byte(r.FormValue("output")),
	); err != nil {
		s.logger.Println("Couldn't create test", err)
		errorData(w, "Couldn't create test", 500)
		return
	}
	returnData(w, "Created test")
}

// initProblem assigns an ID for the problem
// TODO: Handle limits
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

	if s.db.ProblemExists(title) {
		errorData(w, "Problem with specified title already exists in DB", http.StatusBadRequest)
		return
	}

	var problem models.Problem
	problem.Name = title
	problem.User = util.UserFromContext(r)
	problem.ConsoleInput = consoleInput
	problem.TestName = slug.Make(title)
	// default limits
	problem.MemoryLimit = 65536
	problem.StackLimit = 16384
	problem.TimeLimit = 0.1

	if err := s.db.Save(&problem); err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, problem.ID)
}

// getAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) getAllProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := s.db.GetAllVisibleProblems(util.UserFromContext(r))
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, problems)
}

func (s API) pbIDFromReq(r *http.Request) uint {
	return getContextValue(r, "pbID").(uint)
}

// getProblemByID returns a problem from the DB specified by ID
func (s *API) getProblemByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil {
		errorData(w, "Invalid ID", 401)
		return
	}
	problem, err := s.db.GetProblemByID(uint(id))
	if err != nil {
		errorData(w, "Problem with ID doesn't exist", http.StatusBadRequest)
		return
	}
	returnData(w, problem)
}

// getTestData returns the test data from a specified test of a specified problem
// /problem/{id}/get/testData
// URL params:
//  - id - the test id
//  - noIn - if not null, the input file won't be sent
//  - noOut - if not null, the output file won't be sent
func (s API) getTestData(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("id")
	if sid == "" {
		errorData(w, "You must specify a test ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		errorData(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	in, out, err := s.manager.GetTest(s.pbIDFromReq(r), uint(id))
	if err != nil {
		errorData(w, err, 500)
		return
	}

	var ret struct {
		In  string `json:"in"`
		Out string `json:"out"`
	}
	if r.FormValue("noIn") == "" {
		ret.In = string(in)
	}
	if r.FormValue("noOut") == "" {
		ret.Out = string(out)
	}
	returnData(w, ret)
}
