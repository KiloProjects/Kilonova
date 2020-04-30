package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// RegisterProblemRoutes registers the problem routes at /api/problem
func (s *API) RegisterProblemRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/getAll", s.GetAllProblems)
	r.Get("/getByID", s.GetProblemByID)
	r.With(s.MustBeAuthed).Post("/create", s.InitProblem)
	r.With(s.MustBeAuthed).Route("/update/{id}", func(r chi.Router) {
		// TODO: Make sure it is the author or admin who does the change
		r = r.With(s.ValidateProblemID)
		r.Post("/title", func(w http.ResponseWriter, r *http.Request) {
			val := r.FormValue("title")
			s.db.Model(&models.Problem{}).Where("id = ?", s.getContextValue(r, "pbID")).UpdateColumn("name", val)
		})
		r.Post("/addTest", func(w http.ResponseWriter, r *http.Request) {
			// TODO: Change to files, or make both options interchangeable
			score, err := strconv.Atoi(r.FormValue("score"))
			if err != nil {
				s.ErrorData(w, "Score not integer", http.StatusBadRequest)
				return
			}
			var test models.Test
			test.ProblemID = s.getContextValue(r, "pbID").(int)
			test.Score = score
			s.db.Save(&test)
			fmt.Println(test.ID)
			s.manager.SaveTest(
				s.getContextValue(r, "pbID").(int),
				int(test.ID),
				[]byte(r.FormValue("input")),
				[]byte(r.FormValue("output")),
			)
			s.ReturnData(w, "success", "Test created successfully")
		})
		r.Post("/description", func(w http.ResponseWriter, r *http.Request) {
			val := r.FormValue("description")
			s.db.Model(&models.Problem{}).Where("id = ?", s.getContextValue(r, "pbID")).UpdateColumn("text", val)
		})
		r.Post("/updateTest", func(w http.ResponseWriter, r *http.Request) {
			// TODO
		})
	})
	return r
}

// InitProblem assigns an ID for the problem
func (s *API) InitProblem(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		s.ErrorData(w, "Title not provided", http.StatusBadRequest)
		return
	}
	var tmp models.Problem
	s.db.First(&tmp, "lower(name) = lower(?)", title)
	if tmp.ID != 0 {
		s.ErrorData(w, "Title already exists in DB", http.StatusBadRequest)
		return
	}
	fmt.Printf("%v\n", r.Context().Value(models.KNContextType("user")).(models.User))
	s.db.Create(&models.Problem{Name: title, Author: r.Context().Value(models.KNContextType("user")).(models.User)})
	s.db.First(&tmp, "lower(name) = lower(?)", title)
	s.ReturnData(w, "success", tmp.ID)
}

// GetAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) GetAllProblems(w http.ResponseWriter, r *http.Request) {
	var problems []models.Problem
	s.db.Preload("Tests").Preload("Author").Find(&problems)
	s.ReturnData(w, "success", problems)
}

// GetProblemByID returns a problem from the DB specified by ID
func (s *API) GetProblemByID(w http.ResponseWriter, r *http.Request) {
	var problem models.Problem
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		fmt.Fprintln(w, "Invalid ID")
	}
	s.db.Where("id = ?", id).Preload("Tests").Preload("Author").First(&problem)
	s.ReturnData(w, "success", problem)
}

// ValidateProblemID pre-emptively returns if there isnt a valid problem ID in the URL params
func (s *API) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			s.ErrorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), models.KNContextType("pbID"), problemID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}
