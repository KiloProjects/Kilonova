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

func (s *API) fullSubmission(ctx context.Context, id int, lookingUser *kilonova.UserBrief, looking bool) (sub *sudoapi.FullSubmission, err error) {
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
		var args struct {
			SubID int `json:"id"`
		}
		if err := parseRequest(r, &args); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		sub, err := s.fullSubmission(r.Context(), args.SubID, util.UserBrief(r), true)
		if err != nil {
			statusError(w, err)
			return
		}

		returnData(w, sub)
	}
}

func (s *API) filterSubs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var args kilonova.SubmissionFilter
		if err := parseRequest(r, &args); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		subs, err := s.base.Submissions(r.Context(), args, true, util.UserBrief(r))
		if err != nil {
			statusError(w, err)
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
		statusError(w, err)
		return
	}

	problem, err := s.base.Problem(r.Context(), args.ProblemID)
	if err != nil {
		statusError(w, err)
		return
	}

	if !s.base.IsProblemVisible(util.UserBrief(r), problem) {
		errorData(w, "Problem is not visible", 401)
		return
	}

	lang := s.base.Language(r.Context(), args.Lang)
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

	id, err := s.base.CreateSubmission(context.WithoutCancel(r.Context()), util.UserFull(r), problem, code, lang, args.ContestID, false)
	if err != nil {
		statusError(w, err)
		return
	}

	returnData(w, id)
}

type ExportedSubmission struct {
	*kilonova.Submission

	Code string `json:"code"`
}

func (s *API) exportSubmissions(ctx context.Context, args kilonova.SubmissionFilter) ([]*ExportedSubmission, error) {
	if args.Limit > 1000 {
		args.Limit = 1000
	}
	subs, err := s.base.RawSubmissions(ctx, args)
	if err != nil {
		return nil, err
	}
	exp := make([]*ExportedSubmission, len(subs))
	for i, sub := range subs {
		code, err := s.base.RawSubmissionCode(ctx, sub.ID)
		if err != nil {
			return nil, err
		}
		exp[i] = &ExportedSubmission{sub, string(code)}
	}
	return exp, nil
}
