package api

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
)

type subTestLine struct {
	SubTest *kilonova.SubTest `json:"subtest"`
	Test    *kilonova.Test    `json:"pb_test"`
}

func (s *API) fetchSubTests(ctx context.Context, sub *kilonova.Submission) ([]subTestLine, error) {
	stests, err := s.db.SubTestsBySubID(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	var lines = make([]subTestLine, 0, len(stests))
	for _, stest := range stests {
		stest := stest
		test, err := s.db.TestByID(ctx, stest.TestID)
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
		SubTasks      []*kilonova.SubTask  `json:"subtasks,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		subID, ok := getFormInt(w, r, "id")
		if !ok {
			return
		}

		sub, err := s.db.Submission(r.Context(), subID)
		if err != nil {
			errorData(w, "Could not find submission", http.StatusBadRequest)
			return
		}

		pb, err := s.db.Problem(r.Context(), sub.ProblemID)
		if err != nil {
			log.Println("Couldn't get some stuff:", err)
			errorData(w, err, 500)
			return
		}

		if !util.IsProblemVisible(util.User(r), pb) {
			errorData(w, "You can't access this submission because the problem isn't visible!", 403)
			return
		}

		stks, err := s.db.SubTasks(r.Context(), pb.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			errorData(w, err, 500)
			return
		}

		l := line{SubEditor: util.IsSubmissionEditor(sub, util.User(r)), ProblemEditor: util.IsProblemEditor(util.User(r), pb), Sub: sub, SubTasks: stks}

		st, err := s.fetchSubTests(r.Context(), sub)
		if err != nil {
			log.Println("Couldn't get some stuff:", err)
			errorData(w, err, 500)
			return
		}
		l.SubTests = st

		if r.FormValue("expanded") != "" {
			user, err := s.db.User(r.Context(), sub.UserID)
			if err != nil {
				log.Println("Couldn't get some stuff:", err)
				errorData(w, err, 500)
				return
			}
			l.User = user
			l.Problem = pb
		}

		s.kn.FilterCode(sub, util.User(r), s.db)

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
		var args kilonova.SubmissionFilter
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		userCache := make(map[int]*kilonova.User)
		problemCache := make(map[int]*kilonova.Problem)

		if args.Limit == 0 || args.Limit > 50 {
			args.Limit = 50
		}

		if args.Ordering == "" {
			args.Ordering = "id"
		}

		if !(args.Ordering == "id" || args.Ordering == "max_time" || args.Ordering == "max_mem" || args.Ordering == "score") {
			errorData(w, "Invalid ordering type", 400)
			return
		}

		count, err := s.db.CountSubmissions(r.Context(), args)
		if err != nil {
			log.Println(err)
			errorData(w, http.StatusText(500), 500)
			return
		}

		subs, err := s.db.Submissions(r.Context(), args)
		if err != nil {
			log.Println(err)
			errorData(w, http.StatusText(500), 500)
			return
		}

		ret := []line{}

		user := util.User(r)
		for _, sub := range subs {
			s.kn.FilterCode(sub, user, s.db)
			l := line{Sub: sub}

			subProblem, ok := problemCache[sub.ProblemID]
			if ok {
				l.Problem = subProblem
			} else {
				pb, err := s.db.Problem(r.Context(), sub.ProblemID)
				if err != nil {
					log.Println("Couldn't get some stuff:", err)
					errorData(w, err, 500)
					return
				}
				l.Problem = pb
				problemCache[sub.ProblemID] = pb
			}

			if !util.IsProblemVisible(util.User(r), l.Problem) {
				continue
			}

			subUser, ok := userCache[sub.UserID]
			if ok {
				l.User = subUser
			} else {
				user, err := s.db.User(r.Context(), sub.UserID)
				if err != nil {
					log.Println("Couldn't get some stuff:", err)
					errorData(w, err, 500)
					return
				}
				l.User = user
				userCache[sub.UserID] = user
			}

			ret = append(ret, l)
		}

		returnData(w, toRet{Subs: ret, TotalCount: count})
	}
}

func (s *API) setSubmissionQuality(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Quality bool
		ID      int
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

	upd := kilonova.SubmissionUpdate{
		Quality: &args.Quality,
	}
	if args.Quality {
		t := true
		upd.Visible = &t
	}

	if err := s.db.UpdateSubmission(r.Context(), sub.ID, upd); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated quality status")
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

	if err := s.db.UpdateSubmission(r.Context(), sub.ID, kilonova.SubmissionUpdate{Visible: &args.Visible}); err != nil {
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

	problem, err := s.db.Problem(r.Context(), args.ProblemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		return
	}

	if _, ok := eval.Langs[args.Lang]; ok == false {
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
	tests, err := s.db.Tests(ctx, problemID)
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
	if err := s.db.CreateSubmission(ctx, &sub); err != nil {
		return nil, err
	}

	// Add subtests
	for _, test := range tests {
		if err := s.db.CreateSubTest(ctx, &kilonova.SubTest{UserID: userID, TestID: test.ID, SubmissionID: sub.ID}); err != nil {
			return nil, err
		}
	}

	if err := s.db.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting}); err != nil {
		return nil, err
	}

	return &sub, nil
}

func (s *API) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteSubmission(r.Context(), args.ID); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Deleted submission")
}
