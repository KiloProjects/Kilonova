package server

import (
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
)

// RegisterAdminRoutes registers all of the admin router at /api/admin
func (s *API) RegisterAdminRoutes() chi.Router {
	r := chi.NewRouter()
	r.Use(s.MustBeAdmin)

	// /admin/makeAdmin
	r.Get("/makeAdmin", s.MakeAdmin)
	// /admin/getAllUsers
	r.Get("/getAllUsers", func(w http.ResponseWriter, r *http.Request) {
		users, err := s.db.GetAllUsers()
		if err != nil {
			s.errlog("/admin/getAllUsers: Error from DB: %s", err)
			s.ErrorData(w, "Could not read from DB", 500)
			return
		}
		s.ReturnData(w, "success", users)
	})
	// /admin/dropAll
	r.HandleFunc("/dropAll", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("IAmAbsolutelySureIWantToDoThis") == "" {
			s.ErrorData(w, "Sorry, you need a specific form value. Look into the source code if you're sure", http.StatusBadRequest)
			return
		}
		s.db.DB.Migrator().DropTable(&common.EvalTest{}, &common.Problem{}, &common.Task{}, &common.Test{}, &common.User{})
		s.ReturnData(w, "success", "I hope you're proud")
	})
	return r
}

// MakeAdmin updates an account to make it an admin
func (s *API) MakeAdmin(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("id")
	if sid == "" {
		s.ErrorData(w, "Missing User ID", http.StatusBadRequest)
	}
	id, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		s.ErrorData(w, "ID is not int", http.StatusBadRequest)
		return
	}
	if err := s.db.MakeAdmin(uint(id)); err != nil {
		s.ErrorData(w, err.Error(), 500)
		return
	}
}
