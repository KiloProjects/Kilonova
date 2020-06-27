package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
)

// RegisterAdminRoutes registers all of the admin router at /api/admin
func (s *API) RegisterAdminRoutes() chi.Router {
	r := chi.NewRouter()
	// disabled for debugging
	// r.Use(s.mustBeAdmin)

	r.Get("/makeAdmin", s.MakeAdmin)
	r.Post("/newMOTD", s.NewMOTD)
	r.Get("/getUsers", func(w http.ResponseWriter, r *http.Request) {
		var users []common.User
		s.db.Find(&users)
		s.ReturnData(w, "success", users)
	})
	r.HandleFunc("/dropAll", func(w http.ResponseWriter, r *http.Request) {
		s.db.DropTable(common.User{}, common.EvalTest{}, common.MOTD{}, common.Problem{}, common.Task{}, common.Test{})
		fmt.Println("restarting")
		os.Exit(0)
	})
	return r
}

// MakeAdmin updates an account to make it an admin
func (s *API) MakeAdmin(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("id")
	if sid == "" {
		s.ErrorData(w, "Missing User ID", http.StatusBadRequest)
	}
	id, err := strconv.Atoi(sid)
	if err != nil {
		s.ErrorData(w, "ID is not int", http.StatusBadRequest)
	}
	if err := s.db.Model(&common.User{}).Where("id = ?", id).Update(gorm.ToColumnName("IsAdmin"), true).Error; err != nil {
		s.ErrorData(w, err.Error(), http.StatusInternalServerError)
	}
}

// NewMOTD adds a new MOTD to the DB
func (s *API) NewMOTD(w http.ResponseWriter, r *http.Request) {
	newMotd := r.FormValue("data")
	s.db.Create(&common.MOTD{Motd: newMotd})
}
