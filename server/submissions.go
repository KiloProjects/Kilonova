package server

import (
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/proto"
	"github.com/KiloProjects/Kilonova/internal/util"
	"gorm.io/gorm"
)

// GetSubmissionByID returns a submission based on an ID
func (s *API) getSubmissionByID(w http.ResponseWriter, r *http.Request) {
	subID, ok := getFormInt(w, r, "id")
	if !ok {
		return
	}

	sub, err := s.db.GetSubmissionByID(uint(subID))
	if err != nil {
		errorData(w, "Could not find submission", http.StatusBadRequest)
		return
	}

	if !util.IsSubmissionVisible(*sub, util.UserFromContext(r)) {
		sub.SourceCode = ""
	}

	returnData(w, *sub)
}

// getSubmissions returns all Submissions from the DB
// TODO: Pagination and filtering
func (s *API) getSubmissions(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	subs, err := s.db.GetAllSubmissions()
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}

	user := util.UserFromContext(r)
	for i := 0; i < len(subs); i++ {
		if !util.IsSubmissionVisible(subs[i], user) {
			subs[i].SourceCode = ""
		}
	}
	returnData(w, subs)
}

func (s *API) getSubmissionsForProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		PID uint
		UID uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	subs, err := s.db.UserSubmissionsOnProblem(args.UID, args.PID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	user, err := s.db.GetUserByID(args.UID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	for i := 0; i < len(subs); i++ {
		if !util.IsSubmissionVisible(subs[i], *user) {
			subs[i].SourceCode = ""
		}
	}

	returnData(w, subs)
}

func (s *API) getSelfSubmissionsForProblem(w http.ResponseWriter, r *http.Request) {
	pid, ok := getFormInt(w, r, "pid")
	if !ok {
		return
	}
	uid := util.UserFromContext(r).ID
	subs, err := s.db.UserSubmissionsOnProblem(uint(uid), uint(pid))
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, subs)
}

func (s *API) setSubmissionVisible(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Visible bool
		ID      uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	sub, err := s.db.GetSubmissionByID(args.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorData(w, "Submission not found", http.StatusNotFound)
			return
		}
		s.logger.Println(err)
		errorData(w, err, http.StatusNotFound)
		return
	}

	if !util.IsSubmissionEditor(*sub, util.UserFromContext(r)) {
		errorData(w, "You are not allowed to do this", 403)
		return
	}

	if err := s.db.UpdateSubmissionVisibility(args.ID, args.Visible); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated visibility status")
}

// submissionSend registers a submission to be sent to the Eval handler
// Required values:
//	- code=[sourcecode] - source code of the submission, mutually exclusive with file uploads
//  - file=[file] - multipart file, mutually exclusive with the code param
//  - lang=[language] - language key like in common.Languages
//  - problemID=[problem] - problem ID that the submission will be associated with
// Note that the `code` param is prioritized over file upload
func (s *API) submissionSend(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Code      string
		Lang      string
		ProblemID uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	var user = util.UserFromContext(r)

	problem, err := s.db.GetProblemByID(args.ProblemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		return
	}

	if _, ok := proto.Languages[args.Lang]; ok == false {
		errorData(w, "Invalid language", http.StatusBadRequest)
		return
	}

	// figure out if the code is in a file or in a form value
	if args.Code == "" {
		if r.MultipartForm == nil {
			errorData(w, "No code sent", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			errorData(w, "Could not read file", http.StatusBadRequest)
			return
		}

		if problem.SourceSize != 0 && header.Size > problem.SourceSize {
			errorData(w, "File too large", http.StatusBadRequest)
			return
		}

		// Everything should be ok now
		c, err := ioutil.ReadAll(file)
		if err != nil {
			errorData(w, "Could not read file", http.StatusBadRequest)
			return
		}

		args.Code = string(c)
		if args.Code == "" {
			if r.MultipartForm == nil {
				errorData(w, "No code sent", http.StatusBadRequest)
				return
			}
		}
	}

	// create the evalTests
	var evalTests = make([]models.EvalTest, 0)
	for _, test := range problem.Tests {
		evTest := models.EvalTest{
			UserID: user.ID,
			Test:   test,
		}
		s.db.Save(&evTest)
		evalTests = append(evalTests, evTest)
	}

	// add the submission to the DB
	sub := models.Submission{
		Tests:      evalTests,
		User:       user,
		Problem:    *problem,
		SourceCode: args.Code,
		Language:   args.Lang,
	}
	if err := s.db.Save(&sub); err != nil {
		errorData(w, "Couldn't create test", 500)
		return
	}

	statusData(w, "success", sub.ID, http.StatusCreated)
}
