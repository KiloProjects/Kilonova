package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
)

// getSubmissionByID returns a submission based on an ID
func (s *API) getSubmissionByID() func(w http.ResponseWriter, r *http.Request) {
	type line struct {
		ProblemEditor bool                 `json:"problem_editor"`
		Sub           *kilonova.Submission `json:"sub"`
		User          *kilonova.UserBrief  `json:"author,omitempty"`
		Problem       *kilonova.Problem    `json:"problem,omitempty"`
		SubTests      []*sudoapi.SubTest   `json:"subtests,omitempty"`
		SubTasks      []*kilonova.SubTask  `json:"subtasks,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct {
			SubID int `json:"id"`
		}
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, http.StatusBadRequest)
			return
		}

		sub, err := s.base.Submission(r.Context(), args.SubID, util.UserBrief(r))
		if err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, &line{
			ProblemEditor: sub.ProblemEditor,
			Sub:           &sub.Submission,
			User:          sub.Author,
			Problem:       sub.Problem,
			SubTests:      sub.SubTests,
			SubTasks:      sub.SubTasks,
		})
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

		ret := []line{}

		for _, sub := range subs.Submissions {
			ret = append(ret, line{
				Sub:     sub,
				User:    subs.Users[sub.UserID],
				Problem: subs.Problems[sub.ProblemID],
			})
		}

		returnData(w, struct {
			TotalCount int    `json:"count"`
			Subs       []line `json:"subs"`
		}{Subs: ret, TotalCount: subs.Count})
	}
}
