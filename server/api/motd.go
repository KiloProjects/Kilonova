package api

import (
	"math/rand"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// RegisterMOTDRoutes registers the MOTD routes at /api/motd
func (s *API) RegisterMOTDRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.GetMOTD)
	r.Get("/getAll", s.GetAllMOTDs)
	return r
}

// GetAllMOTDs returns all MOTDs in the database
// TODO: Pagination
func (s *API) GetAllMOTDs(w http.ResponseWriter, r *http.Request) {
	var motds []models.MOTD
	s.db.Find(&motds)
	s.ReturnData(w, "success", motds)
}

// GetMOTD returns a random MOTD from the database
func (s *API) GetMOTD(w http.ResponseWriter, r *http.Request) {
	var motds []models.MOTD
	s.db.Find(&motds)
	motd := motds[rand.Intn(len(motds))]
	s.ReturnData(w, "success", motd)
}
