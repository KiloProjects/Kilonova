package api

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

func (s *API) registerMOTD() chi.Router {
	r := chi.NewRouter()
	r.Post("/new", s.newMOTD)
	r.Get("/get", s.getMOTD)
	r.Get("/getAll", s.getAllMOTDs)
	return r
}

func (s *API) newMOTD(w http.ResponseWriter, r *http.Request) {
	newMotd := r.PostFormValue("data")
	s.db.Create(&models.MOTD{Motd: newMotd})

	w.Header().Set("location", "http://localhost:3000/")
	w.WriteHeader(http.StatusMovedPermanently)
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
