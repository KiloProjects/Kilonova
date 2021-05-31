package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/KiloProjects/kilonova/internal/util"
)

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
	if err := s.manager.SaveTestInput(util.Test(r).ID, bytes.NewReader([]byte(args.Input))); err != nil {
		errorData(w, err, 500)
		return
	}
	if err := s.manager.SaveTestOutput(util.Test(r).ID, bytes.NewReader([]byte(args.Output))); err != nil {
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
	if _, err := s.db.Test(r.Context(), util.Problem(r).ID, args.ID); err == nil {
		errorData(w, "Test with that visible id already exists!", 400)
		return
	}
	if err := s.db.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{VisibleID: &args.ID}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test id")
}

func (s *API) orphanTest(w http.ResponseWriter, r *http.Request) {
	if err := s.deleteTest(r.Context(), util.Problem(r).ID, util.Test(r).VisibleID); err != nil {
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
	if err := s.db.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{Score: &args.Score}); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Updated test score")
}

func (s *API) getTests(w http.ResponseWriter, r *http.Request) {
	tests, err := s.db.Tests(r.Context(), util.Problem(r).ID)
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

	test, err := s.db.Test(r.Context(), util.Problem(r).ID, args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, test)
}

func (s *API) purgeTests(w http.ResponseWriter, r *http.Request) {
	if err := s.db.OrphanProblemTests(r.Context(), util.Problem(r).ID); err != nil {
		errorData(w, err, 500)
		return
	}

	if err := s.db.DeleteSubTasks(r.Context(), util.Problem(r).ID); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Purged all tests")
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
		max, err := s.db.BiggestVID(r.Context(), util.Problem(r).ID)
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
	if err := s.db.CreateTest(r.Context(), &test); err != nil {
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

// TODO: Move most stuff to business logic
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

// deleteTest ensures that a test is deleted while also removing it from a subtask
func (s *API) deleteTest(ctx context.Context, pbid, testvid int) error {
	test, err := s.db.Test(ctx, pbid, testvid)
	if err != nil {
		return err
	}

	if err := s.db.OrphanProblemTest(ctx, pbid, testvid); err != nil {
		return err
	}

	stks, err := s.db.SubTasks(ctx, pbid)
	if err != nil {
		return err
	}
	for _, st := range stks {
		newIDs := make([]int, 0, len(st.Tests))
		for _, id := range st.Tests {
			if id == test.ID {
				continue
			}
			newIDs = append(newIDs, st.ID)
		}
		if len(newIDs) != len(st.Tests) {
			if len(newIDs) == 0 {
				if err := s.db.DeleteSubTask(ctx, st.ID); err != nil {
					return err
				}
			} else if err := s.db.UpdateSubTask(ctx, st.ID, kilonova.SubTaskUpdate{Tests: newIDs}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *API) bulkDeleteTests(w http.ResponseWriter, r *http.Request) {
	var removedTests int
	ids, ok := DecodeIntString(r.FormValue("tests"))
	if !ok || len(ids) == 0 {
		errorData(w, "Invalid input string", 400)
		return
	}
	for _, id := range ids {
		if err := s.deleteTest(r.Context(), util.Problem(r).ID, id); err == nil {
			removedTests++
		}
	}
	if removedTests != len(ids) {
		errorData(w, "Some tests could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected tests")
}

func (s *API) bulkUpdateTestScores(w http.ResponseWriter, r *http.Request) {
	var data map[int]int
	var updatedTests int

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&data); err != nil {
		log.Println(err)
		errorData(w, "Could not decode body", 400)
		return
	}
	for k, v := range data {
		if t, err := s.db.Test(r.Context(), util.Problem(r).ID, k); err == nil {
			if err := s.db.UpdateTest(r.Context(), t.ID, kilonova.TestUpdate{Score: &v}); err == nil {
				updatedTests++
			}
		}
	}
	if updatedTests != len(data) {
		errorData(w, "Some tests could not be updated", 500)
		return
	}
	returnData(w, "Updated all tests")
}
