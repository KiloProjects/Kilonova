package api

import (
	"log"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) resetWaitingSubs(w http.ResponseWriter, r *http.Request) {
	err := s.sserv.BulkUpdateSubmissions(r.Context(), kilonova.SubmissionFilter{Status: kilonova.StatusWorking}, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting})
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Reset waiting subs")
}

func (s *API) getUsers(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.UserFilter
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	users, err := s.userv.Users(r.Context(), args)
	if err != nil {
		log.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, users)
}

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

	if args.ID <= 0 {
		errorData(w, "Invalid ID", http.StatusBadRequest)
	}

	if args.Set == false { // additional checks for removing
		if args.ID == 1 {
			errorData(w, "You can't remove admin of the root user", http.StatusNotAcceptable)
			return
		}
		if args.ID == util.User(r).ID {
			errorData(w, "You can't remove your own admin", http.StatusNotAcceptable)
			return
		}
	}

	if err := s.userv.UpdateUser(r.Context(), args.ID, kilonova.UserUpdate{Admin: &args.Set}); err != nil {
		errorData(w, err, 500)
		return
	}
	if args.Set {
		returnData(w, "Succesfully added admin")
	} else {
		returnData(w, "Succesfully removed admin")
	}
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

	if args.ID == 0 {
		errorData(w, "Empty ID", http.StatusBadRequest)
		return
	}

	if args.Set == false { // additional checks for removing
		if args.ID == 1 {
			errorData(w, "You can't remove the proposer rank of the root user", http.StatusNotAcceptable)
			return
		}
		if args.ID == util.User(r).ID {
			errorData(w, "You can't remove your own proposer rank", http.StatusNotAcceptable)
			return
		}
	}

	if err := s.userv.UpdateUser(r.Context(), args.ID, kilonova.UserUpdate{Proposer: &args.Set}); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added proposer")
	} else {
		returnData(w, "Succesfully removed proposer")
	}
}
