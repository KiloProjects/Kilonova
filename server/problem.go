package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/gosimple/slug"
)

func (s *API) setProblemVisible(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("visible") == "" {
		errorData(w, "`visible` not specified", http.StatusBadRequest)
		return
	}
	b, err := strconv.ParseBool(r.FormValue("visible"))
	if err != nil {
		errorData(w, "`visible` not valid bool", http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateProblemField(getContextValue(r, "pbID").(uint), "visible", b); err != nil {
		errorData(w, err.Error(), 500)
		return
	}
	returnData(w, "success", "Updated visibility status")
}

func (s *API) updateTitle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("A", r.ParseForm())
	val := r.FormValue("title")
	fmt.Println(val)
	if err := s.db.UpdateProblemField(getContextValue(r, "pbID").(uint), "name", val); err != nil {
		errorData(w, err.Error(), 500)
		return
	}
	returnData(w, "success", "Updated title")
}

func (s *API) updateDescription(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("text")
	if err := s.db.UpdateProblemField(getContextValue(r, "pbID").(uint), "text", val); err != nil {
		errorData(w, err.Error(), 500)
		return
	}
	returnData(w, "success", "Updated description")
}

func (s *API) updateTest(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func (s *API) getTests(w http.ResponseWriter, r *http.Request) {
	returnData(w, "success", common.ProblemFromContext(r).Tests)
}

func (s *API) getTest(w http.ResponseWriter, r *http.Request) {
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
	for _, t := range common.ProblemFromContext(r).Tests {
		if t.VisibleID == uint(id) {
			returnData(w, "success", t)
			return
		}
	}
	errorData(w, "Test not found (did put the visible ID?)", http.StatusBadRequest)
}

func (s *API) setInputType(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("isSet")
	problem := common.ProblemFromContext(r)
	if val == "true" {
		s.db.UpdateProblemField(problem.ID, "console_input", true)
	} else if val == "false" {
		s.db.UpdateProblemField(problem.ID, "console_input", false)
	} else {
		errorData(w, "Invalid value for `isSet` form value", http.StatusBadRequest)
	}
}

func (s *API) setTestName(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("testName")
	if val == "" {
		errorData(w, "You must set the `testName` form value", http.StatusBadRequest)
		return
	}
	problem := common.ProblemFromContext(r)
	if err := s.db.UpdateProblemField(problem.ID, "test_name", val); err != nil {
		errorData(w, err.Error(), 500)
		return
	}
	returnData(w, "success", "Updated test name")
}

func (s *API) purgeTests(w http.ResponseWriter, r *http.Request) {
	problem := common.ProblemFromContext(r)

	var tests = make([]common.Test, len(problem.Tests))
	copy(tests, problem.Tests)
	s.db.UpdateProblemField(problem.ID, "tests", []common.Test{})
	for _, t := range tests {
		fmt.Println("Trying to delete test with ID", t.ID)
		if err := s.db.DB.Delete(&t).Error; err != nil {
			// fixme(alexv): don't send the error
			errorData(w, err.Error(), 500)
		}
	}
	returnData(w, "success", "Purged all tests")
}

func (s *API) setLimits(w http.ResponseWriter, r *http.Request) {
	pb := common.ProblemFromContext(r)

	// in case limits is empty, set up the problem ID to save it to the DB

	if r.FormValue("memoryLimit") != "" {
		memoryLimit, err := strconv.ParseUint(r.FormValue("memoryLimit"), 10, 0)
		if err != nil || memoryLimit <= 0 {
			errorData(w, "memoryLimit must be a valid float", http.StatusBadRequest)
			return
		}
		fmt.Println("memory limit:", memoryLimit)
		pb.MemoryLimit = memoryLimit
	}
	if r.FormValue("stackLimit") != "" {
		stackLimit, err := strconv.ParseUint(r.FormValue("stackLimit"), 10, 0)
		if err != nil || stackLimit <= 0 {
			errorData(w, "stackLimit must be a valid float", http.StatusBadRequest)
			return
		}
		fmt.Println("stack limit:", stackLimit)
		pb.StackLimit = stackLimit
	}
	if r.FormValue("timeLimit") != "" {
		timeLimit, err := strconv.ParseFloat(r.FormValue("timeLimit"), 64)
		if err != nil || timeLimit <= 0 {
			errorData(w, "timeLimit must be a valid float", http.StatusBadRequest)
			return
		}
		fmt.Println("time limit:", timeLimit)
		pb.TimeLimit = timeLimit
	}
	if err := s.db.Save(&pb); err != nil {
		//s.errlog("API: /problem/%d/update/limits, db saving error: %s\n", pb.ID, err)
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, "success", "Limits saved")
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
		tests := common.ProblemFromContext(r).Tests
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

	var test common.Test
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
		fmt.Println("Hăăăăăă", err)
	}
	returnData(w, "success", test.ID)
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

	var problem common.Problem
	problem.Name = title
	problem.User = common.UserFromContext(r)
	problem.ConsoleInput = consoleInput
	problem.TestName = slug.Make(title)
	// default limits
	problem.MemoryLimit = 65536
	problem.StackLimit = 16384
	problem.TimeLimit = 0.1

	if err := s.db.Save(&problem); err != nil {
		//s.errlog("API: /problem/create, db saving error: %s\n", err)
		returnData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, "success", problem.ID)
}

// getAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) getAllProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := s.db.GetAllVisibleProblems(common.UserFromContext(r))
	if err != nil {
		//s.errlog("%s", err)
		returnData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, "success", problems)
}

func (s API) pbIDFromReq(r *http.Request) uint {
	return getContextValue(r, "pbID").(uint)
}

// getProblemByID returns a problem from the DB specified by ID
func (s *API) getProblemByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil {
		fmt.Fprintln(w, "Invalid ID")
		return
	}
	problem, err := s.db.GetProblemByID(uint(id))
	if err != nil {
		errorData(w, "Problem with ID doesn't exist", http.StatusBadRequest)
		return
	}
	returnData(w, "success", problem)
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
		errorData(w, err.Error(), 500)
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
	returnData(w, "success", ret)
}
