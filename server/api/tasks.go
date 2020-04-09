package api

import (
	"encoding/json"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// Register
func (s *API) RegisterTaskRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.GetTasks) // ?start=0&count=25

	r.Post("/submit", s.SubmitTask)
	return r
}

// GetTasks returns all Tasks in a paginated order
func (s *API) GetTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var tasks []models.Task
	s.db.Find(&tasks)
	json.NewEncoder(w).Encode(tasks)
}

// SubmitTask registers a task to be sent to the Eval handler
func (s *API) SubmitTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	spew.Dump(r.Form)
}
