package api

import (
	"encoding/json"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// /api/user
func (s *API) registerUser() chi.Router {
	r := chi.NewRouter()
	r.Get("/getByName", func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "Name not specified", http.StatusBadRequest)
		}
		s.db.First(&user, "name = ?", name)
		json.NewEncoder(w).Encode(user)
	})
	r.With(s.mustBeAuthed).Get("/getSelf", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(r.Context().Value(models.KNContextType("user")).(models.User))
	})
	return r
}
