package api

import (
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// RegisterTaskRoutes mounts the Task routes at /api/tasks
func (s *API) RegisterTaskRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.GetTasks)

	r.Post("/submit", s.SubmitTask)
	return r
}

// GetTasks returns all Tasks from the DB
// TODO: Pagination
func (s *API) GetTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var tasks []models.Task
	s.db.Find(&tasks)
	s.ReturnData(w, "success", tasks)
}

// SubmitTask registers a task to be sent to the Eval handler
func (s *API) SubmitTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	spew.Dump(r.Form)
}
