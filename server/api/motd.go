package api

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// /api/motd
func (s *API) RegisterMOTDRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.GetMOTD)
	r.Get("/getAll", s.GetAllMOTDs)
	return r
}

func (s *API) GetAllMOTDs(w http.ResponseWriter, r *http.Request) {
	var motds []models.MOTD
	s.db.Find(&motds)
	json.NewEncoder(w).Encode(motds)
}

func (s *API) GetMOTD(w http.ResponseWriter, r *http.Request) {
	var motds []models.MOTD
	s.db.Find(&motds)
	motd := motds[rand.Intn(len(motds))]
	json.NewEncoder(w).Encode(motd)
}
