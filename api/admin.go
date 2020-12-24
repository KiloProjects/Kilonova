package api

import (
	"log"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/util"
)

func (s *API) resetWaitingSubs(w http.ResponseWriter, r *http.Request) {
	err := s.db.ResetWaitingStatus(r.Context())
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Reset waiting subs")
}

func (s *API) getUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.Users(r.Context())
	if err != nil {
		log.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, users)
}

func (s *API) getAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := s.db.Admins(r.Context())
	if err != nil {
		log.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, admins)
}

func (s *API) getProposers(w http.ResponseWriter, r *http.Request) {
	proposers, err := s.db.Proposers(r.Context())
	if err != nil {
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, proposers)
}

func (s *API) setAdmin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID  int64
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

	user, err := s.db.User(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if err := user.SetAdmin(args.Set); err != nil {
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
		ID  int64
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

	user, err := s.db.User(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if err := user.SetProposer(args.Set); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added proposer")
	} else {
		returnData(w, "Succesfully removed proposer")
	}
}
