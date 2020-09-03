package server

import (
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/util"
)

func getIDFromForm(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id, bl := getFormInt(w, r, "id")
	return uint(id), bl
}

func (s *API) getUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetAllUsers()
	if err != nil {
		s.logger.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, users)
}

func (s *API) getAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := s.db.GetAllAdmins()
	if err != nil {
		s.logger.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, admins)
}

func (s *API) getProposers(w http.ResponseWriter, r *http.Request) {
	proposers, err := s.db.GetAllProposers()
	if err != nil {
		s.logger.Println(err)
		errorData(w, "Could not read from DB", 500)
		return
	}
	returnData(w, proposers)
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
			if id == util.UserFromContext(r).ID {
				errorData(w, "You can't remove your own admin", http.StatusNotAcceptable)
				return
			}
		}

		if err := s.db.SetAdmin(id, set); err != nil {
			errorData(w, err, 500)
		}
		if set {
			returnData(w, "Succesfully added admin")
		} else {
			returnData(w, "Succesfully removed admin")
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
			if id == util.UserFromContext(r).ID {
				errorData(w, "You can't remove your own proposer rank", http.StatusNotAcceptable)
				return
			}
		}

		if err := s.db.SetProposer(id, set); err != nil {
			errorData(w, err, 500)
		}
		if set {
			returnData(w, "Succesfully added proposer")
		} else {
			returnData(w, "Succesfully removed proposer")
		}
	}
}

func (s *API) dropAll(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("IAmAbsolutelySureIWantToDoThis") == "" {
		errorData(w, "Sorry, you need a specific form value. Look into the source code if you're sure", http.StatusBadRequest)
		return
	}
	s.db.DB.Migrator().DropTable(&models.EvalTest{}, &models.Problem{}, &models.Task{}, &models.Test{}, &models.User{})
	returnData(w, "I hope you're proud")
}
