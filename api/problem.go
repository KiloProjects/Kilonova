package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

func (s *API) registerProblem() chi.Router {
	r := chi.NewRouter()
	r.Post("/create", s.createProblem)
	r.Get("/getAll", s.getAllProblems)
	r.Get("/getByID/{id}", s.getProblemByID)
	return r
}

func (s *API) createProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(64 * 1024 * 1024)
	spew.Dump(r.MultipartForm)
	tests, err := HandleTests(r)
	if err != nil {
		fmt.Fprintln(w, "ERROR:", err)
		return
	}
	problem := models.Problem{
		Name:  r.MultipartForm.Value["name"][0],
		Text:  r.MultipartForm.Value["description"][0],
		Tests: tests,
	}
	s.db.Create(&problem)
	w.Header().Set("location", "http://localhost:3000/")
	w.WriteHeader(http.StatusMovedPermanently)
}

func (s *API) getAllProblems(w http.ResponseWriter, r *http.Request) {
	var problems []models.Problem
	s.db.Preload("Tests").Find(&problems)
	json.NewEncoder(w).Encode(problems)
}

func (s *API) getProblemByID(w http.ResponseWriter, r *http.Request) {
	var problem models.Problem
	idstr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		fmt.Fprintln(w, "Invalid ID")
	}
	s.db.Where("id = ?", id).Preload("Tests").First(&problem)
	json.NewEncoder(w).Encode(problem)
}
