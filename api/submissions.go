package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
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
	type line struct {
		Sub     *kilonova.Submission `json:"sub"`
		User    *kilonova.UserBrief  `json:"author,omitempty"`
		Problem *kilonova.Problem    `json:"problem,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args kilonova.SubmissionFilter
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		subs, err := s.base.Submissions(r.Context(), args, util.UserBrief(r))
		if err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, subs)

	}
}
