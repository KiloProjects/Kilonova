package server

import (
	"log"
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
		log.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, "success", users)
}

func (s *API) getAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := s.db.GetAllAdmins()
	if err != nil {
		log.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, "success", admins)
}

func (s *API) getProposers(w http.ResponseWriter, r *http.Request) {
	proposers, err := s.db.GetAllProposers()
	if err != nil {
		log.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, "success", proposers)
}

func (s *API) setAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromForm(w, r)
	if ok {
		sset := r.FormValue("set")
		if sset == "" {
			errorData(w, "Missing `set` param", http.StatusBadRequest)
			return
		}
		set, err := strconv.ParseBool(sset)
		if err != nil {
			errorData(w, "`set` not valid bool", http.StatusBadRequest)
			return
		}

		if set == false { // additional checks for removing
			if id == 1 {
				errorData(w, "You can't remove admin of the root user", http.StatusNotAcceptable)
				return
			}
			if id == common.UserFromContext(r).ID {
				errorData(w, "You can't remove your own admin", http.StatusNotAcceptable)
				return
			}
		}

		if err := s.db.SetAdmin(id, set); err != nil {
			errorData(w, err.Error(), 500)
		}
		if set {
			returnData(w, "success", "Succesfully added admin")
		} else {
			returnData(w, "success", "Succesfully removed admin")
		}
	}
}

func (s *API) setProposer(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromForm(w, r)
	if ok {
		sset := r.FormValue("set")
		if sset == "" {
			errorData(w, "Missing `set` param", http.StatusBadRequest)
			return
		}
		set, err := strconv.ParseBool(sset)
		if err != nil {
			errorData(w, "`set` not valid bool", http.StatusBadRequest)
			return
		}

		if set == false { // additional checks for removing
			if id == 1 {
				errorData(w, "You can't remove the proposer rank of the root user", http.StatusNotAcceptable)
				return
			}
			if id == common.UserFromContext(r).ID {
				errorData(w, "You can't remove your own proposer rank", http.StatusNotAcceptable)
				return
			}
		}

		if err := s.db.SetProposer(id, set); err != nil {
			errorData(w, err.Error(), 500)
		}
		if set {
			returnData(w, "success", "Succesfully added proposer")
		} else {
			returnData(w, "success", "Succesfully removed proposer")
		}
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
