package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/models"
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
		r.Use(s.ValidateProblemID)
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
			test.ProblemID = s.pbIDFromReq(r)
			test.Score = score
			s.db.Save(&test)
			s.manager.SaveTest(
				s.pbIDFromReq(r),
				test.ID,
				[]byte(r.FormValue("input")),
				[]byte(r.FormValue("output")),
			)
			s.ReturnData(w, "success", test.ID)
		})
		r.Post("/description", func(w http.ResponseWriter, r *http.Request) {
			val := r.FormValue("description")
			s.db.Model(&models.Problem{}).Where("id = ?", s.getContextValue(r, "pbID")).UpdateColumn("text", val)
		})
		r.Post("/updateTest", func(w http.ResponseWriter, r *http.Request) {
			// TODO
		})
		r.Post("/removeTests", func(w http.ResponseWriter, r *http.Request) {
			var problem models.Problem
			s.db.Preload("Tests").First(&problem, s.pbIDFromReq(r))
			if err := s.db.Unscoped().Delete(&problem.Tests).Error; err != nil {
				s.ErrorData(w, err.Error(), http.StatusInternalServerError)
			}
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
	var cnt int
	s.db.Model(&models.Problem{}).Where("lower(name) = lower(?)", title).Count(&cnt)
	if cnt > 0 {
		s.ErrorData(w, "Title already exists in DB", http.StatusBadRequest)
		return
	}
	var problem models.Problem
	problem.Name = title
	problem.User = s.UserFromContext(r)
	problem.UserID = s.UserFromContext(r).ID
	fmt.Println("FUCKING EMAIL:", s.UserFromContext(r).Email)
	s.db.Create(&problem)
	s.ReturnData(w, "success", problem.ID)
}

// GetAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) GetAllProblems(w http.ResponseWriter, r *http.Request) {
	var problems []models.Problem
	fmt.Println("start")
	s.db.Preload("Tests").Find(&problems)
	fmt.Println("end")
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
	s.db.Where("id = ?", id).Set("gorm:auto_preload", true).First(&problem)
	s.ReturnData(w, "success", problem)
}

// ValidateProblemID pre-emptively returns if there isnt a valid problem ID in the URL params
func (s *API) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			s.ErrorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), models.KNContextType("pbID"), uint(problemID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func (s API) pbIDFromReq(r *http.Request) uint {
	return s.getContextValue(r, "pbID").(uint)
}
