package api

import (
	"net/http"
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
