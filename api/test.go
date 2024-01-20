package api

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/archive/test"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *API) saveTestData(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(20 * 1024 * 1024) // 20MB
	defer cleanupMultipart(r)

	if f, _, err := r.FormFile("input"); err == nil && f != nil {
		defer f.Close()
		if err := s.base.SaveTestInput(util.Test(r).ID, f); err != nil {
			errorData(w, err, 500)
			return
		}
	}
	if f, _, err := r.FormFile("output"); err == nil && f != nil {
		defer f.Close()
		if err := s.base.SaveTestOutput(util.Test(r).ID, f); err != nil {
			errorData(w, err, 500)
			return
		}
	}

	returnData(w, "Updated test data")
}

func (s *API) updateTestInfo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID    int
		Score string
	}
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
	scoreValue, err := decimal.NewFromString(args.Score)
	if err != nil {
		errorData(w, "Invalid score value", 400)
		return
	}

	if err := s.base.UpdateTest(r.Context(), util.Test(r).ID, kilonova.TestUpdate{VisibleID: &args.ID, Score: &scoreValue}); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Updated test info")
}

func (s *API) deleteTest(ctx context.Context, _ struct{}) *kilonova.StatusError {
	return s.base.DeleteTest(ctx, util.TestContext(ctx).ID)
}

func (s *API) getTests(ctx context.Context, _ struct{}) ([]*kilonova.Test, *kilonova.StatusError) {
	return s.base.Tests(ctx, util.ProblemContext(ctx).ID)
}

func (s *API) getTest(ctx context.Context, args struct{ ID int }) (*kilonova.Test, *kilonova.StatusError) {
	return s.base.Test(ctx, util.ProblemContext(ctx).ID, args.ID)
}

// createTest inserts a new test to the problem
// TODO: Move most stuff to logic
func (s *API) createTest(w http.ResponseWriter, r *http.Request) {
	score, err := strconv.ParseFloat(r.FormValue("score"), 64)
	if err != nil {
		errorData(w, "Score not float", http.StatusBadRequest)
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
	test.Score = decimal.NewFromFloat(score).Round(kilonova.MaxScoreRoundingPlaces)
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

func (s *API) processArchive(r *http.Request) *kilonova.StatusError {
	// Since this operation can take a lot of space, I am putting this lock as a precaution.
	// This might create a problem with timeouts, and this should be handled asynchronously.
	// (ie not in a request), but eh, I cant be bothered right now to do it the right way.
	// TODO: Do this the right way (low priority)
	s.testArchiveLock.Lock()
	defer s.testArchiveLock.Unlock()
	r.ParseMultipartForm(20 * 1024 * 1024)
	defer cleanupMultipart(r)

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return kilonova.Statusf(400, "Missing archive")
	}

	// Process zip file
	file, fh, err := r.FormFile("testArchive")
	if err != nil {
		zap.S().Warn(err)
		return kilonova.WrapError(err, "Couldn't open zip file")
	}
	defer file.Close()

	ar, err := zip.NewReader(file, fh.Size)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't read zip archive")
	}

	params := &test.TestProcessParams{
		Requestor:      util.UserFull(r),
		ScoreParamsStr: r.FormValue("scoreParameters"),
	}

	return test.ProcessZipTestArchive(context.Background(), util.Problem(r), ar, s.base, params)
}

func (s *API) processTestArchive(w http.ResponseWriter, r *http.Request) {
	if err := s.processArchive(r); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Processed tests")
}

func (s *API) bulkDeleteTests(w http.ResponseWriter, r *http.Request) {
	var removedTests int
	var testIDs []int
	if err := parseJSONBody(r, &testIDs); err != nil {
		err.WriteError(w)
		return
	}
	for _, id := range testIDs {
		if t, err := s.base.Test(r.Context(), util.Problem(r).ID, id); err == nil {
			if err := s.base.DeleteTest(r.Context(), t.ID); err == nil {
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
	var data map[int]decimal.Decimal
	var updatedTests int

	if err := parseJSONBody(r, &data); err != nil {
		err.WriteError(w)
		return
	}

	for k, v := range data {
		v := v
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
