package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) createContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		SubmissionID int `json:"id"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	panic("TODO")
}

func (s *API) getContest(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.Contest(r))
}
