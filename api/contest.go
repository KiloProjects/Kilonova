package api

import (
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) createContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name string `json:"name"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	id, err := s.base.CreateContest(r.Context(), args.Name, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, id)
}

func (s *API) updateContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name *string `json:"name"`

		PublicJoin *bool `json:"public_join"`
		Visible    *bool `json:"visible"`

		StartTime *string `json:"start_time"`
		EndTime   *string `json:"end_time"`

		MaxSubs *int `json:"max_subs"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	var startTime, endTime *time.Time
	if args.StartTime != nil {
		t, err := time.Parse(time.RFC1123Z, *args.StartTime)
		if err != nil {
			errorData(w, "Invalid timestamp", 400)
			return
		}
		startTime = &t
	}
	if args.EndTime != nil {
		t, err := time.Parse(time.RFC1123Z, *args.EndTime)
		if err != nil {
			errorData(w, "Invalid timestamp", 400)
			return
		}
		endTime = &t
	}

	if err := s.base.UpdateContest(r.Context(), util.Contest(r).ID, kilonova.ContestUpdate{
		Name:       args.Name,
		PublicJoin: args.PublicJoin,
		Visible:    args.Visible,
		StartTime:  startTime,
		EndTime:    endTime,
		MaxSubs:    args.MaxSubs,
	}); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Contest updated")
}

func (s *API) updateContestProblems(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		List []int `json:"list"`
	}
	if err := parseJsonBody(r, &args); err != nil {
		err.WriteError(w)
		return
	}

	if args.List == nil {
		errorData(w, "You must specify a list of problems", 400)
	}

	list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.UpdateContestProblems(r.Context(), util.Contest(r).ID, list); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Updated contest problems")
}

func (s *API) getContest(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.Contest(r))
}
