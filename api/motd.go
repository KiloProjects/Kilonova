package api

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// /api/motd
func (s *API) registerMOTD() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.getMOTD)
	r.Get("/getAll", s.getAllMOTDs)
	return r
}

func (s *API) getAllMOTDs(w http.ResponseWriter, r *http.Request) {
	var motds []models.MOTD
	s.db.Find(&motds)
	json.NewEncoder(w).Encode(motds)
}

func (s *API) getMOTD(w http.ResponseWriter, r *http.Request) {
	var motds []models.MOTD
	s.db.Find(&motds)
	motd := motds[rand.Intn(len(motds))]
	json.NewEncoder(w).Encode(motd)
}
