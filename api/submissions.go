package api

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/languages"
	"github.com/KiloProjects/Kilonova/internal/util"
)

// getSubmissionByID returns a submission based on an ID
func (s *API) getSubmissionByID(w http.ResponseWriter, r *http.Request) {
	subID, ok := getFormInt(w, r, "id")
	if !ok {
		return
	}

	sub, err := s.db.Submission(r.Context(), subID)
	if err != nil {
		errorData(w, "Could not find submission", http.StatusBadRequest)
		return
	}

	if !util.IsSubmissionVisible(sub, util.User(r)) {
		sub.Code = ""
	}

	returnData(w, sub)
}

// getSubmissions returns all Submissions from the DB
func (s *API) getSubmissions(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	subs, err := s.db.Submissions(r.Context())
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}

	user := util.User(r)
	for i := range subs {
		if !util.IsSubmissionVisible(subs[i], user) {
			subs[i].Code = ""
		}
	}
	returnData(w, subs)
}

func (s *API) getSubmissionsForProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		PID int64
		UID int64
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	subs, err := s.db.UserProblemSubmissions(r.Context(), db.UserProblemSubmissionsParams{UserID: args.UID, ProblemID: args.PID})
	if err != nil {
		errorData(w, err, 500)
		return
	}

	user, err := s.db.User(r.Context(), args.UID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	for i := range subs {
		if !util.IsSubmissionVisible(subs[i], user) {
			subs[i].Code = ""
		}
	}

	returnData(w, subs)
}

func (s *API) getSelfSubmissionsForProblem(w http.ResponseWriter, r *http.Request) {
	pid, ok := getFormInt(w, r, "pid")
	if !ok {
		return
	}
	uid := util.User(r).ID
	subs, err := s.db.UserProblemSubmissions(r.Context(), db.UserProblemSubmissionsParams{UserID: uid, ProblemID: pid})
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
		ID      int64
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	sub, err := s.db.Submission(r.Context(), args.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errorData(w, "Submission not found", http.StatusNotFound)
			return
		}
		log.Println(err)
		errorData(w, err, http.StatusNotFound)
		return
	}

	if !util.IsSubmissionEditor(sub, util.User(r)) {
		errorData(w, "You are not allowed to do this", 403)
		return
	}

	if err := s.db.SetSubmissionVisibility(r.Context(), db.SetSubmissionVisibilityParams{ID: args.ID, Visible: args.Visible}); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated visibility status")
}

// submissionSend registers a submission to be sent to the Eval handler
// Required values:
//	- code=[sourcecode] - source code of the submission, mutually exclusive with file uploads
//  - file=[file] - multipart file, mutually exclusive with the code param
//  - lang=[language] - language key like in languages.Languages
//  - problemID=[problem] - problem ID that the submission will be associated with
// Note that the `code` param is prioritized over file upload
func (s *API) submissionSend(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Code      string
		Lang      string
		ProblemID int64
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	var user = util.User(r)

	problem, err := s.db.Problem(r.Context(), args.ProblemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		return
	}

	if _, ok := languages.Languages[args.Lang]; ok == false {
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

		if problem.SourceSize != 0 && header.Size > int64(problem.SourceSize) {
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

	// add the submission to the DB
	id, err := s.db.CreateSubmission(r.Context(), db.CreateSubmissionParams{UserID: user.ID, ProblemID: problem.ID, Code: args.Code, Language: args.Lang})
	if err != nil {
		fmt.Println(err)
		errorData(w, "Couldn't create test", 500)
		return
	}

	// create the subtests
	tests, err := s.db.ProblemTests(r.Context(), problem.ID)
	for _, test := range tests {
		if err := s.db.CreateSubTest(r.Context(), db.CreateSubTestParams{UserID: user.ID, TestID: test.ID, SubmissionID: id}); err != nil {
			log.Println(err)
		}
	}

	statusData(w, "success", id, http.StatusCreated)
}
