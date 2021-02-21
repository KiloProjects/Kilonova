package api

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
)

type subTestLine struct {
	SubTest *kilonova.SubTest `json:"subtest"`
	Test    *kilonova.Test    `json:"pb_test"`
}

func (s *API) fetchSubTests(ctx context.Context, sub *kilonova.Submission) ([]subTestLine, error) {
	stests, err := s.stserv.SubTestsBySubID(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	var lines = make([]subTestLine, 0, len(stests))
	for _, stest := range stests {
		stest := stest
		test, err := s.tserv.TestByID(ctx, stest.TestID)
		if err != nil {
			return nil, err
		}
		lines = append(lines, subTestLine{SubTest: stest, Test: test})
	}
	return lines, nil
}

// getSubmissionByID returns a submission based on an ID
func (s *API) getSubmissionByID() func(w http.ResponseWriter, r *http.Request) {
	type line struct {
		SubEditor     bool                 `json:"sub_editor"`
		ProblemEditor bool                 `json:"problem_editor"`
		Sub           *kilonova.Submission `json:"sub"`
		User          *kilonova.User       `json:"author,omitempty"`
		Problem       *kilonova.Problem    `json:"problem,omitempty"`
		SubTests      []subTestLine        `json:"subtests,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		subID, ok := getFormInt(w, r, "id")
		if !ok {
			return
		}

		sub, err := s.sserv.SubmissionByID(r.Context(), subID)
		if err != nil {
			errorData(w, "Could not find submission", http.StatusBadRequest)
			return
		}

		pb, err := s.pserv.ProblemByID(r.Context(), sub.ProblemID)
		if err != nil {
			log.Println("Couldn't get some stuff:", err)
			errorData(w, err, 500)
			return
		}

		l := line{SubEditor: util.IsSubmissionEditor(sub, util.User(r)), ProblemEditor: util.IsProblemEditor(util.User(r), pb), Sub: sub}

		st, err := s.fetchSubTests(r.Context(), sub)
		if err != nil {
			log.Println("Couldn't get some stuff:", err)
			errorData(w, err, 500)
			return
		}
		l.SubTests = st

		if r.FormValue("expanded") != "" {
			user, err := s.userv.UserByID(r.Context(), sub.UserID)
			if err != nil {
				log.Println("Couldn't get some stuff:", err)
				errorData(w, err, 500)
				return
			}
			l.User = user
			l.Problem = pb
		}

		s.kn.FilterCode(sub, util.User(r), s.sserv)

		returnData(w, l)
	}
}

func (s *API) filterSubs() http.HandlerFunc {
	type line struct {
		Sub     *kilonova.Submission `json:"sub"`
		User    *kilonova.User       `json:"author,omitempty"`
		Problem *kilonova.Problem    `json:"problem,omitempty"`
	}
	type toRet struct {
		TotalCount int    `json:"count"`
		Subs       []line `json:"subs"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct {
			LoadUser    bool `schema:"loadUser"`
			LoadProblem bool `schema:"loadProblem"`
			kilonova.SubmissionFilter
		}
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		if args.Limit == 0 || args.Limit > 50 {
			args.Limit = 50
		}

		count, err := s.sserv.CountSubmissions(r.Context(), args.SubmissionFilter)
		if err != nil {
			log.Println(err)
			errorData(w, http.StatusText(500), 500)
			return
		}

		subs, err := s.sserv.Submissions(r.Context(), args.SubmissionFilter)
		if err != nil {
			log.Println(err)
			errorData(w, http.StatusText(500), 500)
			return
		}

		ret := []line{}

		user := util.User(r)
		for _, sub := range subs {
			s.kn.FilterCode(sub, user, s.sserv)
			l := line{Sub: sub}

			if args.LoadUser {
				user, err := s.userv.UserByID(r.Context(), sub.UserID)
				if err != nil {
					log.Println("Couldn't get some stuff:", err)
					errorData(w, err, 500)
					return
				}
				l.User = user
			}
			if args.LoadProblem {
				pb, err := s.pserv.ProblemByID(r.Context(), sub.ProblemID)
				if err != nil {
					log.Println("Couldn't get some stuff:", err)
					errorData(w, err, 500)
					return
				}
				l.Problem = pb
			}

			ret = append(ret, l)
		}

		returnData(w, toRet{Subs: ret, TotalCount: count})
	}
}

func (s *API) setSubmissionVisible(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Visible bool
		ID      int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	sub, err := s.sserv.SubmissionByID(r.Context(), args.ID)
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

	if err := s.sserv.UpdateSubmission(r.Context(), sub.ID, kilonova.SubmissionUpdate{Visible: &args.Visible}); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated visibility status")
}

// submissionSend registers a submission to be sent to the Eval handler
// Required values:
//	- code=[sourcecode] - source code of the submission, mutually exclusive with file uploads
//  - file=[file] - multipart file, mutually exclusive with the code param
//  - lang=[language] - language key like in config.C.Languages
//  - problemID=[problem] - problem ID that the submission will be associated with
// Note that the `code` param is prioritized over file upload
func (s *API) submissionSend(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Code      string
		Lang      string
		ProblemID int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	var user = util.User(r)

	problem, err := s.pserv.ProblemByID(r.Context(), args.ProblemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		return
	}

	if _, ok := config.Languages[args.Lang]; ok == false {
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
	sub, err := s.addSubmission(r.Context(), user.ID, problem.ID, args.Code, args.Lang, user.DefaultVisible)
	if err != nil {
		log.Println(err)
		errorData(w, err, 500)
		return
	}

	statusData(w, "success", sub.ID, http.StatusCreated)
}

// addSubmission adds the submission to the DB, but also creates the subtests
// was split away from the function above because it got too big
func (s *API) addSubmission(ctx context.Context, userID int, problemID int, code string, lang string, visible bool) (*kilonova.Submission, error) {
	tests, err := s.tserv.Tests(ctx, problemID)
	if err != nil {
		return nil, err
	}

	// Add submission
	var sub kilonova.Submission
	sub.UserID = userID
	sub.ProblemID = problemID
	sub.Code = code
	sub.Language = lang
	sub.Visible = visible
	if err := s.sserv.CreateSubmission(ctx, &sub); err != nil {
		return nil, err
	}

	// Add subtests
	for _, test := range tests {
		if err := s.stserv.CreateSubTest(ctx, &kilonova.SubTest{UserID: userID, TestID: test.ID, SubmissionID: sub.ID}); err != nil {
			return nil, err
		}
	}

	return &sub, nil
}
