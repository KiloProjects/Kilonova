package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// RegisterAdminRoutes registers all of the admin router at /api/admin
func (s *API) RegisterAdminRoutes() chi.Router {
	r := chi.NewRouter()
	// r.Use(s.mustBeAdmin)

	r.Get("/makeAdmin", s.MakeAdmin)
	r.Post("/newMOTD", s.NewMOTD)
	r.Post("/createProblem", s.CreateProblem)
	r.Get("/getUsers", func(w http.ResponseWriter, r *http.Request) {
		var users []models.User
		s.db.Find(&users)
		json.NewEncoder(w).Encode(users)
	})
	r.HandleFunc("/dropAll", func(w http.ResponseWriter, r *http.Request) {
		s.db.DropTable(models.User{}, models.EvalTest{}, models.MOTD{}, models.Problem{}, models.Task{}, models.Test{})
		fmt.Println("restarting")
		os.Exit(0)
	})
	return r
}

// MakeAdmin updates an account to make it an admin
func (s *API) MakeAdmin(w http.ResponseWriter, r *http.Request) {

}

// CreateProblem is a stub
func (s *API) CreateProblem(w http.ResponseWriter, r *http.Request) {
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
}

// NewMOTD adds a new MOTD to the DB
func (s *API) NewMOTD(w http.ResponseWriter, r *http.Request) {
	newMotd := r.FormValue("data")
	s.db.Create(&models.MOTD{Motd: newMotd})
}

// HandleTests fills a models.Test array with the tests from an archive
func HandleTests(r *http.Request) ([]models.Test, error) {
	return nil, nil
}
