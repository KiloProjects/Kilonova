package api

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) setProblemVisible(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Visible bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{Visible: &args.Visible}); err != nil {
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
	var args struct{ UserID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, s.sserv.MaxScore(r.Context(), args.UserID, util.Problem(r).ID))
}

func (s *API) maxScoreSelf(w http.ResponseWriter, r *http.Request) {
	returnData(w, s.sserv.MaxScore(r.Context(), util.User(r).ID, util.Problem(r).ID))
}

func (s *API) updateTitle(w http.ResponseWriter, r *http.Request) {
	val := strings.TrimSpace(r.FormValue("title"))
	if val == "" {
		errorData(w, "Title must not be empty", 400)
		return
	}
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{Name: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated title")
}

func (s *API) updateSourceCredits(w http.ResponseWriter, r *http.Request) {
	val := strings.TrimSpace(r.FormValue("credits"))
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{SourceCredits: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated source credits")
}

func (s *API) updateAuthorCredits(w http.ResponseWriter, r *http.Request) {
	val := strings.TrimSpace(r.FormValue("credits"))
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{AuthorCredits: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated author credits")
}

func (s *API) updateDescription(w http.ResponseWriter, r *http.Request) {
	val := kilonova.Sanitizer.Sanitize(r.FormValue("text"))
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{Description: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated description")
}

func (s *API) updateHelperCode(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("code")
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{HelperCode: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated helper code")
}

func (s *API) updateHelperCodeLang(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("lang")
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{HelperCodeLang: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated helper code language")
}

func (s *API) updateProblemType(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Type kilonova.ProblemType
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{Type: args.Type}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated problem type")
}

func (s *API) updateSubtaskString(w http.ResponseWriter, r *http.Request) {
	substr := r.FormValue("subtask_string")
	if !kilonova.SubtaskRegex.MatchString(substr) {
		errorData(w, "No subtask string found", 400)
		return
	}

	taskstr := kilonova.SubtaskRegex.FindString(substr)
	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{SubtaskString: &taskstr}); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, fmt.Sprintf("Set subtask string to %q", taskstr))
}

func (s *API) deleteProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	if err := s.pserv.DeleteProblem(r.Context(), args.ID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Deleted problem")
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
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	t, err := s.tserv.Test(r.Context(), util.Problem(r).ID, args.ID)
	if t != nil || err == nil {
		errorData(w, "Test with that visible id already exists!", 400)
		return
	}
	if err := s.tserv.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{VisibleID: &args.ID}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test id")
}

func (s *API) orphanTest(w http.ResponseWriter, r *http.Request) {
	var t = true
	if err := s.tserv.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{Orphaned: &t}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Removed test")
}

func (s *API) updateTestScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Score int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := s.tserv.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{Score: &args.Score}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test score")
}

func (s *API) getTests(w http.ResponseWriter, r *http.Request) {
	tests, err := s.tserv.Tests(r.Context(), util.Problem(r).ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, tests)
}

func (s *API) getTest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	test, err := s.tserv.Test(r.Context(), util.Problem(r).ID, args.ID)
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

	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{ConsoleInput: &args.IsSet}); err != nil {
		errorData(w, err, http.StatusInternalServerError)
		return
	}
	returnData(w, "Updated input type")
}

func (s *API) setTestName(w http.ResponseWriter, r *http.Request) {
	val := strings.TrimSpace(r.FormValue("testName"))
	if val == "" {
		errorData(w, "testName is empty", http.StatusBadRequest)
		return
	}

	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{TestName: &val}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test name")
}

func (s *API) purgeTests(w http.ResponseWriter, r *http.Request) {
	if err := s.tserv.OrphanProblemTests(r.Context(), util.Problem(r).ID); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Purged all tests")
}

func (s *API) setLimits(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		MemoryLimit int     `json:"memoryLimit"`
		StackLimit  int     `json:"stackLimit"`
		TimeLimit   float64 `json:"timeLimit"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{MemoryLimit: &args.MemoryLimit, StackLimit: &args.StackLimit, TimeLimit: &args.TimeLimit}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Limits saved")
}

func (s *API) setDefaultPoints(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		DefaultPoints int `json:"defaultPoints"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.pserv.UpdateProblem(r.Context(), util.Problem(r).ID, kilonova.ProblemUpdate{DefaultPoints: &args.DefaultPoints}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Default score changed")
}

// createTest inserts a new test to the problem
// TODO: Move most stuff to logic
func (s *API) createTest(w http.ResponseWriter, r *http.Request) {
	score, err := strconv.Atoi(r.FormValue("score"))
	if err != nil {
		errorData(w, "Score not integer", http.StatusBadRequest)
		return
	}
	var visibleID int
	if vID := r.FormValue("visibleID"); vID != "" {
		visibleID, err = strconv.Atoi(vID)
		if err != nil {
			errorData(w, "Visible ID not int", http.StatusBadRequest)
			return
		}
	} else {
		// set it to be the largest visible id of a test + 1
		max, err := s.tserv.BiggestVID(r.Context(), util.Problem(r).ID)
		if err != nil {
			max = 0
		}
		if max <= 0 {
			visibleID = 1
		} else {
			visibleID = max
		}
	}

	var test kilonova.Test
	test.ProblemID = util.Problem(r).ID
	test.VisibleID = visibleID
	test.Score = score
	if err := s.tserv.CreateTest(r.Context(), &test); err != nil {
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

	pb, err := s.pserv.Problems(r.Context(), kilonova.ProblemFilter{Name: &title})
	if len(pb) > 0 || err != nil {
		errorData(w, "Problem with specified title already exists in DB", http.StatusBadRequest)
		return
	}

	// default limits
	var problem kilonova.Problem
	problem.Name = title
	problem.AuthorID = util.User(r).ID
	problem.ConsoleInput = consoleInput
	if err := s.pserv.CreateProblem(r.Context(), &problem); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, problem.ID)
}

// getAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) getAllProblems(w http.ResponseWriter, r *http.Request) {
	var problems []*kilonova.Problem
	var err error
	if util.User(r).Admin {
		problems, err = s.pserv.Problems(r.Context(), kilonova.ProblemFilter{})
	} else {
		problems, err = s.pserv.Problems(r.Context(), kilonova.ProblemFilter{LookingUserID: &util.User(r).ID})
	}
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
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		errorData(w, "Invalid ID", 401)
		return
	}
	problem, err := s.pserv.ProblemByID(r.Context(), id)
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
	id, err := strconv.Atoi(sid)
	if err != nil {
		errorData(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	if _, err := s.tserv.Test(r.Context(), util.Problem(r).ID, id); err != nil {
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
