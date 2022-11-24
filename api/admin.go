package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
)

func (s *API) setAdmin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID  int
		Set bool
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.base.SetAdmin(r.Context(), args.ID, args.Set); err != nil {
		err.WriteError(w)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added admin")
		return
	}
	returnData(w, "Succesfully removed admin")
}

func (s *API) setProposer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID  int
		Set bool
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.base.SetProposer(r.Context(), args.ID, args.Set); err != nil {
		err.WriteError(w)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added proposer")
		return
	}
	returnData(w, "Succesfully removed proposer")
}

func (s *API) getAllUsers(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var query kilonova.UserFilter
	if err := decoder.Decode(&query, r.Form); err != nil {
		errorData(w, "Invalid request parameters", 400)
		return
	}
	rez, err := s.base.UsersBrief(r.Context(), query)
	if err != nil {
		err.WriteError(w)
		return
	}

	cnt, err := s.base.CountUsers(r.Context(), query)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		Users []*kilonova.UserBrief `json:"users"`

		Count int `json:"total_count"`
	}{Users: rez, Count: cnt})
}
