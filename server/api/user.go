package api

import (
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// RegisterUserRoutes mounts the user-related routes at /api/user
func (s *API) RegisterUserRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/getByName", s.GetUserByName)
	r.With(s.MustBeAuthed).Get("/getSelf", s.GetSelf)
	return r
}

// GetUserByName returns a user based on a specified name
func (s *API) GetUserByName(w http.ResponseWriter, r *http.Request) {
	var user models.User
	name := r.FormValue("name")
	if name == "" {
		s.ErrorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	s.db.First(&user, "lower(name) = lower(?)", name)
	if user.ID == 0 {
		s.ErrorData(w, "User Not Found", http.StatusNotFound)
		return
	}
	s.ReturnData(w, "success", user)
}

// GetSelf returns the authenticated user
func (s *API) GetSelf(w http.ResponseWriter, r *http.Request) {
	s.ReturnData(w, "success", r.Context().Value(models.KNContextType("user")).(models.User))
}
