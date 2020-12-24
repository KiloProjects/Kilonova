package api

import (
	"database/sql"
	"errors"
	"io"
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

	if r.FormValue("expanded") != "" {
		if err := sub.LoadAll(); err != nil {
			log.Println("Couldn't get some stuff:", err)
			errorData(w, err, 500)
			return
		}
	}

	s.kn.FilterCode(sub, util.User(r))

	returnData(w, struct {
		SubEditor  bool           `json:"sub_editor"`
		Submission *db.Submission `json:"sub"`
	}{
		SubEditor:  util.IsSubmissionEditor(sub, util.User(r)),
		Submission: sub,
	})
}

func (s *API) filterSubs(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		LoadUser    bool
		LoadProblem bool
		db.SubmissionFilter
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	subs, err := s.db.FilterSubmissions(r.Context(), args.SubmissionFilter)
	if err != nil {
		log.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}

	user := util.User(r)
	for i := range subs {
		s.kn.FilterCode(subs[i], user)
		if args.LoadUser {
			if _, err := subs[i].GetUser(); err != nil {
				log.Println("Couldn't get some stuff:", err)
				errorData(w, err, 500)
				return
			}
		}
		if args.LoadProblem {
			if _, err := subs[i].GetProblem(); err != nil {
				log.Println("Couldn't get some stuff:", err)
				errorData(w, err, 500)
				return
			}
		}
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

	if err := sub.SetVisibility(args.Visible); err != nil {
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
		c, err := io.ReadAll(file)
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

	// add the submission along with subtests to the DB
	id, err := s.db.AddSubmission(r.Context(), user.ID, problem.ID, args.Code, args.Lang, user.DefaultVisible)

	statusData(w, "success", id, http.StatusCreated)
}
