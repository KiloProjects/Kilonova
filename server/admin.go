package server

import (
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
)

func getIDFromForm(w http.ResponseWriter, r *http.Request) (uint, bool) {
	sid := r.FormValue("id")
	if sid == "" {
		errorData(w, "Missing User ID", http.StatusBadRequest)
		return 0, false
	}
	id, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		errorData(w, "ID is not int", http.StatusBadRequest)
		return 0, false
	}
	return uint(id), true
}

func (s *API) getUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetAllUsers()
	if err != nil {
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, "success", users)
}

func (s *API) makeAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromForm(w, r)
	if ok {
		if err := s.db.MakeAdmin(id); err != nil {
			errorData(w, err.Error(), 500)
		}
		returnData(w, "success", "Succesfully added admin")
	}
}

func (s *API) stripAdmin(w http.ResponseWriter, r *http.Request) {
	if common.UserFromContext(r).ID != 1 {
		errorData(w, "Only the root user can strip admin", http.StatusUnauthorized)
		return
	}

	id, ok := getIDFromForm(w, r)
	if ok {
		if id == 1 {
			errorData(w, "You can't remove the admin of the root user", http.StatusNotAcceptable)
			return
		}

		if err := s.db.RemoveAdmin(id); err != nil {
			errorData(w, err.Error(), 500)
		}
		returnData(w, "success", "Succesfully removed admin")
	}
}

func (s *API) stripProposer(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromForm(w, r)
	if ok {
		if id == 1 {
			errorData(w, "You can't remove the proposer rank of the root user", http.StatusNotAcceptable)
			return
		}
		if id == common.UserFromContext(r).ID {
			errorData(w, "You can't remove your own proposer rank", http.StatusNotAcceptable)
			return
		}
		if err := s.db.RemoveProposer(id); err != nil {
			errorData(w, err.Error(), 500)
		}
		returnData(w, "success", "Succesfully stripped proposer role")
	}
}

func (s *API) makeProposer(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromForm(w, r)
	if ok {
		if err := s.db.MakeProposer(id); err != nil {
			errorData(w, err.Error(), 500)
		}
		returnData(w, "success", "Succesfully made proposer")
	}
}

func (s *API) dropAll(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("IAmAbsolutelySureIWantToDoThis") == "" {
		errorData(w, "Sorry, you need a specific form value. Look into the source code if you're sure", http.StatusBadRequest)
		return
	}
	s.db.DB.Migrator().DropTable(&common.EvalTest{}, &common.Problem{}, &common.Task{}, &common.Test{}, &common.User{})
	returnData(w, "success", "I hope you're proud")
}
