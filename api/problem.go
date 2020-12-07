package api

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

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
	if err := util.Problem(r).SetName(val); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated title")
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
	var args struct {
		Input  string
		Output string
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if err := s.manager.SaveTest(util.ID(r, util.PbID), util.ID(r, util.TestID), []byte(args.Input), []byte(args.Output)); err != nil {
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
		errorData(w, "You must set the `testName` form value", http.StatusBadRequest)
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

	pbID := util.ID(r, util.PbID)
	if _, err := s.db.CreateTest(r.Context(), pbID, visibleID, int32(score)); err != nil {
		errorData(w, err, 500)
		return
	}

	if err := s.manager.SaveTest(
		pbID,
		visibleID,
		[]byte(r.FormValue("input")),
		[]byte(r.FormValue("output")),
	); err != nil {
		log.Println("Couldn't create test", err)
		errorData(w, "Couldn't create test", 500)
		return
	}
	returnData(w, "Created test")
}

type testPair struct {
	InFile  fs.File
	OutFile fs.File
	Score   int
}

type archiveCtx struct {
	tests        map[int64]testPair
	hasScoreFile bool
	scoredTests  []int64
}

func getFirstInt(s string) int64 {
	var poz int
	for poz < len(s) && s[poz] >= '0' && s[poz] <= '9' {
		poz++
	}
	val, err := strconv.ParseInt(s[:poz], 10, 64)
	if err != nil {
		return -1
	}
	return val
}

var errBadTestFile = errors.New("Bad test score file")
var errBadArchive = errors.New("Bad archive")

func analyzeFile(ctx *archiveCtx, r *zip.Reader, name string, file fs.File) error {
	if name == "test.txt" || name == "tests.txt" {
		ctx.hasScoreFile = true

		data, err := io.ReadAll(file)
		if err != nil {
			return errBadArchive
		}

		lines := strings.Split(string(data), "\n")
		if len(lines) > 256 { // impose a hard limit on test lines
			return errBadTestFile
		}

		for _, line := range lines {
			vals := strings.Fields(line)
			if line == "" { // empty line, skip
				continue
			}
			if len(vals) != 2 {
				return errBadTestFile
			}
			testID, err := strconv.ParseInt(vals[0], 10, 64)
			if err != nil || testID < 0 {
				return errBadTestFile
			}
			score, err := strconv.Atoi(vals[1])
			if err != nil {
				return errBadTestFile
			}
			test := ctx.tests[testID]
			test.Score = score
			ctx.tests[testID] = test
			for _, ex := range ctx.scoredTests {
				if ex == testID {
					return errBadTestFile
				}
			}
			log.Println(testID)
			ctx.scoredTests = append(ctx.scoredTests, testID)
		}
		return nil
	}
	tid := getFirstInt(name)
	if tid == -1 {
		return errBadArchive
	}
	if strings.HasSuffix(name, ".in") {
		tf := ctx.tests[tid]
		if tf.InFile != nil {
			return errBadArchive
		}

		tf.InFile = file
		ctx.tests[tid] = tf
	}
	if strings.HasSuffix(name, ".out") || strings.HasSuffix(name, ".ok") {
		tf := ctx.tests[tid]
		if tf.OutFile != nil {
			return errBadArchive
		}

		tf.OutFile = file
		ctx.tests[tid] = tf
	}
	return nil
}

// TODO: Move most stuff to logic
func (s *API) processTestArchive(w http.ResponseWriter, r *http.Request) {
	// Since this operation can take at most 100MB, I am putting this lock as a precaution.
	// This might create a problem with timeouts, and this should be handled asynchronously.
	// (ie not in a request), but eh, I cant be bothered right now to do it the right way.
	// TODO: Do this the right way (low priority)
	s.testArchiveLock.Lock()
	defer s.testArchiveLock.Unlock()
	r.ParseMultipartForm(100 * 1024 * 1024) // 100MB, I should document this hard limit sometime TODO (low priority)

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		errorData(w, "Missing archive", 400)
		return
	}

	ctx := archiveCtx{}
	ctx.tests = make(map[int64]testPair)
	ctx.scoredTests = make([]int64, 0, 10)

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
		errorData(w, errBadArchive, 400)
		return
	}
	for _, file := range ar.File {
		if file.FileInfo().IsDir() {
			continue
		}
		f, err := ar.Open(file.Name)
		if err != nil {
			log.Println(err)
			errorData(w, "Unknown error", 500)
			return
		}
		defer f.Close() // This will always close all files, regardless of when the program leaves
		if err := analyzeFile(&ctx, ar, path.Base(file.Name), f); err != nil {
			errorData(w, err.Error(), 400)
			return
		}
	}

	if len(ctx.scoredTests) != len(ctx.tests) {
		errorData(w, errBadArchive, 400)
		return
	}

	for _, v := range ctx.tests {
		if v.InFile == nil || v.OutFile == nil {
			errorData(w, errBadArchive, 400)
			return
		}
	}

	if !ctx.hasScoreFile {
		errorData(w, "Missing test score file", 400)
		return
	}

	// If we are loading an archive, the user might want to remove all tests first
	// So let's do it for them
	if err := util.Problem(r).ClearTests(); err != nil {
		log.Println(err)
		errorData(w, err, 500)
		return
	}

	for testID, v := range ctx.tests {
		inFile, err := ioutil.ReadAll(v.InFile)
		if err != nil {
			log.Println(err)
			errorData(w, err, 500)
			return
		}

		outFile, err := ioutil.ReadAll(v.OutFile)
		if err != nil {
			log.Println(err)
			errorData(w, err, 500)
			return
		}

		pbID := util.ID(r, util.PbID)
		if _, err := s.db.CreateTest(r.Context(), pbID, testID, int32(v.Score)); err != nil {
			log.Println(err)
			errorData(w, err, 500)
			return
		}

		if err := s.manager.SaveTest(
			pbID,
			testID,
			inFile,
			outFile,
		); err != nil {
			log.Println("Couldn't create test", err)
			errorData(w, "Couldn't create test", 500)
			return
		}
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
//  - noIn - if not null, the input file won't be sent
//  - noOut - if not null, the output file won't be sent
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

	in, out, err := s.manager.Test(util.ID(r, util.PbID), id)
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
