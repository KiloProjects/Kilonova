package api

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/archive/test"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

func (s *API) saveTestData(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Input  *string
		Output *string
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if args.Input != nil {
		if err := s.base.SaveTestInput(util.Test(r).ID, strings.NewReader(*args.Input)); err != nil {
			errorData(w, err, 500)
			return
		}
	}
	if args.Output != nil {
		if err := s.base.SaveTestOutput(util.Test(r).ID, strings.NewReader(*args.Output)); err != nil {
			errorData(w, err, 500)
			return
		}
	}
	returnData(w, "Updated test data")
}

func (s *API) updateTestInfo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID, Score int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	if util.Test(r).VisibleID != args.ID {
		if t, err := s.base.Test(r.Context(), util.Problem(r).ID, args.ID); err == nil && t != nil {
			errorData(w, "Test with that visible id already exists!", 400)
			return
		}
	}
	if err := s.base.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{VisibleID: &args.ID, Score: &args.Score}); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Updated test info")
}

func (s *API) orphanTest(w http.ResponseWriter, r *http.Request) {
	if err := s.base.OrphanTest(r.Context(), util.Test(r).ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Removed test")
}

func (s *API) getTests(w http.ResponseWriter, r *http.Request) {
	tests, err := s.base.Tests(r.Context(), util.Problem(r).ID)
	if err != nil {
		err.WriteError(w)
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

	test, err := s.base.Test(r.Context(), util.Problem(r).ID, args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, test)
}

func (s *API) purgeTests(w http.ResponseWriter, r *http.Request) {
	if err := s.base.OrphanTests(r.Context(), util.Problem(r).ID); err != nil {
		err.WriteError(w)
		return
	}

	// NOTE: This may be redundant? Not fully sure
	if err := s.base.DeleteSubTasks(r.Context(), util.Problem(r).ID); err != nil {
		err.WriteError(w)
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
		visibleID = s.base.NextVID(r.Context(), util.Problem(r).ID)
	}

	var test kilonova.Test
	test.ProblemID = util.Problem(r).ID
	test.VisibleID = visibleID
	test.Score = score
	if err := s.base.CreateTest(r.Context(), &test); err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.SaveTestInput(test.ID, bytes.NewBufferString(r.FormValue("input"))); err != nil {
		zap.S().Warn("Couldn't create test input", err)
		errorData(w, "Couldn't create test input", 500)
		return
	}
	if err := s.base.SaveTestOutput(test.ID, bytes.NewBufferString(r.FormValue("output"))); err != nil {
		zap.S().Warn("Couldn't create test output", err)
		errorData(w, "Couldn't create test output", 500)
		return
	}
	returnData(w, "Created test")
}

func (s *API) processTestArchive(w http.ResponseWriter, r *http.Request) {
	// Since this operation can take a lot of space, I am putting this lock as a precaution.
	// This might create a problem with timeouts, and this should be handled asynchronously.
	// (ie not in a request), but eh, I cant be bothered right now to do it the right way.
	// TODO: Do this the right way (low priority)
	s.testArchiveLock.Lock()
	defer s.testArchiveLock.Unlock()
	r.ParseMultipartForm(20 * 1024 * 1024)

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		errorData(w, "Missing archive", 400)
		return
	}

	// Process zip file
	file, fh, err := r.FormFile("testArchive")
	if err != nil {
		zap.S().Warn(err)
		errorData(w, err.Error(), 400)
		return
	}
	defer file.Close()

	ar, err := zip.NewReader(file, fh.Size)
	if err != nil {
		errorData(w, test.ErrBadArchive, 400)
		return
	}

	if err := test.ProcessZipTestArchive(context.Background(), util.Problem(r), ar, s.base); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Processed tests")
}

func (s *API) bulkDeleteTests(w http.ResponseWriter, r *http.Request) {
	var removedTests int
	var testIDs []int
	if err := parseJsonBody(r, &testIDs); err != nil {
		err.WriteError(w)
		return
	}
	for _, id := range testIDs {
		if t, err := s.base.Test(r.Context(), util.Problem(r).ID, id); err == nil {
			if err := s.base.OrphanTest(r.Context(), t.ID); err == nil {
				removedTests++
			}
		}
	}
	if err := s.base.CleanupSubTasks(context.Background(), util.Problem(r).ID); err != nil {
		errorData(w, "Couldn't clean up subtasks", 500)
		return
	}

	if removedTests != len(testIDs) {
		errorData(w, "Some tests could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected tests")
}

func (s *API) bulkUpdateTestScores(w http.ResponseWriter, r *http.Request) {
	var data map[int]int
	var updatedTests int

	if err := parseJsonBody(r, &data); err != nil {
		err.WriteError(w)
		return
	}

	for k, v := range data {
		if t, err := s.base.Test(r.Context(), util.Problem(r).ID, k); err == nil {
			if err := s.base.UpdateTest(r.Context(), t.ID, kilonova.TestUpdate{Score: &v}); err == nil {
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
