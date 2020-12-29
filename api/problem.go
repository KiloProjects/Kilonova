package api

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/KiloProjects/Kilonova/internal/util"
)

func (s *API) setProblemVisible(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Visible bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}
	if err := util.Problem(r).SetVisibility(args.Visible); err != nil {
		errorData(w, err, 500)
		return
	}
	if args.Visible {
		returnData(w, "Made visible")
	} else {
		returnData(w, "Made invisible")
	}
}

func (s *API) maxScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ UserID int64 }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, util.Problem(r).MaxScore(args.UserID))
}

func (s *API) maxScoreSelf(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.User(r).MaxScore(util.ID(r, util.PbID)))
}

func (s *API) updateTitle(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("title")
	if val == "" {
		errorData(w, "Title must not be empty", 400)
		return
	}
	if err := util.Problem(r).SetName(val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated title")
}

func (s *API) updateCredits(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("credits")
	if err := util.Problem(r).SetCredits(val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated credits")
}

func (s *API) updateDescription(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("text")
	if err := util.Problem(r).SetDescription(val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated description")
}

func (s *API) saveTestData(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Input  string
		Output string
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := s.manager.SaveTestInput(util.Test(r).ID, bytes.NewBufferString(args.Input)); err != nil {
		errorData(w, err, 500)
		return
	}
	if err := s.manager.SaveTestOutput(util.Test(r).ID, bytes.NewBufferString(args.Output)); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test data")
}

func (s *API) updateTestID(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int64 }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := util.Test(r).SetVID(args.ID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test id")
}

func (s *API) updateTestScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Score int32 }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := util.Test(r).SetScore(args.Score); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test score")
}

func (s *API) getTests(w http.ResponseWriter, r *http.Request) {
	tests, err := util.Problem(r).Tests()
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, tests)
}

func (s *API) getTest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int64 }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	test, err := s.db.Test(r.Context(), util.Problem(r).ID, args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, test)
}

func (s *API) setInputType(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ IsSet bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := util.Problem(r).SetConsoleInput(args.IsSet); err != nil {
		errorData(w, err, http.StatusInternalServerError)
		return
	}
	returnData(w, "Updated input type")
}

func (s *API) setTestName(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("testName")
	if val == "" {
		errorData(w, "testName is empty", http.StatusBadRequest)
		return
	}

	if err := util.Problem(r).SetTestName(val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test name")
}

func (s *API) purgeTests(w http.ResponseWriter, r *http.Request) {
	if err := util.Problem(r).ClearTests(); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Purged all tests")
}

func (s *API) setLimits(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		MemoryLimit int32   `schema:"memoryLimit"`
		StackLimit  int32   `schema:"stackLimit"`
		TimeLimit   float64 `schema:"timeLimit"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := util.Problem(r).SetLimits(args.MemoryLimit, args.StackLimit, args.TimeLimit); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Limits saved")
}

// createTest inserts a new test to the problem
// TODO: Move most stuff to logic
func (s *API) createTest(w http.ResponseWriter, r *http.Request) {
	score, err := strconv.Atoi(r.FormValue("score"))
	if err != nil {
		errorData(w, "Score not integer", http.StatusBadRequest)
		return
	}
	var visibleID int64
	if vID := r.FormValue("visibleID"); vID != "" {
		visibleID, err = strconv.ParseInt(vID, 10, 32)
		if err != nil {
			errorData(w, "Visible ID not int", http.StatusBadRequest)
			return
		}
	} else {
		// set it to be the largest visible id of a test + 1
		max, err := util.Problem(r).BiggestVID()
		if err != nil {
			max = 0
		}
		if max <= 0 {
			visibleID = 1
		} else {
			visibleID = int64(max)
		}
	}

	test, err := s.db.CreateTest(r.Context(), util.Problem(r).ID, visibleID, int32(score))
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if err := s.manager.SaveTestInput(test.ID, bytes.NewBufferString(r.FormValue("input"))); err != nil {
		log.Println("Couldn't create test input", err)
		errorData(w, "Couldn't create test input", 500)
		return
	}
	if err := s.manager.SaveTestOutput(test.ID, bytes.NewBufferString(r.FormValue("output"))); err != nil {
		log.Println("Couldn't create test output", err)
		errorData(w, "Couldn't create test output", 500)
		return
	}
	returnData(w, "Created test")
}

type testPair struct {
	InFile  io.Reader
	OutFile io.Reader
	Score   int
}

type archiveCtx struct {
	tests        map[int64]testPair
	hasScoreFile bool
	scoredTests  []int64
}

// TODO: Move most stuff to logic
func (s *API) processTestArchive(w http.ResponseWriter, r *http.Request) {
	// Since this operation can take at most 100MB, I am putting this lock as a precaution.
	// This might create a problem with timeouts, and this should be handled asynchronously.
	// (ie not in a request), but eh, I cant be bothered right now to do it the right way.
	// TODO: Do this the right way (low priority)
	s.testArchiveLock.Lock()
	defer s.testArchiveLock.Unlock()
	r.ParseMultipartForm(100 * 1024 * 1024)

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		errorData(w, "Missing archive", 400)
		return
	}

	// Process zip file
	file, fh, err := r.FormFile("testArchive")
	if err != nil {
		log.Println(err)
		errorData(w, err.Error(), 400)
		return
	}
	defer file.Close()

	ar, err := zip.NewReader(file, fh.Size)
	if err != nil {
		errorData(w, logic.ErrBadArchive, 400)
		return
	}

	if err := s.kn.ProcessZipTestArchive(util.Problem(r), ar); err != nil {
		errorData(w, err, 400)
		return
	}

	returnData(w, "Processed tests")
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

	pb, err := s.db.ProblemByName(r.Context(), title)
	if pb != nil || err == nil {
		errorData(w, "Problem with specified title already exists in DB", http.StatusBadRequest)
		return
	}

	// default limits
	pb, err = s.db.CreateProblem(r.Context(), title, util.User(r).ID, consoleInput)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, pb.ID)
}

// getAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) getAllProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := s.db.VisibleProblems(r.Context(), util.User(r))
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, problems)
}

func (s *API) pbIDFromReq(r *http.Request) uint {
	return getContextValue(r, "pbID").(uint)
}

// getProblemByID returns a problem from the DB specified by ID
func (s *API) getProblemByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		errorData(w, "Invalid ID", 401)
		return
	}
	problem, err := s.db.Problem(r.Context(), id)
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
//  - noIn - if not empty, the input file won't be sent
//  - noOut - if not empty, the output file won't be sent
func (s *API) getTestData(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("id")
	if sid == "" {
		errorData(w, "You must specify a test ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(sid, 10, 64)
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
