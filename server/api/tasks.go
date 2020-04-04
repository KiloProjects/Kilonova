package api

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

func (s *API) registerTasks() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.getTasks) // ?start=0&count=25

	r.Post("/submit", s.submitTask)
	return r
}

// TODO
func (s *API) getTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
}

// TODO
func (s *API) submitTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	spew.Dump(r.Form)
	w.Header().Set("location", "http://localhost:3000/")
	w.WriteHeader(http.StatusMovedPermanently)
}
