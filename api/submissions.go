package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
)

func (s *API) fullSubmission(ctx context.Context, id int, lookingUser *kilonova.UserBrief, looking bool) (*sudoapi.FullSubmission, *kilonova.StatusError) {
	var sub *sudoapi.FullSubmission
	var err *kilonova.StatusError
	if looking {
		sub, err = s.base.Submission(ctx, id, lookingUser)
	} else {
		sub, err = s.base.FullSubmission(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	return sub, nil
}

// getSubmissionByID returns a submission based on an ID
func (s *API) getSubmissionByID() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct {
			SubID int `json:"id"`
		}
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		sub, err := s.fullSubmission(r.Context(), args.SubID, util.UserBrief(r), true)
		if err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, sub)
	}
}

func (s *API) filterSubs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args kilonova.SubmissionFilter
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		subs, err := s.base.Submissions(r.Context(), args, true, util.UserBrief(r))
		if err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, subs)
	}
}

func (s *API) createSubmission(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1 * 1024 * 1024) // 1MB
	defer cleanupMultipart(r)
	var args struct {
		Lang      string `json:"language"`
		ProblemID int    `json:"problem_id"`
		ContestID *int   `json:"contest_id"`
	}
	if err := parseRequest(r, &args); err != nil {
		err.WriteError(w)
		return
	}

	problem, err1 := s.base.Problem(r.Context(), args.ProblemID)
	if err1 != nil {
		err1.WriteError(w)
		return
	}

	if !s.base.IsProblemVisible(util.UserBrief(r), problem) {
		errorData(w, "Problem is not visible", 401)
		return
	}

	lang := s.base.Language(args.Lang)
	if lang == nil {
		errorData(w, "Invalid language", 400)
		return
	}

	f, _, err := r.FormFile("code")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			errorData(w, "Missing `code` file with source code", 400)
			return
		}
		zap.S().Warn(err)
		errorData(w, "Could not open multipart file", 500)
		return
	}

	code, err := io.ReadAll(f)
	if err != nil {
		zap.S().Warn(err)
		errorData(w, "Could not read source code", 500)
		return
	}

	id, err1 := s.base.CreateSubmission(context.WithoutCancel(r.Context()), util.UserFull(r), problem, code, lang, args.ContestID, false)
	if err1 != nil {
		err1.WriteError(w)
		return
	}

	returnData(w, id)
}
