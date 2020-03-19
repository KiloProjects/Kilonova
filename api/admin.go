package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// /api/admin
func (s *API) registerAdmin() chi.Router {
	r := chi.NewRouter()
	// r.Use(s.mustBeAdmin)

	r.Post("/newMOTD", s.newMOTD)
	r.Post("/createProblem", s.createProblem)
	r.Get("/getUsers", func(w http.ResponseWriter, r *http.Request) {
		var users []models.User
		s.db.Find(&users)
		json.NewEncoder(w).Encode(users)
	})
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

func (s *API) newMOTD(w http.ResponseWriter, r *http.Request) {
	newMotd := r.PostFormValue("data")
	s.db.Create(&models.MOTD{Motd: newMotd})

	w.Header().Set("location", "http://localhost:3000/")
	w.WriteHeader(http.StatusMovedPermanently)
}

// HandleTests fills a models.Test array with the tests from an archive
func HandleTests(r *http.Request) ([]models.Test, error) {
	return nil, nil
}
