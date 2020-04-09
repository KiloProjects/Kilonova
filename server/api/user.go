package api

import (
	"encoding/json"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// /api/user
func (s *API) RegisterUserRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/getByName", s.GetUserByName)
	r.With(s.MustBeAuthed).Get("/getSelf", s.GetSelf)
	return r
}

func (s *API) GetUserByName(w http.ResponseWriter, r *http.Request) {
	var user models.User
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name not specified", http.StatusBadRequest)
		return
	}
	s.db.First(&user, "lower(name) = lower(?)", name)
	if user.ID == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
	}
	err := json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// GetSelf returns the
func (s *API) GetSelf(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(r.Context().Value(models.KNContextType("user")).(models.User))
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
